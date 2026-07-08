# Segrob Workflow

This document describes the standard sequence of operations to manage content in the `segrob` system, from initial setup to publication in the live production database.

The system uses a two-stage architecture:
1. **Corpus Staging**: Where documents are ingested, text and NLP are curated, and topics are defined.
2. **Live Production**: The optimized database used by query and find commands.

---

## 0. Environment Configuration

To reduce noise and avoid repeating paths in every command, set the following environment variables in your shell (e.g., in `.bashrc` or `.zshrc`):

```bash
export SEGROB_CORPUS_DB="corpus.db"      # Path to the staging database
export SEGROB_LIVE_DB="segrob.db"        # Path to the production database
export SEGROB_NLP_SCRIPT="/path/to/nlp.py" # Path to your NLP processing script
```

When these are set, the `--db`, `--from`, `--to`, and `--nlp-script` flags become optional. The examples below assume these variables are configured.

---

## 1. Initial Setup

Initialize the corpus staging database:

```bash
segrob corpus init
```

This creates the `corpus` and `corpus_topics` tables.

---

## 2. Document Lifecycle

### 2.1. Ingestion

Scan a directory of EPUB files to extract metadata and full text:

```bash
segrob corpus ingest-meta /path/to/epubs
```

- Each book is assigned a unique ID based on its content hash.
- Metadata (labels) and cleaned text are stored.
- `TxtAck` and `NlpAck` are initially false.

### 2.2. Text Curation (TxtAck)

Review the extracted text for quality:

```bash
# List documents to check status
segrob corpus ls

# Dump raw text content of a specific document
segrob corpus dump-txt <doc_id>

# (Optional) Update text if errors are found
segrob corpus push-txt --by "curator" <doc_id> edited_text.txt

# Acknowledge the text quality
segrob corpus ack --by "curator" <doc_id>
```

`TxtAck = true` is required before running NLP processing.

### 2.3. NLP Processing (NlpAck)

Run the NLP pipeline to tokenize and lemmatize the text:

```bash
# Process with NLP (requires TxtAck)
segrob corpus ingest-nlp <doc_id>

# Review rendered NLP results (sentences and lemmas)
segrob corpus show <doc_id>

# Acknowledge the NLP results
segrob corpus ack --nlp --by "curator" <doc_id>
```

`NlpAck = true` is required before a document can be published to live.

### 2.4. Publication

Move the curated document to the live production database:

```bash
# Publish a single document (requires TxtAck and NlpAck)
segrob corpus publish <doc_id>

# Batch publish all acknowledged documents
segrob corpus publish
```

---

## 3. Labels Workflow

Labels are metadata (e.g., `creator:borges`) used for filtering.

### 3.1. Managing Labels in Corpus

```bash
# List all labels
segrob corpus ls-label

# Add or remove labels for a document
segrob corpus set-label <doc_id> "genre:ficción"
segrob corpus set-label <doc_id> --delete "old:label"
```

### 3.2. Synchronizing Labels to Live

If you update labels in the corpus after the document has been published, you can push them without republishing the entire text/NLP:

```bash
segrob corpus publish-label <id>
```

---

## 4. Topics Workflow

Topics are sets of expressions used for semantic search.

### 4.1. Managing Topics in Corpus

```bash
# Ingest all topics from a single JSON file.
# Each topic in the file fully replaces any existing row with the same name.
# Topics already in the DB but absent from the file are left untouched.
# All corpus topics use user_id="" (global scope).
segrob corpus ingest-topic /path/to/topics.json

# List and show topics in the corpus
segrob corpus ls-topic
segrob corpus show-topic <topic_name>

# Edit topics interactively
segrob corpus edit

# Dump all topics to JSON for external backup/editing
segrob corpus dump-topic > topics.json
```

### 4.2. Publishing Topics to Live

Topics must be published to the live database to be used by `live find` or `live query`:

```bash
# Copy all topics from corpus to live.
# Each topic fully replaces any existing row with the same name in the live DB.
# Idempotent: safe to re-run; topics in live but absent from corpus are left untouched.
# Published topics retain user_id="" from the corpus.
segrob corpus publish-topic
```

### 4.3. Live Topic Operations

Once topics are published, the live database offers inspection, search, and management commands:

```bash
# List all topic names in the live database
segrob live ls-topic

# Show expressions for a specific topic
segrob live show-topic <topic_name>

# Show topics associated with a specific sentence
segrob live find-topics <doc_id> <sentence_id>

# Enter interactive query mode (searches both docs and topics)
segrob live query

# Dump all live topics as a single JSON file (optionally filtered by user)
segrob live dump-topic
segrob live dump-topic -u <user_id> > topics.json

# Remove a topic from the live database
segrob live unpublish-topic <topic_name>
```

All live commands accept `--db` to point to the SQLite file, defaulting to `SEGROB_LIVE_DB`.

## 5. Backup Workflow

The backup command produces a gzipped SQLite file containing the two staging tables: `corpus` and `corpus_topics`.

By default the heavy `nlp` column (raw NLP JSON payload) is **excluded** to keep backups compact. Use `--with-nlp` to include it.

```bash
# Basic backup (corpus + corpus_topics, nlp excluded)
# Auto-generates a timestamped file like corpus.db-20260330T161922Z.gz in the current directory
segrob corpus backup

# Full backup (corpus + corpus_topics + nlp data)
segrob corpus backup --with-nlp

# Backup to an explicit, exact path (no timestamp is appended)
segrob corpus backup -o backups/corpus_lite.gz
```

---

## 6. Topics Backup

Topic data is lightweight (name and expressions only, no heavy text or NLP payloads). Segrob can dump all topics to a pretty-printed JSON file from either the corpus staging or the live production database. Both commands write to stdout — redirect to a file to save the backup.

The output is deterministically ordered (topics sorted alphabetically by name, expressions deduplicated), producing stable diffs — ideal for tracking in git.

### 6.1. From Corpus

```bash
# Dump all corpus topics as indented JSON
segrob corpus dump-topic > topics_backup.json
```

### 6.2. From Live (with optional user filter)

The live version supports filtering by user ID, so you can export only one user's topics:

```bash
# Dump all live topics
segrob live dump-topic > topics_backup.json

# Dump topics for a specific user only
segrob live dump-topic -u <user_id> > topics_backup.json
```

### 6.3. Restoring Topics

To restore global topics from a dump file (created with `corpus dump-topic` or `live dump-topic`):

```bash
# 1. Ingest the dump into the corpus staging DB
#    Each topic in the file fully replaces any existing row with the same name.
segrob corpus ingest-topic topics.json

# 2. Publish from corpus to the live DB
#    Idempotent: safe to re-run.
segrob corpus publish-topic
```

This is the canonical path for global topics — both commands operate on `user_id=""`.

