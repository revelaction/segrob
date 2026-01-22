# NLP Framework Differences: Stanza vs spaCy

This document details the native technical differences between the Stanza and spaCy NLP frameworks as they pertain to Spanish language processing. These differences are normalized by Segrob's integration layer to provide a consistent analysis format.

## Summary Table

| Feature | Stanza (Native) | spaCy (Native) |
| :--- | :--- | :--- |
| **MWT Strategy** | Multi-Word Tokens (MWTs) as separate syntactic units | Single tokens with composite lemmas |
| **Contractions** | Splits "al" into "a" + "el" | Keeps "al" as a single token |
| **Clitics** | Decomposes "dámelo" into "dar" + "me" + "lo" | Single token with lemma "dar yo él" |
| **Morphology** | Universal Dependencies (UD) compliant | Custom morphological attributes |
| **Speed** | High latency (GPU recommended) | High throughput (CPU optimized) |
| **Precision** | Higher (especially for MWTs and morphology) | Moderate |

---

## Multi-Word Token (MWT) Handling

The most significant architectural difference is how frameworks handle syntactic units that are fused in the surface text.

### Contractions (e.g., "al")

*   **Stanza**: Identifies "al" as a contraction and natively produces two syntactic words linked to the same surface text.
    *   **Native Output (Simplified)**:
        ```json
        [
          {"text": "al", "lemma": "a", "pos": "ADP"},
          {"text": "al", "lemma": "el", "pos": "DET"}
        ]
        ```
*   **spaCy**: Treats "al" as an atomic syntactic unit.
    *   **Native Output (Simplified)**:
        ```json
        [
          {"text": "al", "lemma": "al", "pos": "ADP"}
        ]
        ```

### Clitics (e.g., "envolverse")

*   **Stanza**: Decomposes the token into its constituent grammatical parts, each with its own POS and morphological features.
    *   **Native Output (Simplified)**:
        ```json
        [
          {"text": "envolver", "lemma": "envolver", "pos": "VERB", "feats": "VerbForm=Inf"},
          {"text": "se", "lemma": "él", "pos": "PRON", "feats": "Case=Acc|Person=3|Reflex=Yes"}
        ]
        ```
*   **spaCy**: Maintains the surface token as a single unit but indicates the decomposition in the lemma attribute.
    *   **Native Output (Simplified)**:
        ```json
        [
          {"text": "envolverse", "lemma": "envolver él", "pos": "VERB", "morph": "VerbForm=Inf"}
        ]
        ```

---

## POS and Morphological Tagging

### Tag Granularity

*   **Stanza**: Provides morphological features in a standardized string format (UD `feats`). POS and features are separate.
    *   *Example*: `pos: VERB`, `feats: Mood=Ind|Number=Sing|Person=3`
*   **spaCy**: Provides a `morph` object and a `tag_` (fine-grained POS) attribute. The `tag_` often contains framework-specific codes.
    *   *Example*: `pos: VERB`, `morph: Mood=Ind|Number=Sing|Person=3`, `tag: VMIP3S0`

### Normalization in Segrob

To allow consistent querying across frameworks, Segrob's integration scripts (`nlp_spacy.py` and `nlp_stanza.py`) unify these outputs into a common `POS__Morph` format:

*   **Unified Format**: `VERB__Mood=Ind|Number=Sing|Person=3`

This allows a single Segrob query to find matches regardless of whether the document was processed by Stanza or spaCy.

---

## Indexing and Dependencies

The frameworks differ significantly in how they reference other tokens (e.g., for dependency parsing).

*   **Stanza**: Uses **1-based indexing** relative to the start of the sentence.
    *   `head=1` refers to the first word in the sentence.
    *   `head=0` refers to the root of the sentence.
*   **spaCy**: Uses **document-level indexing** (absolute token offset).
    *   `token.head.i` returns the index of the head token within the entire document, not just the sentence.

*Segrob Normalization*: The integration scripts convert both systems to **0-based indexing relative to the sentence** (where `head=index` indicates root).

---

## Performance and Use Cases

| Framework | Best For... | Trade-off |
| :--- | :--- | :--- |
| **Stanza** | Academic-grade linguistic analysis, high-precision morphology, and complex clitic handling. | Very slow (~15+ min/book). |
| **spaCy** | Large-scale corpus indexing, real-time processing, and general-purpose NLP. | Loses granularity in MWT decomposition (e.g., clitics share parent POS/Tag in unified output). |

---

*Note: The Segrob integration scripts are responsible for mapping these native behaviors to the [Doc JSON Format](doc-json-format.md).*