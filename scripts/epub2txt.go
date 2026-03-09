package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"os"
	"path"
	"strings"
)

// EPUB is essentially a ZIP archive containing HTML files, images, and metadata.
// To extract text, we need to:
// 1. Open the ZIP archive.
// 2. Find the "Open Packaging Format" (OPF) file, which describes the book's structure.
//    The path to the OPF file is stored in "META-INF/container.xml".
// 3. Parse the OPF file to find the "Spine" (the reading order of chapters).
// 4. Iterate through the Spine, find the corresponding HTML files in the "Manifest",
//    and extract text from them in order.

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

func extractEpubText(epubPath string) (string, error) {
	// Step 1: Open the EPUB file as a ZIP archive.
	z, err := zip.OpenReader(epubPath)
	if err != nil {
		return "", fmt.Errorf("opening zip: %w", err)
	}
	defer z.Close()

	// Step 2: Find the path to the OPF file.
	// Every valid EPUB has a "META-INF/container.xml" file that points to the .opf file.
	opfPath, err := findOPFPath(z)
	if err != nil {
		return "", fmt.Errorf("finding OPF: %w", err)
	}

	// Step 3: Parse the OPF file.
	// The OPF contains the "Manifest" (list of all files) and "Spine" (reading order).
	pkg, err := parseOPF(z, opfPath)
	if err != nil {
		return "", fmt.Errorf("parsing OPF: %w", err)
	}

	// Map Manifest IDs to file paths (Hrefs).
	// The Spine uses IDs to reference items in the Manifest.
	idToHref := make(map[string]string)
	for _, item := range pkg.Manifest.Items {
		idToHref[item.ID] = item.Href
	}

	// Step 4: Iterate through the Spine to process chapters in reading order.
	var fullText strings.Builder
	opfDir := path.Dir(opfPath) // Paths in OPF are relative to the OPF file itself.

	for _, itemRef := range pkg.Spine.ItemRefs {
		// Find the file path for this spine item using its IDRef.
		href, ok := idToHref[itemRef.IDRef]
		if !ok {
			continue // Should not happen in a valid EPUB, but safe to skip.
		}

		// Resolve the absolute path within the logical ZIP archive.
		// Example: If OPF is "OEBPS/content.opf" and href is "chap1.xhtml",
		// the file in ZIP is "OEBPS/chap1.xhtml". We use the 'path' package
		// (not 'filepath') because ZIP files use forward slashes regardless
		// of the host operating system.
		fullPath := path.Join(opfDir, href)

		// Read the content of the XHTML file.
		content, err := readZipFile(z, fullPath)
		if err != nil {
			// Fallback: Try reading without the directory prefix if the first attempt fails.
			// Some malformed EPUBs might have issues with path resolution.
			content, err = readZipFile(z, href)
			if err != nil {
				return "", fmt.Errorf("reading content file %s: %w", fullPath, err)
			}
		}

		// Extract plain text from the XHTML content.
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
// These structs map to the XML elements in the EPUB standard.

// ContainerXML represents META-INF/container.xml
type ContainerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

// PackageXML represents the OPF file content (<package> element).
// The OPF file acts as the "brain" of the EPUB, linking everything together.
type PackageXML struct {
	Manifest Manifest `xml:"manifest"`
	Spine    Spine    `xml:"spine"`
}

// Manifest lists *all* resources (HTML, CSS, Images, Fonts) in the EPUB.
// Think of it as an inventory. Each item has a unique 'id' and a 'href' (path).
//
// <manifest>
//
//	<item id="intro" href="intro.xhtml" media-type="application/xhtml+xml"/>
//	<item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
//	<item id="css" href="style.css" media-type="text/css"/>
//
// </manifest>
type Manifest struct {
	Items []Item `xml:"item"`
}

type Item struct {
	ID   string `xml:"id,attr"`
	Href string `xml:"href,attr"` // Path to the file relative to the OPF file
}

// Spine defines the linear *reading order* of the book.
// It does NOT contain file paths directly. Instead, it contains 'itemref' elements
// that point to 'id's in the Manifest.
//
// This allows the book to re-use the same content file in different places if needed,
// or just separates the "order" logic from the "file" logic.
//
// <spine>
//
//	<itemref idref="intro"/>  <!-- First, read the item with id="intro" -->
//	<itemref idref="chap1"/>  <!-- Then, read the item with id="chap1" -->
//
// </spine>
type Spine struct {
	ItemRefs []ItemRef `xml:"itemref"`
}

type ItemRef struct {
	IDRef string `xml:"idref,attr"` // References an 'id' in the Manifest
}

// Helpers

// findOPFPath parses META-INF/container.xml to find the location of the OPF file.
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

// parseOPF reads and unmarshals the OPF file.
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

// readZipFile finds and reads a specific file from the ZIP archive.
func readZipFile(z *zip.ReadCloser, name string) ([]byte, error) {
	// Normalize name to forward slashes just in case we are on Windows.
	name = strings.ReplaceAll(name, "\\", "/")

	for _, f := range z.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(rc)
			// Explicitly close the file without using defer inside a loop
			// to avoid potential resource leaks if the loop logic changes.
			rc.Close()
			return data, err
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// Text Extraction

// extractTextFromXHTML parses XHTML content and extracts human-readable text.
// It skips scripts, styles, and metadata, and preserves block-level structure (newlines).
//
// Compatibility Note:
// This function works for both EPUB 2 (XHTML 1.1) and EPUB 3 (HTML5 XML serialization).
// The core structure (container -> OPF -> Spine) is consistent across versions.
func extractTextFromXHTML(content []byte) (string, error) {

	// Create an XML decoder to parse the content token by token.
	// Unlike DOM parsers that load the whole tree into memory,
	// this stream parser is efficient and low-memory.
	decoder := xml.NewDecoder(bytes.NewReader(content))

	// Configure the decoder for "loose" real-world HTML/XHTML parsing:
	//
	// 1. Strict = false
	//    By default, Go's XML parser is extremely strict. Older EPUBs often have
	//    <!DOCTYPE> declarations with external DTDs that cause the parser to "choke".
	//    Setting Strict=false bypasses DTD validation entirely.
	//    Additionally, it permits unknown HTML entities (like &copy;). Instead of
	//    crashing, the parser safely passes the raw string "&copy;" to our CharData.
	decoder.Strict = false

	// 2. AutoClose
	//    Many HTML tags (like <br> or <hr>) do not have closing tags. Strict XML
	//    will complain about "unexpected EOF" or mismatched tags if we don't
	//    give it a list of "void elements" to close automatically.
	decoder.AutoClose = []string{
		"br", "hr", "img", "meta", "link", "param", "area", "input", "col", "base",
	}

	// 3. Entity map
	//    When Strict=false, unknown entities safely pass into CharData as raw text.
	//    However, mapping frequent ones here provides an excellent performance
	//    shortcut at the lexer level (saving UnescapeString from working later).
	decoder.Entity = map[string]string{
		"nbsp": "\u00A0",
	}

	var buf strings.Builder

	// Tags to ignore content from (we don't want script code or CSS styles).
	ignoreTags := map[string]bool{
		"head": true, "script": true, "style": true, "title": true,
	}
	// Tags that should trigger a visual line break (block-level elements).
	blockTags := map[string]bool{
		"p": true, "div": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"li": true, "blockquote": true, "pre": true, "section": true, "header": true, "footer": true, "article": true,
	}

	ignoreDepth := 0 // Tracks if we are currently inside an ignored tag (like <script>).

	// Iterate through the XML stream one token at a time.
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			return "", err
		}

		// Handle the 3 main types of XML tokens:
		switch t := token.(type) {

		// 1. Start Element: <tag>
		//    We check if we are entering an ignored section (like <script>)
		//    or if we need to insert a newline (like <p>).
		//
		//    NOTE: Self-closing tags (like <br /> or <hr />) are parsed by xml.Decoder
		//    as a StartElement immediately followed by an EndElement.
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if ignoreTags[name] {
				ignoreDepth++
			}
			// If starting a block element (like <p>), ensure we have a newline before it.
			if blockTags[name] {
				buf.WriteString("\n")
			}
			if name == "br" {
				buf.WriteString("\n")
			}

		// 2. End Element: </tag>
		//    We check if we are leaving an ignored section.
		case xml.EndElement:
			name := strings.ToLower(t.Name.Local)
			if ignoreTags[name] {
				ignoreDepth--
			}
			// If ending a block element, ensure we have a newline after it.
			if blockTags[name] {
				buf.WriteString("\n")
			}

		// 3. Character Data: "Some text"
		//    This is the actual content we want to extract.
		case xml.CharData:
			// If we are inside an ignored tag (e.g., <script>), skip this text.
			if ignoreDepth > 0 {
				continue
			}

			// We fetch the text. Because Strict=false, the parser handles numeric
			// references properly (via built-in support) but allows completely
			// unknown entities (like &mdash;) to slip through raw.
			// Example: "&mdash;" passes through unmodified into string(t).
			rawText := string(t)

			// We apply html.UnescapeString as a perfect "catch-all". It safely and
			// accurately decodes all named and numeric entities WITHOUT modifying
			// the XML structure (since the tokenizer already did its job).
			text := html.UnescapeString(rawText)

			// Text Normalization Logic:
			// Goal: "  Hello   \n  World  " -> " Hello World "

			var sb strings.Builder
			// Optimization: Pre-allocate memory for the builder.
			// We know the result won't be larger than the input 'text'.
			// This avoids resizing the buffer multiple times.
			sb.Grow(len(text))

			spaceCount := 0 // Tracks consecutive spaces to collapse them.

			for _, r := range text {
				// Treat newlines, tabs, and returns as spaces.
				if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
					spaceCount++
					// Only write the FIRST space in a sequence.
					if spaceCount == 1 {
						sb.WriteRune(' ')
					}
				} else {
					// It's a non-whitespace character. Write it.
					sb.WriteRune(r)
					// Reset space counter so the next space will be written.
					spaceCount = 0
				}
			}
			text = sb.String()

			if len(text) > 0 {
				buf.WriteString(text)
			}
		}
	}

	// Post-processing: cleanup multiple consecutive newlines.
	result := buf.String()
	// Collapse 3+ newlines into 2 (one blank line).
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(result), nil
}
