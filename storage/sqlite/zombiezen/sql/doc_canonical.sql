CREATE TABLE IF NOT EXISTS docs (
    id        TEXT PRIMARY KEY,
    source    TEXT NOT NULL,
    label_ids TEXT NOT NULL DEFAULT ''
);

-- Label dictionary: one row per unique label name.
-- Small table (~100-200 rows for the full vocabulary).
CREATE TABLE IF NOT EXISTS labels (
    id    INTEGER PRIMARY KEY,
    name  TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS sentences (
    rowid       INTEGER PRIMARY KEY,
    doc_id      TEXT NOT NULL,
    sentence_id INTEGER NOT NULL, -- Sequential index (0, 1, ...)
    data        TEXT NOT NULL,    -- Full JSON content (tokens)
    FOREIGN KEY (doc_id) REFERENCES docs(id)
);

CREATE INDEX IF NOT EXISTS idx_sentences_doc_id ON sentences(doc_id);
