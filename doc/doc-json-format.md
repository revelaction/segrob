# Doc JSON Format

This document specifies the JSON format used by segrob for ingesting tokenized documents. The format provides a common structure for NLP-tokenized text, independent of whether the input was processed by spacy or stanza.

## Table of Contents

- [Document Structure](#document-structure)
- [Token Fields Reference](#token-fields-reference)
- [Sentence-Oriented Structure](#sentence-oriented-structure)
- [Multi-Word Token Handling](#multi-word-token-handling)
- [Framework Differences: spacy vs stanza](#framework-differences-spacy-vs-stanza)
- [Labels Field](#labels-field)
- [Complete Example](#complete-example)

---

## Document Structure

A document is a JSON object with two top-level fields:

```json
{
    "labels": [...],
    "tokens": [[...], [...], ...]
}
```

| Field    | Type              | Description                                      |
|----------|-------------------|--------------------------------------------------|
| `labels` | `array<string>`   | Metadata labels for the document                 |
| `tokens` | `array<array<Token>>` | 2D array: list of sentences, each a list of tokens |

---

## Token Fields Reference

Each token is a JSON object with the following fields:

| Field   | Type     | Description                                                                 |
|---------|----------|-----------------------------------------------------------------------------|
| `id`    | `int`    | Global token ID within the document (auto-incremented across all sentences) |
| `pos`   | `string` | Part-of-speech tag (Universal Dependencies: `VERB`, `NOUN`, `ADJ`, etc.)    |
| `tag`   | `string` | Detailed morphological features (see [Framework Differences](#framework-differences-spacy-vs-stanza)) |
| `dep`   | `string` | Dependency relation (lowercase: `nsubj`, `det`, `root`, etc.)               |
| `head`  | `int`    | Index of the syntactic head token (0-based within the document)             |
| `text`  | `string` | Original word as it appears in the source text                              |
| `sent`  | `int`    | Sentence ID (deprecated, always `0`)                                        |
| `idx`   | `int`    | Character offset from the start of the source document (UTF-8 rune-based)   |
| `index` | `int`    | Token position within the sentence (0-based)                                |
| `lemma` | `string` | Lowercase lemma of the word                                                 |

### Example Token

```json
{
    "id": 15,
    "pos": "DET",
    "tag": "DET__Gender=Masc|Number=Sing|PronType=Art",
    "dep": "det",
    "head": 16,
    "text": "El",
    "sent": 0,
    "idx": 0,
    "index": 0,
    "lemma": "el"
}
```

---

## Sentence-Oriented Structure

The segrob CLI is designed primarily for sentence-level operations. The `tokens` array reflects this by organizing tokens as a **2D array**:

```
tokens[sentence_index][token_index]
```

- The **outer array** contains sentences in document order.
- Each **inner array** contains the tokens of a single sentence.

This structure enables efficient iteration over sentences for matching, querying, and rendering.

---

## Multi-Word Token Handling

Some NLP frameworks (notably stanza) split compound words into multiple tokens. For example, the Spanish word "envolverse" is analyzed as two tokens: "envolver" (verb) + "él" (pronoun).

The segrob format signals multi-word tokens by assigning **identical `idx` values** to all tokens belonging to the same compound word:

```json
{
    "id": 455,
    "pos": "VERB",
    "tag": "VerbForm=Inf",
    "dep": "xcomp",
    "head": 3,
    "text": "envolverse",
    "sent": 0,
    "idx": 2431,
    "index": 4,
    "lemma": "envolver"
},
{
    "id": 456,
    "pos": "PRON",
    "tag": "Case=Acc,Dat|Person=3|PrepCase=Npr|PronType=Prs|Reflex=Yes",
    "dep": "obj",
    "head": 4,
    "text": "envolverse",
    "sent": 0,
    "idx": 2431,
    "index": 5,
    "lemma": "él"
}
```

Key points:
- Both tokens share the same `text` field (the original unsplit word).
- Both tokens share the same `idx` value (character offset).
- The renderer uses `idx` to avoid duplicating the word in output.
- Each token retains its own `lemma`, `pos`, `tag`, and `dep` information.

---

## Framework Differences: spacy vs stanza

The `tag` field format differs between spacy and stanza. The segrob scripts normalize this difference:

| Framework | Tag Format                                        |
|-----------|---------------------------------------------------|
| **spacy** | POS prefix included: `VERB__Mood=Ind\|Number=Sing\|Person=3` |
| **stanza** | No prefix (raw features): `Mood=Ind\|Number=Sing\|Person=3`  |

The segrob tokenizer scripts add the POS prefix for stanza output to ensure compatibility:

```python
# stanza_lemmatize.py normalization
if word.feats is None:
    t['tag'] = word.upos 
else:
    t['tag'] = word.upos + "__" + word.feats
```

This allows segrob expressions to use tag values consistently regardless of the tokenizer used.

### Other Framework Differences

| Aspect                  | spacy                                    | stanza                                      |
|-------------------------|------------------------------------------|---------------------------------------------|
| Multi-word tokens       | Limited support (lemma: `"combinar él"`) | Full support (separate tokens per word)     |
| Lemmatizer              | Statistical only                         | Rule-based + statistical                    |
| Sentence segmentation   | Good                                     | Slightly better                             |
| Speed                   | Fast (~1 min/book)                       | Slow (~15+ min/book)                        |

---

## Labels Field

The `labels` array contains metadata strings for the document:

```json
"labels": [
    "cien-años-de-soledad",
    "gabriel-garcía-márquez",
    "stanza-1.4.0",
    "novela"
]
```

Labels are derived from:
1. **Filename parts**: The source filename is split by `-` (e.g., `title-author-translator.txt`).
2. **Tokenizer version**: Automatically added (e.g., `stanza-1.4.0` or `spacy-3.5.0`).
3. **Command-line labels**: Additional labels passed via `-l` flag.

Labels enable filtering documents by author, title, genre, or tokenizer version.

---

## Complete Example

```json
{
    "labels": [
        "el-coronel",
        "garcía-márquez",
        "stanza-1.4.0",
        "novela"
    ],
    "tokens": [
        [
            {
                "id": 0,
                "pos": "DET",
                "tag": "DET__Gender=Masc|Number=Sing|PronType=Art",
                "dep": "det",
                "head": 1,
                "text": "El",
                "sent": 0,
                "idx": 0,
                "index": 0,
                "lemma": "el"
            },
            {
                "id": 1,
                "pos": "NOUN",
                "tag": "NOUN__Gender=Masc|Number=Sing",
                "dep": "nsubj",
                "head": 2,
                "text": "coronel",
                "sent": 0,
                "idx": 3,
                "index": 1,
                "lemma": "coronel"
            },
            {
                "id": 2,
                "pos": "VERB",
                "tag": "VERB__Mood=Ind|Number=Sing|Person=3|Tense=Past|VerbForm=Fin",
                "dep": "root",
                "head": 2,
                "text": "esperó",
                "sent": 0,
                "idx": 11,
                "index": 2,
                "lemma": "esperar"
            },
            {
                "id": 3,
                "pos": "PUNCT",
                "tag": "PUNCT__PunctType=Peri",
                "dep": "punct",
                "head": 2,
                "text": ".",
                "sent": 0,
                "idx": 18,
                "index": 3,
                "lemma": "."
            }
        ],
        [
            {
                "id": 4,
                "pos": "PRON",
                "tag": "PRON__Case=Nom|Number=Sing|Person=3|PronType=Prs",
                "dep": "nsubj",
                "head": 5,
                "text": "Él",
                "sent": 0,
                "idx": 20,
                "index": 0,
                "lemma": "él"
            },
            {
                "id": 5,
                "pos": "VERB",
                "tag": "VERB__Mood=Ind|Number=Sing|Person=3|Tense=Past|VerbForm=Fin",
                "dep": "root",
                "head": 5,
                "text": "esperaba",
                "sent": 0,
                "idx": 23,
                "index": 1,
                "lemma": "esperar"
            }
        ]
    ]
}
```

This example shows:
- Two sentences in the `tokens` array
- Standard token fields with normalized `tag` format
- Sequential `id` values across sentences
- Per-sentence `index` values (reset to 0 for each sentence)
