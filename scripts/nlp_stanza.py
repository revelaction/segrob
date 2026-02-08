#!/usr/bin/env python3
# coding: utf-8
"""
Tokenize, POS tag and lemmatize a text file using Stanza.

Outputs a JSON document in the segrob doc format to standard output.
"""
import argparse
import json
from pathlib import Path
import stanza

def add_labels(cmd_labels, file_path):
    """Build labels list from filename parts, stanza version, and command-line labels."""
    labels = []

    # Optimistically consider the file name properly formatted
    # <title>-<author>-<traductor>.txt
    file_name = Path(file_path).stem
    labels.extend(file_name.split("-"))

    # Lemmatizer version
    labels.append(f"{stanza.__name__}-{stanza.__version__}")

    # Command-line labels
    if cmd_labels is not None:
        labels.extend(cmd_labels)

    return labels


def main():
    parser = argparse.ArgumentParser(
        description='Tokenize, POS tag and lemmatize a text to standard output'
    )
    parser.add_argument(
        'filename', 
        type=argparse.FileType('r', encoding='UTF-8'),
        help="input text file", 
        metavar="FILE"
    )
    parser.add_argument(
        '-l', '--label', 
        action='append', 
        help="add a label to the document metadata (can be used multiple times)"
    )

    args = parser.parse_args()

    # Read input file
    with args.filename as txt_file:
        text = txt_file.read()
        file_name = txt_file.name

    # STANZA
    # Initialize pipeline (download if necessary)
    stanza.download('es')
    nlp = stanza.Pipeline("es")
    doc = nlp(text)

    res = {
        "labels": add_labels(args.label, file_name),
        "sentences": []
    }
    
    token_index = 0
    sentence_id = 0

    for sentence in doc.sentences:
        sent_dict = []
        for token in sentence.tokens:
            # Multi-Word Token (MWT) Logic:
            # Stanza identifies MWTs (like "envolverse" or "del") as a single Token 
            # that contains multiple syntactic Words (Word objects).
            #
            # Stanza's internal representation for an MWT (e.g. "envolverse" -> "envolver" + "se") looks like this:
            # 1. Surface Token: { "id": "5-6", "text": "envolverse", "start_char": 2431, ... }
            # 2. Words:
            #    - Word 1: { "id": "5", "text": "envolver", "lemma": "envolver", "pos": "VERB", ... }
            #    - Word 2: { "id": "6", "text": "se", "lemma": "él", "pos": "PRON", ... }
            # 
            # Full Token JSON (to_dict):
            # 		[
            # 		  {
            # 		    "id": [
            # 		      13,
            # 		      14
            # 		    ],
            # 		    "text": "meterse",
            # 		    "start_char": 62,
            # 		    "end_char": 69,
            # 		    "ner": "O",
            # 		    "multi_ner": [
            # 		      "O"
            # 		    ]
            # 		  },
            # 		  {
            # 		    "id": 13,
            # 		    "text": "meter",
            # 		    "lemma": "meter",
            # 		    "upos": "VERB",
            # 		    "xpos": "vmn0000",
            # 		    "feats": "VerbForm=Inf",
            # 		    "head": 5,
            # 		    "deprel": "conj",
            # 		    "start_char": 62,
            # 		    "end_char": 67
            # 		  },
            # 		  {
            # 		    "id": 14,
            # 		    "text": "se",
            # 		    "lemma": "él",
            # 		    "upos": "PRON",
            # 		    "feats": "Case=Acc|Person=3|PrepCase=Npr|PronType=Prs|Reflex=Yes",
            # 		    "head": 13,
            # 		    "deprel": "expl:pv",
            # 		    "start_char": 67,
            # 		    "end_char": 69
            # 		  }
            # 		]
            # 		        #
            # Segrob's Domain Logic Requirement:
            # ----------------------------------
            # Segrob expects a flat sequence of syntactic units. For MWTs, we create
            # a separate entry for each syntactic Word, but we preserve the link to the 
            # original surface text by:
            # 1. Assigning the SAME 'text' (e.g. "envolverse") to all constituent words.
            # 2. Assigning the SAME 'idx' (start character offset) to all constituent words.
            #
            # This allows Segrob to reconstruct the sentence with correct spacing while 
            # maintaining the full syntactic analysis.

            # Grab the physical start character index from the parent Token wrapper.
            # This attribute is the modern way to get position data (replacing regex on token.misc).
            idx = token.start_char
            if idx is None:
                raise ValueError(f"Token '{token.text}' is missing start_char information")
            
            # Process each syntactic Word within the Stanza Token.
            # For regular tokens, this loop runs once. For MWTs, it runs multiple times.
            for word in token.words:
                t = {}
                t['id'] = token_index
                t['pos'] = word.upos

                # The way both frameworks deal the pos and detailed pos information is different:
                # 
                # `spacy` copies the pos field value as prefix in the tag field:
                # 
                #         "pos": "VERB",
                #         "tag": "VERB__Mood=Ind|Number=Sing|Person=3|Tense=Pres|VerbForm=Fin",
                #         "dep": "root",
                # 
                # `stanza` does not do this:
                #         "pos": "VERB",
                #         "tag": "Mood=Ind|Number=Sing|Person=3|Tense=Pres|VerbForm=Fin",
                #         "dep": "root",
                #
                # we add here for simplicity and compatibility.
                if word.feats is None:
                # sometime feats is not existant in stanza: repeat pos
                if word.feats is None:
                    t['tag'] = word.upos 
                else:
                    t['tag'] = word.upos + "__" + word.feats

                t['dep'] = word.deprel

                # Head index adjustment (Stanza is 1-based, Segrob internal is 0-based)
                if word.head > 0:
                    t['head'] = int(word.head) - 1 
                else:
                    t['head'] = 0

                t['text'] = token.text

                # Character offset from start of document
                t['idx'] = idx

                # Token position in sentence (internal 0-based index)
                # word.id might be a string "1", "2" etc.
                t['index'] = int(word.id) - 1
                
                # Lemma lowercase 
                t['lemma'] = word.lemma.lower() 
                
                sent_dict.append(t)
                token_index += 1

        res['sentences'].append({
            "id": sentence_id,
            "tokens": sent_dict
        })
        sentence_id += 1

    print(json.dumps(res))


if __name__ == "__main__":
    main()
