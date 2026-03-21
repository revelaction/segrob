-- Time format for reference: strftime('%Y-%m-%dT%H:%M:%SZ', 'now')

CREATE TABLE IF NOT EXISTS corpus (
    id                TEXT NOT NULL PRIMARY KEY,
    labels            TEXT NOT NULL DEFAULT '',
    epub              TEXT NOT NULL DEFAULT '',
    txt               TEXT NOT NULL DEFAULT '',
    txt_hash          TEXT NOT NULL DEFAULT '',
    txt_created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    txt_edit          BOOL NOT NULL DEFAULT false,
    txt_edit_at       TEXT NOT NULL DEFAULT '',
    txt_edit_by       TEXT NOT NULL DEFAULT '',
    txt_edit_notes    TEXT NOT NULL DEFAULT '',
    txt_ack           BOOL NOT NULL DEFAULT false,
    txt_ack_at        TEXT NOT NULL DEFAULT '',
    txt_ack_by        TEXT NOT NULL DEFAULT '',
    nlp               TEXT NOT NULL DEFAULT '',
    nlp_created_at    TEXT NOT NULL DEFAULT '',
    nlp_ack           BOOL NOT NULL DEFAULT false,
    nlp_ack_at        TEXT NOT NULL DEFAULT '',
    nlp_ack_by        TEXT NOT NULL DEFAULT '',
    deleted_at        TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS corpus_topics (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id   TEXT,
    name      TEXT NOT NULL,
    exprs     TEXT NOT NULL,
    created   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(user_id, name)
);
