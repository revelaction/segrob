-- Time format for reference: strftime('%Y-%m-%dT%H:%M:%SZ', 'now')

CREATE TABLE IF NOT EXISTS corpus (
    id                TEXT NOT NULL PRIMARY KEY,
    labels            TEXT NOT NULL DEFAULT '',
    epub              TEXT NOT NULL DEFAULT '',
    txt               TEXT NOT NULL DEFAULT '',
    txt_hash          TEXT NOT NULL DEFAULT '',
    txt_created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    txt_edited        BOOL NOT NULL DEFAULT false,
    txt_edited_at     TEXT NOT NULL DEFAULT '',
    txt_editor        TEXT NOT NULL DEFAULT '',
    txt_edit_notes    TEXT NOT NULL DEFAULT '',
    nlp               TEXT NOT NULL DEFAULT '',
    nlp_created_at    TEXT NOT NULL DEFAULT '',
    nlp_reviewed      BOOL NOT NULL DEFAULT false,
    nlp_reviewed_at   TEXT NOT NULL DEFAULT '',
    nlp_reviewer      TEXT NOT NULL DEFAULT '',
    nlp_review_notes  TEXT NOT NULL DEFAULT '',
    deleted_at        TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
