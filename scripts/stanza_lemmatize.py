#!/usr/bin/python3
# coding: utf-8
import argparse
import json
import re
from pathlib import Path
import stanza

def add_labels(cmd_labels, file_path):
    labels = []

    # optimistically consider the file name properly formated
    # <title>-<author>-<traductor>.txt
    file_name = Path(file_path).stem
    labels.extend(file_name.split("-"))

    # lemmatizer version
    labels.append(stanza.__name__ + "-" + stanza.__version__)

    # possible command
    # check commmand options exists
    if cmd_labels is not None:
        for l in cmd_labels:
            labels.append(l)

    return labels


# Arg parser
parser = argparse.ArgumentParser(description='tokenize, POS  tag and lemmatize a text to standard output')
parser.add_argument('filename', type=argparse.FileType('r', encoding='UTF-8'),
        help="input text file", metavar="FILE")

parser.add_argument("-d", "--diff", action="store_true", help="show only diff in lemma or pos fields")

# will provide a list
parser.add_argument('-l', '--label', action='append', help="show only diff in lemma or pos fields")

args = parser.parse_args()

txt_file = args.filename
f = txt_file.read()

#
# STANZA
#
stanza.download('es') # nach hause telefonieren
nlp = stanza.Pipeline("es")
doc = nlp(f)
# For debugging
#doc = nlp("El coronel habría preferido envolverse en una manta de lana y meterse otra vez en la hamaca.")



res = {}
res["labels"] = add_labels(args.label, txt_file.name)
res["tokens"] = []
token_index = 0


for sentence in doc.sentences:
    sent_dict = []
    for token in sentence.tokens:

        # Multi-Word Token (MWT) Logic:
        # -----------------------------
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
                t['tag'] = word.upos 
            else:
                t['tag'] = word.upos + "__" + word.feats
                    
            t['dep'] = word.deprel
            # head: stanza is 1 based, segron is 0 based
            if word.head > 0:
                t['head'] = int(word.head) -1 
            else:
                t['head'] = 0

            t['text'] = token.text

            # not used anymore TODO remove
            #t['sent'] = 0 
                
            # This is the character offset from start of the document (uft8).
            # It is used to position the word with spaces when printing the
            # sentence. 
            t['idx'] = idx
            # Index is the token position in the sentence
            t['index'] = int(word.id) - 1
            # lemma lowercase 
            t['lemma'] = word.lemma.lower()
            sent_dict.append(t)
            token_index +=1


    res['tokens'].append(sent_dict)

#for sentence in doc.sentences:
#    print("-------", sentence.text)
#    for word in sentence.tokens:
#    #for word in sentence.words:
#        #print(word.text, word.lemma, word.pos)
#        print(word)

print(json.dumps(res))
