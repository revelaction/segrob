#!/usr/bin/env python3
# coding: utf-8
"""
Tokenize, POS tag and lemmatize a text file using spaCy.

Outputs a JSON document in the segrob doc format to standard output.
"""
import argparse
import json
from pathlib import Path

import spacy
import es_dep_news_trf


def add_labels(cmd_labels, file_path):
    """Build labels list from filename parts, spacy version, and command-line labels."""
    labels = []

    # Optimistically consider the file name properly formatted
    # <title>-<author>-<traductor>.txt
    file_name = Path(file_path).stem
    labels.extend(file_name.split("-"))

    # Lemmatizer version
    labels.append(f"{spacy.__name__}-{spacy.__version__}")

    # Command-line labels
    if cmd_labels is not None:
        labels.extend(cmd_labels)

    return labels


def main():
    parser = argparse.ArgumentParser(
        description="Tokenize, POS tag and lemmatize a text to standard output"
    )
    parser.add_argument(
        "filename",
        type=argparse.FileType("r", encoding="UTF-8"),
        help="input text file",
        metavar="FILE",
    )
    parser.add_argument(
        "-l", "--label",
        action="append",
        help="add a label to the document metadata (can be used multiple times)",
    )

    args = parser.parse_args()

    # Read input file
    with args.filename as txt_file:
        text = txt_file.read()
        file_name = txt_file.name

    # Load spaCy model and set max_length BEFORE processing
    nlp = es_dep_news_trf.load()
    nlp.max_length = 5_000_000

    doc = nlp(text)

    # Build result structure
    res = {
        "labels": add_labels(args.label, file_name),
        "tokens": [],
    }

    token_index = 0

    for sentence in doc.sents:
        sent_tokens = []
        sentence_index = 0
        # Absolute index of the first token in the sentence
        sent_start_idx = sentence[0].i

        for token in sentence:
            morph = str(token.morph)
            tag = token.pos_ if not morph else f"{token.pos_}__{morph}"

            # Sentence-relative head index
            head_idx = token.head.i - sent_start_idx

            # Multi-word token handling:
            # spaCy signals MWT in the lemma with spaces: "combinar él"
            # We split and create separate tokens with identical idx to signal MWT.
            # it does not split "del"
            for word in token.lemma_.split(" "):
                t = {
                    "id": token_index,
                    "pos": token.pos_,
                    "tag": tag,
                    "dep": token.dep_.lower(),  # segrob requires lowercase
                    "head": head_idx,
                    "text": token.text,
                    "idx": token.idx,  # Character offset in source document
                    "index": sentence_index,  # Token position in sentence
                    "lemma": word.lower(),
                }

                sent_tokens.append(t)
                token_index += 1
                sentence_index += 1

        res["tokens"].append(sent_tokens)

    print(json.dumps(res))


if __name__ == "__main__":
    main()
