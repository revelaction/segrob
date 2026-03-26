CREATE TABLE IF NOT EXISTS sentence_lemmas (
    lemma           TEXT NOT NULL,
    sentence_rowid  INTEGER NOT NULL,
    FOREIGN KEY (sentence_rowid) REFERENCES sentences(rowid)
);

-- Integer label_id instead of text label for hotspot performance.
CREATE TABLE IF NOT EXISTS sentence_labels (
    label_id        INTEGER NOT NULL,
    sentence_rowid  INTEGER NOT NULL,
    FOREIGN KEY (label_id)       REFERENCES labels(id),
    FOREIGN KEY (sentence_rowid) REFERENCES sentences(rowid)
);

CREATE INDEX IF NOT EXISTS idx_lemma_rowid ON sentence_lemmas(lemma, sentence_rowid);
CREATE INDEX IF NOT EXISTS idx_label_rowid ON sentence_labels(label_id, sentence_rowid);

-- Reverse indexes (for EXISTS probes in FindCandidates)
CREATE INDEX IF NOT EXISTS idx_rowid_lemma ON sentence_lemmas(sentence_rowid, lemma);
CREATE INDEX IF NOT EXISTS idx_rowid_label ON sentence_labels(sentence_rowid, label_id);
