-- Time format for reference: strftime('%Y-%m-%dT%H:%M:%SZ', 'now')

CREATE TABLE IF NOT EXISTS corpus (
    id               TEXT PRIMARY KEY,
    labels           TEXT,
    epub             TEXT,
    txt              TEXT,
    txt_hash         TEXT,
    txt_edited       BOOL DEFAULT false,
    txt_edited_at    TEXT NOT NULL DEFAULT '',
    txt_editor       TEXT,
    txt_edit_notes   TEXT,
    nlp              TEXT,
    nlp_reviewed     BOOL DEFAULT false,
    nlp_reviewed_at  TEXT NOT NULL DEFAULT '',
    nlp_reviewer     TEXT,
    nlp_review_notes TEXT,
    deleted          BOOL DEFAULT false,
    deleted_at       TEXT NOT NULL DEFAULT ''
);
