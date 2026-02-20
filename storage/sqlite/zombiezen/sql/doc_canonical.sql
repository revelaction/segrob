CREATE TABLE IF NOT EXISTS docs (
    id      INTEGER PRIMARY KEY,
    title   TEXT NOT NULL UNIQUE,
);

CREATE TABLE IF NOT EXISTS sentences (
    rowid       INTEGER PRIMARY KEY,
    doc_id      INTEGER NOT NULL,
    sentence_id INTEGER NOT NULL, -- Sequential index (0, 1, ...)
    data        TEXT NOT NULL,    -- Full JSON content (tokens)
    FOREIGN KEY (doc_id) REFERENCES docs(id)
);

CREATE INDEX IF NOT EXISTS idx_sentences_doc_id ON sentences(doc_id);

CREATE TABLE IF NOT EXISTS labels (
    doc_id  INTEGER NOT NULL,
    label   TEXT NOT NULL,
    FOREIGN KEY (doc_id) REFERENCES docs(id),
    UNIQUE(doc_id, label)
);

CREATE INDEX IF NOT EXISTS idx_labels_doc_id ON labels(doc_id);
