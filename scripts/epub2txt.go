package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <epub-file>\n", os.Args[0])
		os.Exit(1)
	}

	epubPath := os.Args[1]
	text, err := extractEpubText(epubPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(text)
}

func extractEpubText(path string) (string, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("opening zip: %w", err)
	}
	defer z.Close()

	// 1. Find OPF path from META-INF/container.xml
	opfPath, err := findOPFPath(z)
	if err != nil {
		return "", fmt.Errorf("finding OPF: %w", err)
	}

	// 2. Parse OPF to get Manifest and Spine
	pkg, err := parseOPF(z, opfPath)
	if err != nil {
		return "", fmt.Errorf("parsing OPF: %w", err)
	}

	// 3. Map Manifest IDs to Hrefs
	idToHref := make(map[string]string)
	for _, item := range pkg.Manifest.Items {
		idToHref[item.ID] = item.Href
	}

	// 4. Iterate Spine to get reading order
	var fullText strings.Builder
	opfDir := filepath.Dir(opfPath)

	for _, itemRef := range pkg.Spine.ItemRefs {
		href, ok := idToHref[itemRef.IDRef]
		if !ok {
			continue // Should not happen in valid EPUB
		}

		// Resolve path relative to OPF file
		// href in OPF is relative to the OPF file itself
		fullPath := filepath.Join(opfDir, href)
		// filepath.Join uses backslash on Windows, but zip uses forward slash
		fullPath = filepath.ToSlash(fullPath)

		content, err := readZipFile(z, fullPath)
		if err != nil {
			// Try without directory prefix if failed (sometimes paths are tricky)
			content, err = readZipFile(z, href)
			if err != nil {
				return "", fmt.Errorf("reading content file %s: %w", fullPath, err)
			}
		}

		text, err := extractTextFromXHTML(content)
		if err != nil {
			return "", fmt.Errorf("extracting text from %s: %w", fullPath, err)
		}

		fullText.WriteString(text)
		fullText.WriteString("\n")
	}

	return fullText.String(), nil
}

// XML Structures

type ContainerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type PackageXML struct {
	Manifest Manifest `xml:"manifest"`
	Spine    Spine    `xml:"spine"`
}

type Manifest struct {
	Items []Item `xml:"item"`
}

type Item struct {
	ID   string `xml:"id,attr"`
	Href string `xml:"href,attr"`
}

type Spine struct {
	ItemRefs []ItemRef `xml:"itemref"`
}

type ItemRef struct {
	IDRef string `xml:"idref,attr"`
}

// Helpers

func findOPFPath(z *zip.ReadCloser) (string, error) {
	data, err := readZipFile(z, "META-INF/container.xml")
	if err != nil {
		return "", err
	}

	var container ContainerXML
	if err := xml.Unmarshal(data, &container); err != nil {
		return "", err
	}

	if len(container.Rootfiles) == 0 {
		return "", fmt.Errorf("no rootfiles found in container.xml")
	}
	return container.Rootfiles[0].FullPath, nil
}

func parseOPF(z *zip.ReadCloser, opfPath string) (PackageXML, error) {
	data, err := readZipFile(z, opfPath)
	if err != nil {
		return PackageXML{}, err
	}

	var pkg PackageXML
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return PackageXML{}, err
	}
	return pkg, nil
}

func readZipFile(z *zip.ReadCloser, name string) ([]byte, error) {
	// Normalize name to forward slashes just in case
	name = strings.ReplaceAll(name, "\\", "/")
	
	for _, f := range z.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// Text Extraction

var entityPattern = regexp.MustCompile(`&[a-zA-Z]+;`)

func resolveEntities(content []byte) []byte {
	return entityPattern.ReplaceAllFunc(content, func(match []byte) []byte {
		s := string(match)
		// Keep standard XML entities as is
		if s == "&lt;" || s == "&gt;" || s == "&amp;" || s == "&apos;" || s == "&quot;" {
			return match
		}
		// Unescape others (e.g. &nbsp; -> \u00A0)
		return []byte(html.UnescapeString(s))
	})
}

func extractTextFromXHTML(content []byte) (string, error) {
	// Pre-process content to resolve HTML entities that XML doesn't support
	content = resolveEntities(content)
	
	decoder := xml.NewDecoder(bytes.NewReader(content))
	var buf strings.Builder
	
	// Tags to ignore content from
	ignoreTags := map[string]bool{
		"head": true, "script": true, "style": true, "title": true,
	}
	// Tags that trigger a newline/break
	blockTags := map[string]bool{
		"p": true, "div": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"li": true, "blockquote": true, "pre": true, "section": true, "header": true, "footer": true, "article": true,
	}

	ignoreDepth := 0
	
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := token.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if ignoreTags[name] {
				ignoreDepth++
			}
			if blockTags[name] {
				// Ensure we have a break before block
				buf.WriteString("\n")
			}
			if name == "br" {
				buf.WriteString("\n")
			}

		case xml.EndElement:
			name := strings.ToLower(t.Name.Local)
			if ignoreTags[name] {
				ignoreDepth--
			}
			if blockTags[name] {
				// Ensure we have a break after block
				buf.WriteString("\n")
			}

		case xml.CharData:
			if ignoreDepth > 0 {
				continue
			}
			text := string(t)
			// Normalize whitespace: replace newlines/tabs with space
			text = strings.Map(func(r rune) rune {
				if r == '\n' || r == '\r' || r == '\t' {
					return ' '
				}
				return r
			}, text)
			
			// Collapse multiple spaces? Pandoc "plain" might do this for HTML input.
			// Simple collapse:
			text = strings.Join(strings.Fields(text), " ")
			
			if len(text) > 0 {
				// Determine if we need a space before this text chunk
				// If the buffer ends with a newline, we don't need a space
				// If it ends with a character, we might need a space if the previous chunk didn't end with one?
				// Actually, HTML collapses whitespace between tags too. 
				// Simple approach: just write it, maybe with a leading space if needed.
				// But strings.Fields strips leading/trailing spaces.
				
				// Let's just write it. Ideally, we should track spacing.
				// For simple extraction, writing it with a preceding space if buffer is not empty/newline is safer.
				
				currLen := buf.Len()
				if currLen > 0 {
					lastChar := buf.String()[currLen-1]
					if lastChar != '\n' && lastChar != ' ' {
						buf.WriteString(" ")
					}
				}
				
				buf.WriteString(text)
			}
		}
	}

	// Post-processing to clean up excessive newlines
	result := buf.String()
	// Replace 3+ newlines with 2
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(result), nil
}
