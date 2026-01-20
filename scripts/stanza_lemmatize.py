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
        # some mwt multi word tokens will have many words
        # we convert to segrob tokens, where the first segrob token 
        # [
        #   {
        #     "id": "5-6",
        #     "text": "envolverse",
        #     "ner": "O",
        #     "misc": "start_char=2431|end_char=2441"
        #   },
        #   {
        #     "id": "5",
        #     "text": "envolver",
        #     "lemma": "envolver",
        #     "upos": "VERB",
        #     "xpos": "VERB",
        #     "feats": "VerbForm=Inf",
        #     "head": 4,
        #     "deprel": "xcomp"
        #   },
        #   {
        #     "id": "6",
        #     "text": "se",
        #     "lemma": "él",
        #     "upos": "PRON",
        #     "xpos": "PRON",
        #     "feats": "Case=Acc,Dat|Person=3|PrepCase=Npr|PronType=Prs|Reflex=Yes",
        #     "head": 5,
        #     "deprel": "obj"
        #   }
        # ]

        # todo document
        # all words of multi token also recevive the same index, that of the token 
        #     "misc": "start_char=2431|end_char=2441"
        idx = 0
        m = re.search(r'start_char=(.+)\|', token.misc)
        if m:
            idx = int(m.group(1))
        else:
            raise ValueError('A very specific bad thing happened.')
            
        # do not get the dict with the misc

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
