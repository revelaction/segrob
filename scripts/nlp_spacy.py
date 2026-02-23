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
        "sentences": [],
    }

    token_index = 0
    sentence_id = 0

    for sentence in doc.sents:
        sent_tokens = []
        unique_lemmas = set()
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
                lemma = word.lower()
                t = {
                    "id": token_index,
                    "pos": token.pos_,
                    "tag": tag,
                    "dep": token.dep_.lower(),  # segrob requires lowercase
                    "head": head_idx,
                    "text": token.text,
                    "idx": token.idx,  # Character offset in source document
                    "index": sentence_index,  # Token position in sentence
                    "lemma": lemma,
                }
                if lemma:
                    unique_lemmas.add(lemma)

                sent_tokens.append(t)
                token_index += 1
                sentence_index += 1

        res["sentences"].append({
            "id": sentence_id,
            "lemmas": list(unique_lemmas),
            "tokens": sent_tokens
        })
        sentence_id += 1

    print(json.dumps(res))


if __name__ == "__main__":
    main()
