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
export SEGROB_TOPIC_DB="segrob.db"       # Path to topics (usually the same as live)
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
# Ingest topics from a directory of JSON files
segrob corpus ingest-topic /path/to/topics

# List and show topics in the corpus
segrob corpus ls-topic
segrob corpus show-topic <topic_name>

# Edit topics interactively
segrob corpus edit

# Dump a topic to JSON for external backup/editing
segrob corpus dump-topic <topic_name> > topic.json
```

### 4.2. Publishing Topics to Live

Topics must be published to the live database to be used by `live find` or `live query`:

```bash
segrob corpus publish-topic
```

---

## 5. Backup Workflow

Create a gzipped backup of the corpus staging database for safety or portability.

```bash
# Basic backup (excludes NLP data to keep file size small)
segrob corpus backup -o backups/corpus_lite.gz

# Full backup (includes NLP data)
segrob corpus backup -o backups/corpus_full.gz --with-nlp
```

The output file will have a timestamp automatically appended to the provided name.

