//go:build ignore

package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/revelaction/segrob/epub"
)

// metaLsFmt omits FLAGS and ID columns.
// TITLE(25) CREATOR(14) TRANSLATOR(14) DATE(4) LANG
const metaLsFmt = "%-25s  %-14s  %-14s  %-4s  %s\n"

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: go run go_epub_meta.go <directory>")
		os.Exit(1)
	}
	directory := args[0]

	if info, err := os.Stat(directory); err != nil || !info.IsDir() {
		log.Fatalf("Error: %s is not a valid directory.", directory)
	}

	files, err := filepath.Glob(filepath.Join(directory, "*.epub"))
	if err != nil {
		log.Fatal(err)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No .epub files found in %s.\n", directory)
		return
	}

	// Print header
	fmt.Printf(metaLsFmt, "TITLE", "CREATOR", "TRANSLATOR", "DATE", "LANG")

	for _, epubPath := range files {
		processEPUB(epubPath)
	}
}

func processEPUB(epubPath string) {
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		return
	}
	defer r.Close()

	book, err := epub.New(&r.Reader)
	if err != nil {
		return
	}

	meta := book.Labels()

	get := func(key string) string {
		v := meta[key]
		if v == "" {
			return "-"
		}
		return v
	}

	fmt.Printf(metaLsFmt,
		truncate(get("title"), 25),
		truncate(get("creator"), 14),
		truncate(get("translator"), 14),
		get("date"),
		get("language"),
	)
}
