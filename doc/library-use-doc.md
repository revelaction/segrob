# Using segrob as a Library

`segrob` is designed with a clear separation between storage, data models, and rendering, making it easy to integrate into other Go applications (such as web servers, desktop tools, or analysis scripts).

This guide focuses on using the **SQLite backend** (powered by `zombiezen.com/go/sqlite`) to retrieve and render documents.

## Architecture Overview

1.  **Storage**: The `storage` package defines interfaces (`DocRepository`). The `storage/sqlite/zombiezen` package provides a high-performance implementation using a connection pool.
2.  **Models**: The `sentence` package contains the core data structures like `Doc` and `Token`.
3.  **Rendering**: The `render` package handles the conversion of annotated tokens into human-readable text (with optional color support).

## Dependency Setup

To use `segrob` in your project, you will need the following imports:

```go
import (
	"context"
	"fmt"
	"log"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
	"zombiezen.com/go/sqlite/sqlitex"
)
```

## Core Workflow

### 1. Initialize the SQLite Pool

The SQLite backend requires a `sqlitex.Pool`. This pool manages concurrent access to the database file.

```go
dbPath := "path/to/your/corpus.db"
pool, err := sqlitex.Open(dbPath, 0, 10) // 10 is the pool size
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

### 2. Setup the Repository

Instantiate the `DocStore` using the pool. This object implements the `storage.DocRepository` interface.

```go
repo := zombiezen.NewDocStore(pool)
```

### 3. Accessing Data

You can list metadata for all documents or read a specific document by its ID.

```go
// List all documents (metadata only, no tokens loaded)
docs, err := repo.List()
if err != nil {
    log.Fatal(err)
}

for _, d := range docs {
    fmt.Printf("ID: %d, Title: %s\n", d.Id, d.Title)
}

// Read a full document by ID (loads all sentences and tokens)
targetDocID := 1
doc, err := repo.Read(targetDocID)
if err != nil {
    log.Fatal(err)
}
```

### 4. Rendering Content

Use the `render` package to display the document's sentences. This is useful for building custom "book viewers".

```go
r := render.NewRenderer()
r.HasColor = false // Set to true if outputting to a TTY

for i, sentence := range doc.Tokens {
    prefix := fmt.Sprintf("[%d] ", i)
    r.Sentence(sentence, prefix)
}
```

## Complete Example: Rendering a Book

Here is a complete function that demonstrates the full process of opening a database and printing a specific book to standard output.

```go
func PrintBook(dbPath string, docID int) error {
	// 1. Initialize Pool
	pool, err := sqlitex.Open(dbPath, 0, 10)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer pool.Close()

	// 2. Initialize Repository
	repo := zombiezen.NewDocStore(pool)

	// 3. Read Document
	doc, err := repo.Read(docID)
	if err != nil {
		return fmt.Errorf("failed to read doc %d: %w", docID, err)
	}

	// 4. Render
	fmt.Printf("--- %s ---\
", doc.Title)
	r := render.NewRenderer()
	r.HasColor = false

	for i, tokens := range doc.Tokens {
		prefix := fmt.Sprintf("%d: ", i)
		r.Sentence(tokens, prefix)
	}

	return nil
}
```

## Advanced Usage: Lemma Search

The SQLite backend also supports efficient lemma-based searching via `FindCandidates`. This allows you to find sentences containing specific words across the entire corpus:

```go
lemmas := []string{"querer", "hacer"}
results, nextCursor, err := repo.FindCandidates(lemmas, 0, 20)
if err != nil {
    log.Fatal(err)
}

for _, res := range results {
    fmt.Printf("Found in %s (DocID %d):\n", res.DocTitle, res.DocID)
    r.Sentence(res.Tokens, "> ")
}
```

