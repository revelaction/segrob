CREATE TABLE IF NOT EXISTS docs (
    id      INTEGER PRIMARY KEY,
    title   TEXT NOT NULL UNIQUE,
    labels  TEXT
);

CREATE TABLE IF NOT EXISTS sentences (
    rowid       INTEGER PRIMARY KEY,
    doc_id      INTEGER NOT NULL,
    sentence_id INTEGER NOT NULL, -- Sequential index (0, 1, ...)
    data        TEXT NOT NULL,    -- Full JSON content (tokens)
    FOREIGN KEY (doc_id) REFERENCES docs(id)
);

CREATE TABLE IF NOT EXISTS sentence_lemmas (
    lemma           TEXT NOT NULL,
    sentence_rowid  INTEGER NOT NULL,
    FOREIGN KEY (sentence_rowid) REFERENCES sentences(rowid)
);

CREATE TABLE IF NOT EXISTS sentence_labels (
    label           TEXT NOT NULL,
    sentence_rowid  INTEGER NOT NULL,
    FOREIGN KEY (sentence_rowid) REFERENCES sentences(rowid)
);

CREATE INDEX IF NOT EXISTS idx_lemma_rowid ON sentence_lemmas(lemma, sentence_rowid);
CREATE INDEX IF NOT EXISTS idx_label_rowid ON sentence_labels(label, sentence_rowid);
CREATE INDEX IF NOT EXISTS idx_sentences_doc_id ON sentences(doc_id);

-- Reverse indexes (for EXISTS probes in FindCandidates)
CREATE INDEX IF NOT EXISTS idx_rowid_lemma ON sentence_lemmas(sentence_rowid, lemma);
CREATE INDEX IF NOT EXISTS idx_rowid_label ON sentence_labels(sentence_rowid, label);
