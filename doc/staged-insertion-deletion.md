# Concurrent Read-Write Feasibility on Live SQLite Database

## Current State Summary

The SQLite storage layer uses [zombiezen/go-sqlite](https://github.com/zombiezen/go-sqlite) with a connection pool (`sqlitex.Pool`).

| Aspect | Current State |
|---|---|
| **WAL mode** | ✅ Enabled via `sqlite.OpenWAL` flag in [db.go](file:///home/lipo/src/public/revelaction/segrob/storage/sqlite/zombiezen/db.go#L17) |
| **Pool size** | `runtime.NumCPU()` connections |
| **Write method** | Uses `sqlitex.Save(conn)` (savepoint-based transactions) |
| **Delete method** | ❌ Does not exist |
| **Foreign keys** | Declared but **no** `ON DELETE CASCADE` |
| **busy_timeout** | ❌ Not configured |
| **Preloader** | Not implemented in SQLite store |

---

## Can You Add Documents While Hundreds of Reads Are Happening?

**Yes, largely.** WAL mode is the critical enabler here. With WAL:

- **Readers never block writers**, and **writers never block readers**. Each reader sees a consistent snapshot from the moment its transaction started.
- A single writer can proceed concurrently with any number of readers.

The current [Write](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#54-56) method in [doc_store.go](file:///home/lipo/src/public/revelaction/segrob/storage/sqlite/zombiezen/doc_store.go#L232-L296) already does the right thing:
1. Takes a connection from the pool (`pool.Take`)
2. Opens a savepoint transaction (`sqlitex.Save`)
3. Inserts into `docs`, `sentences`, `sentence_lemmas`, `sentence_labels`
4. Commits on success

This means **adding a document on a live database with hundreds of reads will work correctly today**, with one caveat: see "Writer-Writer Contention" below.

---

## Can You Remove Documents?

**Not yet.** There is no `Delete` method in the codebase. See the **Staged Insertion and Deletion** section below for the recommended approach.

---

## Issues and Improvements for a Web/Library Scenario

### 1. Writer-Writer Contention (⚠️ most important)

SQLite allows only **one writer at a time** (even in WAL mode). If two concurrent requests try to [Write](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#54-56) or `Delete`, one will get `SQLITE_BUSY`.

**Current risk:** The code has **no `busy_timeout`** set. If a write attempt finds the database locked by another writer, it will fail immediately with an error instead of retrying.

**Fix:**

```go
// After pool.Take, set busy timeout on the connection
conn.SetBusyTimeout(5 * time.Second)  // or equivalent PRAGMA
```

Or set it via the connection URI:

```go
initString := fmt.Sprintf("file:%s?_busy_timeout=5000", dbPath)
```

This makes SQLite wait up to N milliseconds for a write lock before returning `SQLITE_BUSY`, which is essential for a multi-user web scenario.

### 2. Connection Pool Saturation

Pool size is `runtime.NumCPU()`. In a web scenario with hundreds of concurrent reads:
- Each `pool.Take` blocks if all connections are in use
- A long-running write transaction holds a connection for the duration of the import

**Consideration:** For a web/library scenario, you might want:
- A **larger pool size** (e.g., `NumCPU * 4` or a configurable value) to absorb read spikes
- Or a **separate write connection** outside the read pool (a common SQLite pattern: 1 write conn + N read conns)

### 3. Long Write Transactions → Staged Insertion/Deletion

The current [Write](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#54-56) method inserts a doc with **all** its sentences, lemmas, and labels in a single transaction. For a large document this holds the write lock for a significant time, blocking all other writes.

**Solution:** Use staged, partial writes and deletes. See the dedicated section below.

### 4. No `PRAGMA foreign_keys = ON`

FK enforcement is **off by default** in SQLite. The [docs.sql](file:///home/lipo/src/public/revelaction/segrob/storage/sqlite/zombiezen/sql/docs.sql) schema declares FK constraints, but they are decorative without this PRAGMA.

**Fix:** Set `PRAGMA foreign_keys = ON` on each new connection via an init function on pool creation.

---

## Staged Insertion and Deletion

The key insight: **the lookup tables** (`sentence_lemmas`, `sentence_labels`) are what make data **findable** by [FindCandidates](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#42-46). Controlling when data appears/disappears in these tables gives explicit control over visibility, while keeping write lock duration minimal.

### Current table dependencies

```
docs (metadata)  →  sentences (bulk data)  →  sentence_lemmas (lookup)
                                            →  sentence_labels (lookup)
```

### Staged Insertion

With `PRAGMA foreign_keys = ON`, the `docs` row **must** exist before any `sentences` row can reference it. This constrains the phase ordering:

**Phase 1 — Doc metadata (tiny transaction, FK parent)**

Insert the doc row first. This is a single-row insert, so the write lock is held for microseconds:

```
BEGIN
INSERT INTO docs (title, labels) VALUES (?, ?)
COMMIT
→ returns doc_id
```

After this, the doc appears in [List](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#34-38) results but has zero sentences — it is an empty shell. [FindCandidates](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#42-46) returns nothing for it (no lookup entries).

**Phase 2 — Bulk sentence data (batched short transactions)**

Insert `sentences` rows referencing the existing `doc_id`. Batch to keep each write lock brief:

```
for each batch of N sentences:
    BEGIN
    INSERT INTO sentences (doc_id, sentence_id, data) VALUES (doc_id, ?, ?)
    ...
    COMMIT
    → collect sentence rowids
```

After each batch commits, the sentences exist but are still **unfindable** — no lookup table rows reference them.

**Phase 3 — Lookup link (the visibility switch)**

Insert `sentence_lemmas` and `sentence_labels` rows. This is the moment the document's sentences become findable by [FindCandidates](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#42-46):

```
BEGIN
for each sentence_rowid:
    INSERT INTO sentence_lemmas (lemma, sentence_rowid) ...
    INSERT INTO sentence_labels (label, sentence_rowid) ...
COMMIT
```

> [!IMPORTANT]
> Phase 3 is the **atomic visibility switch**. Before it commits, no [FindCandidates](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#42-46) query can find the new sentences. After it commits, all of them appear at once.

If Phase 3 is still too large for one transaction, it can be batched per-sentence, with the tradeoff that the document's sentences appear incrementally in search results.

#### Failure Recovery for Staged Insertion

Each phase is independently committable, so a crash between phases leaves the database in a **recoverable intermediate state**. The question is: how does the insertion job resume?

**If Phase 1 fails (doc insert):**

Nothing was written. Simply retry the entire insertion from Phase 1. No cleanup needed.

**If Phase 2 fails mid-batch (some sentences inserted, some not):**

The doc exists (`doc_id` is known). Some sentence batches committed, some didn't. To resume:

```go
// Query which sentence_ids already exist for this doc
existing, _ := query("SELECT sentence_id FROM sentences WHERE doc_id = ?", docID)

// Insert only the missing ones
for _, sentence := range doc.Sentences {
    if !existing.Contains(sentence.Id) {
        insert(sentence)
    }
}
```

The `sentence_id` column (sequential index 0, 1, 2...) acts as the natural idempotency key. Each resume attempt skips already-inserted rows.

> [!NOTE]
> This works because [(doc_id, sentence_id)](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#13-15) is a unique pair by construction — sentence_id is the sequential index within the document.

**If Phase 3 fails mid-batch (some lookup rows inserted, some not):**

Some `sentence_lemmas`/`sentence_labels` rows exist, some don't. The document is partially findable. To resume:

```go
// Query which sentence_rowids already have lookup entries
linked, _ := query("SELECT DISTINCT sentence_rowid FROM sentence_lemmas WHERE sentence_rowid IN (SELECT rowid FROM sentences WHERE doc_id = ?)", docID)

// Insert lookup rows only for unlinked sentences
for _, rowid := range allSentenceRowIDs {
    if !linked.Contains(rowid) {
        insertLemmas(rowid)
        insertLabels(rowid)
    }
}
```

Each sentence is either fully linked (lemmas + labels inserted) or not linked at all, because the per-sentence lookup inserts happen within the same batch transaction.

**General resume logic:**

A single resume function can inspect the database state and determine which phase to continue from:

```
1. Does docs row exist for this title?
   No  → start at Phase 1
   Yes → get doc_id

2. Are all sentences inserted?
   count(sentences WHERE doc_id) < expected
   → resume Phase 2 (skip existing sentence_ids)

3. Are all lookup rows inserted?
   count(DISTINCT sentence_rowid in sentence_lemmas WHERE doc_id) < count(sentences WHERE doc_id)
   → resume Phase 3 (skip already-linked rowids)

4. All counts match → insertion complete
```

> [!TIP]
> The resume logic relies on the existing schema — no new columns or tables are needed. The `title` uniqueness constraint in `docs` and the sequential `sentence_id` within each doc naturally serve as idempotency keys.

### Staged Deletion

Deletion is the reverse: **unlink first** (make unfindable), **clean up later**.

**Phase 1 — Atomic lookup unlink (the visibility switch)**

Remove all lookup table rows for the document's sentences in one transaction:

```
BEGIN
DELETE FROM sentence_lemmas WHERE sentence_rowid IN
    (SELECT rowid FROM sentences WHERE doc_id = ?)
DELETE FROM sentence_labels WHERE sentence_rowid IN
    (SELECT rowid FROM sentences WHERE doc_id = ?)
COMMIT
```

After this commits, the document's sentences are **instantly unfindable** by [FindCandidates](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#42-46) — no lookup rows point to them.

> [!IMPORTANT]
> Phase 1 is the only time-critical step. Once the lookup rows are gone, the document is effectively deleted from the user's perspective.

**Phase 2 — Data cleanup (can be deferred)**

Delete the bulk data. This can happen immediately or be **scheduled as a background job**:

```
DELETE FROM sentences WHERE doc_id = ?
DELETE FROM docs WHERE id = ?
```

These rows are orphaned (no lookup references them), so they don't affect query results. Deferring them avoids holding the write lock during cleanup of potentially large `sentences` data.

> [!TIP]
> A simple cleanup strategy: a periodic goroutine that runs `DELETE FROM sentences WHERE doc_id NOT IN (SELECT id FROM docs WHERE ...)` or processes a deletion queue table.

### Why not ON DELETE CASCADE

`ON DELETE CASCADE` delegates the deletion order to SQLite. You lose control over:
- **Which tables are cleaned first** (lookup vs. data)
- **How long the write lock is held** (SQLite cascades everything in one transaction)
- **Whether cleanup can be deferred** (it can't — it's all-or-nothing)

Explicit staged deletion gives full control over visibility semantics and write lock duration.

---

## Summary

| Feature | Status | Path Forward |
|---|---|---|
| Adding docs during reads | ✅ Works (WAL) | Refactor to staged insertion |
| Removing docs | ❌ No method | Staged deletion (unlink lookup → defer cleanup) |
| Concurrent writes | ⚠️ No busy_timeout | Set PRAGMA or URI param |
| Pool sizing | ⚠️ May be small | Increase or split read/write pools |
| Write lock duration | ⚠️ Long transactions | Staged insertion/deletion solves this |

### Recommended priority

1. **`busy_timeout`** — essential for any concurrent write scenario
2. **Staged insertion** — refactor [Write](file:///home/lipo/src/public/revelaction/segrob/storage/storage.go#54-56) into phases (bulk → metadata → lookup link)
3. **Staged deletion** — implement `Delete` as (lookup unlink → deferred cleanup)
4. **Pool size tuning** — for web/library with many concurrent readers
5. **`PRAGMA foreign_keys = ON`** — integrity safeguard
