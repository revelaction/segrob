#!/usr/bin/env python3
# coding: utf-8
"""
View raw metadata from EPUB files.

Reads the raw OPF package file from EPUB archives and extracts:
1. The EPUB version from the <package> tag.
2. The raw <metadata> block containing Dublin Core elements and attributes.
"""

import argparse
import sys
import zipfile
import re
import xml.etree.ElementTree as ET
from pathlib import Path


def get_opf_path(zip_ref):
    """
    Locate the OPF file in the EPUB archive.
    Tries to read META-INF/container.xml first, then falls back to searching for .opf files.
    """
    try:
        with zip_ref.open('META-INF/container.xml') as f:
            root = ET.fromstring(f.read())
            # Find the rootfile element in the container namespace
            ns = {'n': 'urn:oasis:names:tc:opendocument:xmlns:container'}
            rootfile = root.find('.//n:rootfile', ns)
            if rootfile is not None:
                return rootfile.get('full-path')
    except Exception:
        pass

    # Fallback: search for first .opf file
    for name in zip_ref.namelist():
        if name.endswith('.opf'):
            return name
    return None


def main():
    parser = argparse.ArgumentParser(
        description="View raw metadata from EPUB files."
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

    # Regex to extract version from <package> tag
    # Matches: <package ... version="2.0" ...>
    version_pattern = re.compile(r'<package[^>]*\sversion\s*=\s*["\']([^"\']+)["\']', re.IGNORECASE)

    # Regex to extract the metadata block
    # Matches: <metadata ...> ... </metadata> (dotall to capture newlines)
    metadata_pattern = re.compile(r'(<metadata.*?>.*?</metadata>)', re.IGNORECASE | re.DOTALL)

    for epub_path in epub_files:
        print(f"[{epub_path.name}]")
        try:
            if not zipfile.is_zipfile(epub_path):
                print(f"  Error: Not a valid zip/epub file.")
                continue

            with zipfile.ZipFile(epub_path, 'r') as z:
                opf_path = get_opf_path(z)
                if not opf_path:
                    print("  Error: Could not locate OPF file.")
                    continue

                with z.open(opf_path) as f:
                    content = f.read().decode('utf-8', errors='replace')

                # Extract version
                v_match = version_pattern.search(content)
                version = v_match.group(1) if v_match else "Unknown"
                print(f"  version: {version}")

                # Extract metadata block
                m_match = metadata_pattern.search(content)
                if m_match:
                    # Print the raw block.
                    print(m_match.group(1).strip())
                else:
                    print("  No <metadata> block found.")

        except Exception as e:
            print(f"  Error reading {epub_path.name}: {e}")

        print("-" * 40) # Separator

if __name__ == "__main__":
    try:
        main()
    except BrokenPipeError:
        sys.stdout.flush()
        sys.exit(0)
