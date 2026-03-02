-- Time format for reference: strftime('%Y-%m-%dT%H:%M:%SZ', 'now')

CREATE TABLE IF NOT EXISTS corpus (
    id               TEXT PRIMARY KEY,
    labels           TEXT,
    epub             TEXT,
    txt              TEXT,
    txt_hash         TEXT,
    txt_reviewed     BOOL DEFAULT false,
    txt_reviewed_at  TEXT NOT NULL DEFAULT '',
    txt_reviewer     TEXT,
    txt_review_notes TEXT,
    deleted          BOOL DEFAULT false,
    deleted_at       TEXT NOT NULL DEFAULT ''
);
