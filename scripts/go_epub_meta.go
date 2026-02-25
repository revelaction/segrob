package main

import (
	"archive/zip"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ContainerXML represents META-INF/container.xml
type ContainerXML struct {
	Rootfiles []Rootfile `xml:"rootfiles>rootfile"`
}

type Rootfile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

// PackageXML represents the OPF file content
type PackageXML struct {
	Metadata Metadata `xml:"metadata"`
}

type Metadata struct {
	Titles       []string      `xml:"title"`
	Creators     []Creator     `xml:"creator"`
	Contributors []Creator     `xml:"contributor"`
	Dates        []Date        `xml:"date"`
	Language     []string      `xml:"language"`
	Description  []string      `xml:"description"`
}

type Creator struct {
	Value  string `xml:",chardata"`
	Role   string `xml:"role,attr"`
	FileAs string `xml:"file-as,attr"`
}

type Date struct {
	Value string `xml:",chardata"`
	Event string `xml:"event,attr"`
}

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
	// Open the EPUB (zip) file
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filepath.Base(epubPath), err)
		return
	}
	defer r.Close()

	// Find OPF path
	opfPath, err := findOPFPath(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filepath.Base(epubPath), err)
		return
	}

	// Parse OPF
	pkg, err := parseOPF(r, opfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing OPF in %s: %v\n", filepath.Base(epubPath), err)
		return
	}

	// Extract and normalize metadata
	m := pkg.Metadata
	creator := normalizeValue(extractCreator(m))
	title := normalizeValue(extractTitle(m))
	date := normalizeDate(extractDatePublication(m))
	language := normalizeValue(extractLanguage(m))
	translator := normalizeValue(extractTranslator(m))

	// Status output
	essentials := map[string]string{
		"creator":  creator,
		"title":    title,
		"date":     date,
		"language": language,
	}
	var found, missing []string
	
	// Check essentials
	// Order isn't guaranteed in map, but let's check keys in a stable order for printing
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

func findOPFPath(r *zip.ReadCloser) (string, error) {
	for _, f := range r.File {
		if f.Name == "META-INF/container.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			var container ContainerXML
			if err := xml.NewDecoder(rc).Decode(&container); err != nil {
				return "", err
			}

			if len(container.Rootfiles) > 0 {
				return container.Rootfiles[0].FullPath, nil
			}
			break
		}
	}
	return "", fmt.Errorf("could not find OPF path in container.xml")
}

func parseOPF(r *zip.ReadCloser, opfPath string) (PackageXML, error) {
	var pkg PackageXML
	for _, f := range r.File {
		if f.Name == opfPath {
			rc, err := f.Open()
			if err != nil {
				return pkg, err
			}
			defer rc.Close()

			if err := xml.NewDecoder(rc).Decode(&pkg); err != nil {
				return pkg, err
			}
			return pkg, nil
		}
	}
	return pkg, fmt.Errorf("OPF file not found in zip archive")
}

// ---------------------------------------------------------------------------
// Normalization
// ---------------------------------------------------------------------------

func normalizeValue(val string) string {
	if val == "" {
		return ""
	}
	val = strings.ToLower(val)
	val = strings.ReplaceAll(val, " ", "_")
	val = strings.ReplaceAll(val, "-", "_")
	return val
}

func normalizeDate(val string) string {
	if val == "" {
		return ""
	}
	re := regexp.MustCompile(`\d{4}`)
	return re.FindString(val)
}

// ---------------------------------------------------------------------------
// Extract functions
// ---------------------------------------------------------------------------

func extractCreator(m Metadata) string {
	for _, c := range m.Creators {
		// return first where role != 'trl'
		if c.Role != "trl" {
			return c.Value
		}
	}
	return ""
}

func extractTranslator(m Metadata) string {
	// Check contributors first
	for _, c := range m.Contributors {
		if c.Role == "trl" {
			return c.Value
		}
	}
	// Then check creators
	for _, c := range m.Creators {
		if c.Role == "trl" {
			return c.Value
		}
	}
	return ""
}

func extractTitle(m Metadata) string {
	if len(m.Titles) > 0 {
		return m.Titles[0]
	}
	return ""
}

func extractLanguage(m Metadata) string {
	if len(m.Language) > 0 {
		return m.Language[0]
	}
	return ""
}

func extractDatePublication(m Metadata) string {
	for _, d := range m.Dates {
		// Python script checks for 'event' attribute being 'publication'
		// It checks opf:event, event, etc.
		// Our struct 'Event' captures the attribute 'event' (ignoring namespace typically in standard encoding/xml if not strict)
		if d.Event == "publication" {
			return d.Value
		}
	}
	return ""
}
