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

-- Reverse indexes (for EXISTS probes in FindCandidates)
CREATE INDEX IF NOT EXISTS idx_rowid_lemma ON sentence_lemmas(sentence_rowid, lemma);
CREATE INDEX IF NOT EXISTS idx_rowid_label ON sentence_labels(sentence_rowid, label);
