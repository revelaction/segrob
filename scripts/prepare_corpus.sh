#!/bin/bash
# scripts/prepare_corpus.sh
# Preparation of texts: EPUB -> TXT and normalization for NLP tokenization.

set -e

# --- Default configuration ---
RUN_CONVERT=false
RUN_CLEAN=false
RUN_DOT=false
FORCE_ALL=true

usage() {
    echo "Usage: $0 [OPTIONS] <INPUT_FILE> <OUTPUT_FILE>"
    echo ""
    echo "Options:"
    echo "  -c, --convert   Run Pandoc conversion (EPUB -> TXT)"
    echo "  -l, --clean     Remove empty lines"
    echo "  -d, --dot       Append a period to lines not ending in one"
    echo "  -a, --all       Run all steps (default if no specific step is selected)"
    echo "  -h, --help      Show this help message"
    exit 1
}

# --- Parse arguments ---
# Using a loop to handle both short and long options
while [[ $# -gt 0 ]]; do
    case "$1" in
        -c|--convert)
            RUN_CONVERT=true
            FORCE_ALL=false
            shift
            ;;
        -l|--clean)
            RUN_CLEAN=true
            FORCE_ALL=false
            shift
            ;;
        -d|--dot)
            RUN_DOT=true
            FORCE_ALL=false
            shift
            ;;
        -a|--all)
            FORCE_ALL=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        -*)
            echo "Unknown option: $1"
            usage
            ;;
        *)
            # Stop parsing options if we hit something that doesn't start with -
            break
            ;;
    esac
done

# Check if we have exactly two positional arguments left
if [[ $# -ne 2 ]]; then
    echo "Error: Missing input or output file."
    usage
fi

INPUT_FILE="$1"
OUTPUT_FILE="$2"

# If no specific flags were set, default to running all steps
if [ "$FORCE_ALL" = true ]; then
    RUN_CONVERT=true
    RUN_CLEAN=true
    RUN_DOT=true
fi

# --- Validation ---

# 1. Input file exists
if [[ ! -f "$INPUT_FILE" ]]; then
    echo "Error: Input file '$INPUT_FILE' not found."
    exit 1
fi

# 2. Pandoc exists if conversion is needed
if [ "$RUN_CONVERT" = true ]; then
    if ! command -v pandoc &> /dev/null; then
        echo "Error: pandoc is not installed. Required for conversion."
        exit 1
    fi
fi

# 3. Ensure output directory exists
mkdir -p "$(dirname "$OUTPUT_FILE")"

# --- Execution ---

# Step 1: Conversion
if [ "$RUN_CONVERT" = true ]; then
    echo "[1/3] Converting $INPUT_FILE to $OUTPUT_FILE..."
    pandoc "$INPUT_FILE" -t plain --wrap=preserve -o "$OUTPUT_FILE"
else
    # If we are not converting, the input file for the next steps is the output file
    # We might need to copy it if it doesn't exist yet to avoid editing the source
    if [[ ! -f "$OUTPUT_FILE" ]]; then
        echo "Copying $INPUT_FILE to $OUTPUT_FILE for processing..."
        cp "$INPUT_FILE" "$OUTPUT_FILE"
    fi
fi

# Step 2: Cleanup
if [ "$RUN_CLEAN" = true ]; then
    echo "[2/3] Removing empty lines from $OUTPUT_FILE..."
    sed -i '/^$/d' "$OUTPUT_FILE"
fi

# Step 3: Punctuation Normalization
if [ "$RUN_DOT" = true ]; then
    echo "[3/3] Normalizing line endings in $OUTPUT_FILE..."
    # Appends a '.' to lines that do not end with a literal '.'
    sed -i '/[^.]$/s/$/./' "$OUTPUT_FILE"
fi

echo "Done. Prepared text is available at: $OUTPUT_FILE"
