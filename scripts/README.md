# Scripts

A collection of utility scripts for debugging, metadata extraction, and NLP processing.

## Go Scripts

These scripts use the `//go:build ignore` tag to prevent conflicts during project-wide builds. They can be run directly or compiled into standalone binaries.

### `epub2txt.go`
Extracts plain text from an EPUB file using the internal `epub` package.
- **Run**: `go run scripts/epub2txt.go <file.epub>`
- **Compile**: `go build -o epub2txt scripts/epub2txt.go`

### `go_epub_meta.go`
Displays a tabular view of metadata (Title, Creator, etc.) for all EPUB files in a directory. The output format is compatible with the `live ls` command.
- **Run**: `go run scripts/go_epub_meta.go <directory>`
- **Compile**: `go build -o go_epub_meta scripts/go_epub_meta.go`

### `opf_metadata.go`
Extracts the EPUB version and the raw `<metadata>` block from an EPUB's OPF file. Useful for inspecting internal metadata structure and debugging namespace issues.
- **Run**: `go run scripts/opf_metadata.go <file.epub>`
- **Compile**: `go build -o opf_metadata scripts/opf_metadata.go`

---

## Python Scripts

### `nlp_spacy.py`
Processes a text file using **spaCy** (specifically the `es_dep_news_trf` model) to produce a JSON document in the `segrob` format.
- **Run**: `python3 scripts/nlp_spacy.py <file.txt>`

### `nlp_stanza.py`
Processes a text file using **Stanza** to produce a JSON document in the `segrob` format.
- **Run**: `python3 scripts/nlp_stanza.py <file.txt>`

### `prepare_meta.py`
Scans a directory for EPUB files and generates `.meta.toml` sidecar files containing normalized Dublin Core metadata.
- **Run**: `python3 scripts/prepare_meta.py <directory>`

### `json_diff_spacy.py`
Compares two `segrob` JSON documents and reports differences in labels, sentences, or token attributes. Useful for regression testing NLP output.
- **Run**: `python3 scripts/json_diff_spacy.py file1.json file2.json`

### `view_epub_dc.py`
Prints Dublin Core metadata for EPUB files in a directory using the `ebooklib` library.
- **Run**: `python3 scripts/view_epub_dc.py <directory>`

### `view_epub_opf.py`
Prints the raw `<metadata>` block from the OPF file within EPUB archives. Useful for debugging malformed EPUBs.
- **Run**: `python3 scripts/view_epub_opf.py <directory>`

### `debug_spacy_tokens.py` / `debug_stanza_tokens.py`
Small utilities to print internal token structures for a sample sentence. Used to calibrate MWT (Multi-Word Token) handling.
- **Run**: `python3 scripts/debug_spacy_tokens.py`

---

## Shell Scripts

### `prepare_corpus.sh`
A wrapper around `pandoc` and `sed` to convert EPUBs to text and normalize them (removing empty lines, ensuring periods at line endings) for NLP ingestion.
- **Run**: `./scripts/prepare_corpus.sh [options] <input.epub> <output.txt>`
- **Options**: `-c` (convert), `-l` (clean lines), `-d` (normalize dots), `-a` (all steps).
