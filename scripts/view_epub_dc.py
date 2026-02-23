#!/usr/bin/env python3
# coding: utf-8
"""
View metadata from EPUB files.

Reads standard Dublin Core metadata and EPUB version from EPUB files
and prints them to the console.
"""

import argparse
import sys
from pathlib import Path
from ebooklib import epub


def main():
    parser = argparse.ArgumentParser(
        description="View metadata from EPUB files."
    )
    parser.add_argument(
        "directory",
        type=Path,
        help="Directory to scan for *.epub files"
    )

    args = parser.parse_args()

    if not args.directory.is_dir():
        print(f"Error: {args.directory} is not a valid directory.", file=sys.stderr)
        sys.exit(1)

    epub_files = list(args.directory.glob("*.epub"))
    if not epub_files:
        print(f"No .epub files found in {args.directory}.", file=sys.stderr)
        return

    for epub_path in epub_files:
        print(f"[{epub_path.name}]")
        try:
            book = epub.read_epub(str(epub_path))
        except Exception as e:
            print(f"  Error reading {epub_path.name}: {e}")
            continue

        print(f"  version: {book.version}")

        # book.metadata is a dict: namespace -> { key: [(value, {attr}), ...] }
        # Collect all metadata that looks like Dublin Core
        for ns, data in book.metadata.items():
            if 'dc' in ns.lower():
                # Sort keys for consistent output
                for key in sorted(data.keys()):
                    entries = data[key]
                    for value, attrs in entries:
                        out = f"  {key}: {value}"
                        if attrs:
                            out += f" {attrs}"
                        print(out)

if __name__ == "__main__":
    try:
        main()
    except BrokenPipeError:
        # Standard way to handle pipe closing (e.g. when used with head)
        sys.stdout.flush()
        sys.exit(0)
