package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/revelaction/segrob/epub"
)

func main() {
	// Parse arguments
	outputDir := flag.String("output-dir", "", "Optional output directory for .meta.toml files")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: go run go_epub_meta.go [--output-dir DIR] <directory>")
		os.Exit(1)
	}
	directory := args[0]

	// Validate directories
	if info, err := os.Stat(directory); err != nil || !info.IsDir() {
		log.Fatalf("Error: %s is not a valid directory.", directory)
	}
	if *outputDir != "" {
		if info, err := os.Stat(*outputDir); err != nil || !info.IsDir() {
			log.Fatalf("Error: Output directory %s is not a valid directory.", *outputDir)
		}
	}

	// Glob .epub files
	files, err := filepath.Glob(filepath.Join(directory, "*.epub"))
	if err != nil {
		log.Fatal(err)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No .epub files found in %s.\n", directory)
		return
	}

	for _, epubPath := range files {
		processEPUB(epubPath, *outputDir)
	}
}

func processEPUB(epubPath string, outputDir string) {
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filepath.Base(epubPath), err)
		return
	}
	defer r.Close()

	book, err := epub.New(&r.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing OPF in %s: %v\n", filepath.Base(epubPath), err)
		return
	}

	creator := book.Creator()
	title := book.Title()
	date := book.Date()
	language := book.Language()
	translator := book.Translator()

	// Status output
	essentials := map[string]string{
		"creator":  creator,
		"title":    title,
		"date":     date,
		"language": language,
	}
	var found, missing []string

	// Check essentials
	keys := []string{"creator", "title", "date", "language"}
	for _, k := range keys {
		if essentials[k] != "" {
			found = append(found, k)
		} else {
			missing = append(missing, k)
		}
	}
	if translator != "" {
		found = append(found, "translator")
	}

	foundStr := "None"
	if len(found) > 0 {
		foundStr = strings.Join(found, ", ")
	}
	missingStr := "None"
	if len(missing) > 0 {
		missingStr = strings.Join(missing, ", ")
	}

	fmt.Fprintf(os.Stderr, "[%s] Found: %s | Missing: %s\n", filepath.Base(epubPath), foundStr, missingStr)

	// Prepare TOML
	source := strings.TrimSuffix(filepath.Base(epubPath), filepath.Ext(epubPath))
	var labels []string
	if creator != "" {
		labels = append(labels, fmt.Sprintf("creator:%s", creator))
	}
	if title != "" {
		labels = append(labels, fmt.Sprintf("title:%s", title))
	}
	if date != "" {
		labels = append(labels, fmt.Sprintf("date:%s", date))
	}
	if language != "" {
		labels = append(labels, fmt.Sprintf("language:%s", language))
	}
	if translator != "" {
		labels = append(labels, fmt.Sprintf("translator:%s", translator))
	}

	// Quote labels
	for i, l := range labels {
		labels[i] = fmt.Sprintf("\"%s\"", l)
	}
	labelsStr := strings.Join(labels, ", ")
	tomlString := fmt.Sprintf("source = \"%s\"\nlabels = [%s]\n", source, labelsStr)

	// Determine output path
	outDir := filepath.Dir(epubPath)
	if outputDir != "" {
		outDir = outputDir
	}
	outPath := filepath.Join(outDir, source+".meta.toml")

	// Write file
	if err := os.WriteFile(outPath, []byte(tomlString), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing TOML for %s: %v\n", filepath.Base(epubPath), err)
	}
}
