#!/usr/bin/env python3
# coding: utf-8
"""
Extract metadata from EPUB files and generate TOML sidecars.

Reads standard Dublin Core metadata (title, creator, date, language) from EPUB files
and saves a `<filename>.meta.toml` file alongside the EPUB or in a specified output directory.
Values are normalized (lowercased, spaces/dashes to underscores) while preserving accents.
"""

import argparse
import sys
from pathlib import Path
from ebooklib import epub


def normalize_value(val):
    """Lowercase and replace spaces/dashes with underscores. Preserve accents."""
    if not val:
        return ""
    val = str(val).lower()
    val = val.replace(" ", "_").replace("-", "_")
    return val


def get_dc_meta(book, namespace, name):
    """Helper to extract the first Dublin Core metadata value if present."""
    metadata = book.get_metadata(namespace, name)
    if metadata and len(metadata) > 0:
        return metadata[0][0]
    return None


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
            book = epub.read_epub(str(epub_path))
        except Exception as e:
            print(f"Error reading {epub_path.name}: {e}", file=sys.stderr)
            continue

        raw_title = get_dc_meta(book, "DC", "title")
        raw_author = get_dc_meta(book, "DC", "creator")
        raw_date = get_dc_meta(book, "DC", "date")
        raw_lang = get_dc_meta(book, "DC", "language")

        # Normalize values
        title = normalize_value(raw_title)
        author = normalize_value(raw_author)
        # Date might be long format, extract year if possible (just simplistic approach here)
        year = ""
        if raw_date:
            year = normalize_value(raw_date)[:4] if len(str(raw_date)) >= 4 else normalize_value(raw_date)
        lang = normalize_value(raw_lang)

        # Essential attributes check
        essentials = {
            "title": title,
            "author": author,
            "date": year,
            "language": lang
        }
        
        found = [k for k, v in essentials.items() if v]
        missing = [k for k, v in essentials.items() if not v]

        print(f"[{epub_path.name}] Found: {', '.join(found) if found else 'None'} | Missing: {', '.join(missing) if missing else 'None'}", file=sys.stderr)

        source = epub_path.stem
        labels = []
        if author:
            labels.append(f"author:{author}")
        if year:
            labels.append(f"year:{year}")
        if lang:
            labels.append(f"lang:{lang}")
        if title:
            labels.append(f"title:{title}")

        out_dir = args.output_dir if args.output_dir else epub_path.parent
        out_path = out_dir / f"{source}.meta.toml"

        labels_str = ", ".join(f'"{label}"' for label in labels)
        toml_string = f'source = "{source}"\nlabels = [{labels_str}]\n'

        try:
            with open(out_path, "w", encoding="utf-8") as f:
                f.write(toml_string)
        except Exception as e:
            print(f"Error writing TOML for {epub_path.name}: {e}", file=sys.stderr)

if __name__ == "__main__":
    main()
