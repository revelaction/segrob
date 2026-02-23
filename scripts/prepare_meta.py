#!/usr/bin/env python3
# coding: utf-8
"""
Extract metadata from EPUB files and generate TOML sidecars.

Reads standard Dublin Core metadata (title, creator, date, language, translator)
from EPUB files and saves a `<filename>.meta.toml` file alongside the EPUB or
in a specified output directory.
Values are normalized (lowercased, spaces/dashes to underscores) while preserving accents.
"""

import argparse
import re
import sys
from pathlib import Path
from ebooklib import epub

OPF = 'http://www.idpf.org/2007/opf'


# ---------------------------------------------------------------------------
# Low-level helper
# ---------------------------------------------------------------------------

def dc(book, name):
    """Return raw list of (value, attrs) for a DC element, or []."""
    return book.get_metadata('DC', name) or []


# ---------------------------------------------------------------------------
# Normalization
# ---------------------------------------------------------------------------

def normalize_value(val):
    """Lowercase and replace spaces/dashes with underscores. Preserve accents."""
    if not val:
        return ""
    val = str(val).lower()
    return val.replace(" ", "_").replace("-", "_")


def normalize_date(val):
    """Extract a 4-digit year from a date string, or return empty string."""
    if not val:
        return ""
    match = re.search(r'\d{4}', str(val))
    return match.group(0) if match else ""


# ---------------------------------------------------------------------------
# Extract functions
# ---------------------------------------------------------------------------

def extract_creator(book):
    items = dc(book, 'creator')
    return items[0][0] if items else ""


def extract_title(book):
    items = dc(book, 'title')
    return items[0][0] if items else ""


def extract_language(book):
    items = dc(book, 'language')
    return items[0][0] if items else ""


def extract_date_publication(book):
    """Return publication date. Prefers opf:event=publication if present, else first date."""
    for value, attrs in dc(book, 'date'):
        if attrs.get('opf:event') == 'publication':
            return value
    items = dc(book, 'date')
    return items[0][0] if items else ""


def extract_translator(book):
    """Return translator name (role=trl) from contributor or creator, or empty string."""
    # Build EPUB 3 refinements map: {element_id: {property: value}}
    refinements = {}
    for value, attrs in book.metadata.get(OPF, {}).get('meta', []):
        ref = attrs.get('refines', '')
        if ref.startswith('#'):
            refinements.setdefault(ref[1:], {})[attrs.get('property', '')] = value

    for dc_name in ('contributor', 'creator'):
        for value, attrs in dc(book, dc_name):
            # EPUB 2: opf:role attribute directly; EPUB 3: resolved from refinements
            role = attrs.get('opf:role') or refinements.get(attrs.get('id', ''), {}).get('role', '')
            if role == 'trl':
                return value
    return ""


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Extract metadata from EPUB files and generate TOML sidecars."
    )
    parser.add_argument(
        "directory",
        type=Path,
        help="Directory to scan for *.epub files"
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        help="Optional output directory for .meta.toml files (default: alongside EPUB)"
    )

    args = parser.parse_args()

    if not args.directory.is_dir():
        print(f"Error: {args.directory} is not a valid directory.", file=sys.stderr)
        sys.exit(1)

    if args.output_dir and not args.output_dir.is_dir():
        print(f"Error: Output directory {args.output_dir} is not a valid directory.", file=sys.stderr)
        sys.exit(1)

    epub_files = list(args.directory.glob("*.epub"))
    if not epub_files:
        print(f"No .epub files found in {args.directory}.", file=sys.stderr)
        return

    for epub_path in epub_files:
        try:
            book = epub.read_epub(str(epub_path), options={'ignore_ncx': True})
        except Exception as e:
            print(f"Error reading {epub_path.name}: {e}", file=sys.stderr)
            continue

        creator    = normalize_value(extract_creator(book))
        title      = normalize_value(extract_title(book))
        date       = normalize_date(extract_date_publication(book))
        language   = normalize_value(extract_language(book))
        translator = normalize_value(extract_translator(book))

        essentials = {"creator": creator, "title": title, "date": date, "language": language}
        found   = [k for k, v in essentials.items() if v]
        missing = [k for k, v in essentials.items() if not v]
        print(
            f"[{epub_path.name}] Found: {', '.join(found) or 'None'} | "
            f"Missing: {', '.join(missing) or 'None'}",
            file=sys.stderr
        )

        source = epub_path.stem
        labels = []
        if creator:    labels.append(f"creator:{creator}")
        if title:      labels.append(f"title:{title}")
        if date:       labels.append(f"date:{date}")
        if language:   labels.append(f"language:{language}")
        if translator: labels.append(f"translator:{translator}")

        out_dir  = args.output_dir if args.output_dir else epub_path.parent
        out_path = out_dir / f"{source}.meta.toml"

        labels_str  = ", ".join(f'"{label}"' for label in labels)
        toml_string = f'source = "{source}"\nlabels = [{labels_str}]\n'

        try:
            with open(out_path, "w", encoding="utf-8") as f:
                f.write(toml_string)
        except Exception as e:
            print(f"Error writing TOML for {epub_path.name}: {e}", file=sys.stderr)


if __name__ == "__main__":
    main()
