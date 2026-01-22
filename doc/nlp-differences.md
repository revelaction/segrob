# NLP Framework Differences: Stanza vs spaCy in Segrob

This document details the technical differences between the outputs of the Stanza and spaCy pipelines as integrated into Segrob. These differences primarily concern Multi-Word Token (MWT) handling, Part-of-Speech (POS) tagging granularity, and morphological feature sets.

## Summary Table

| Feature | Stanza (`nlp_stanza.py`) | spaCy (`nlp_spacy.py`) |
| :--- | :--- | :--- |
| **Contractions (e.g., "al")** | Splits into syntactic units (`a` + `el`) | Keeps as single token (`al`) |
| **Clitics (e.g., "dámelo")** | Splits into syntactic units | Splits based on `lemma_` string |
| **MWT Metadata** | Unique `pos`, `tag`, `lemma` per part | Duplicates `pos` and `tag` from parent |
| **Tag Format** | `POS__Morph` (Normalized) | `POS__Morph` (Normalized) |
| **Speed** | Slow (~15+ min/book) | Fast (~1 min/book) |
| **Precision** | High (Rule-based + Statistical) | Moderate (Statistical) |

---

## Multi-Word Token (MWT) Handling

One of the most significant differences is how compound words and clitics are decomposed.

### Contractions (e.g., "al")

*   **Stanza**: Correctly identifies "al" as a contraction and produces two distinct tokens sharing the same source text and offset (`idx`).
    ```json
    [
      {"lemma": "a", "pos": "ADP", "text": "al", "idx": 4},
      {"lemma": "el", "pos": "DET", "text": "al", "idx": 4}
    ]
    ```
*   **spaCy**: Treats "al" as a single token.
    ```json
    [
      {"lemma": "al", "pos": "ADP", "text": "al", "idx": 4}
    ]
    ```

### Clitics (e.g., "Envolverse")

Both frameworks split clitics, but the metadata associated with the split parts differs.

*   **Stanza**: Assigns the correct POS and morphological tags to each component.
    *   `envolver` -> `pos: VERB`, `tag: VERB__VerbForm=Inf`
    *   `se` (lemma `él`) -> `pos: PRON`, `tag: PRON__Case=Acc|Person=3...`
*   **spaCy**: The integration script `nlp_spacy.py` splits the token based on the space-separated `lemma_` attribute (`envolver él`). However, it **duplicates** the parent token's `pos` and `tag` for all resulting parts.
    *   `envolver` -> `pos: VERB`, `tag: VERB__VerbForm=Inf...`
    *   `él` -> `pos: VERB`, `tag: VERB__VerbForm=Inf...` (Incorrect POS/Tag)

---

## POS and Tag Normalization

Segrob normalizes tags to the `POS__Morph` format to allow consistent querying.

*   **Stanza**: Morphological features are provided by the `feats` attribute in Universal Dependencies format.
*   **spaCy**: Uses the `token.morph` attribute. The morphological feature set may differ slightly from Stanza's UD-compliant features.

---

## Indexing and Offsets

Both scripts ensure compatibility with Segrob's core logic:

1.  **`idx`**: Character offset from the start of the document. For MWTs, all constituent tokens share the same `idx` to prevent duplication during reconstruction/rendering.
2.  **`index`**: Token position within the sentence (0-based). Both scripts increment this for each syntactic unit, even within an MWT.
3.  **`head`**: Syntactic head index. Both scripts normalize this to a 0-based index relative to the start of the sentence.

---

## Recommendation

*   **Use Stanza** when grammatical precision is paramount, especially for complex linguistic analysis of clitics and contractions.
*   **Use spaCy** for large-scale processing where speed is critical, and the slight loss of metadata granularity in multi-word tokens is acceptable.
