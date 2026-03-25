package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Book represents a parsed EPUB.
// It holds a reference to an open *zip.Reader but does not own its lifecycle.
// Callers MUST ensure the underlying zip source (e.g., os.File or memory buffer)
// remains open and valid for the entire lifetime of the Book instance.
// CRITICAL: If you built this from a *zip.ReadCloser (e.g., zip.OpenReader),
// do NOT call Close() on it until you are completely finished calling
// all methods on Book (like Text()).
type Book struct {
	z       *zip.Reader
	Package PackageXML
	opfDir  string // Directory containing the OPF file, for relative path resolution
}

// New creates a new Book from an existing zip reader.
// It parses the metadata immediately. The provided *zip.Reader must remain
// open to allow subsequent calls to methods like Text() to lazily read files.
//
// Example usage with a file:
//
//	r, _ := zip.OpenReader("file.epub")
//	book, _ := epub.New(&r.Reader) // Pass the embedded Reader
//	text, _ := book.Text()         // Must happen BEFORE r.Close()
//	r.Close()
func New(z *zip.Reader) (*Book, error) {
	// Find and parse OPF
	opfPath, err := FindOPFPath(z)
	if err != nil {
		return nil, err
	}

	pkg, err := parseOPF(z, opfPath)
	if err != nil {
		return nil, err
	}

	return &Book{
		z:       z,
		Package: pkg,
		opfDir:  filepath.Dir(opfPath),
	}, nil
}

// getContent finds and reads a specific file from the ZIP archive.
func (b *Book) getContent(name string) ([]byte, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	for _, f := range b.z.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, readErr := io.ReadAll(rc)
			closeErr := rc.Close()
			if readErr != nil {
				return nil, readErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// Metadata Accessors (wrapping the unexported extract* and normalize* helpers)
func (b *Book) creator() string {
	return extractCreator(b.Package.Metadata)
}

func (b *Book) title() string {
	return extractTitle(b.Package.Metadata)
}

func (b *Book) date() string {
	return normalizeDate(extractDatePublication(b.Package.Metadata))
}

func (b *Book) language() string {
	return extractLanguage(b.Package.Metadata)
}

func (b *Book) translator() string {
	return extractTranslator(b.Package.Metadata)
}

// Labels extracts DC metadata as a map of prefix → raw value.
// Only non-empty fields are included. Values are returned without
// normalization — the caller or storage layer is responsible for
// sanitization before persistence.
func (b *Book) Labels() map[string]string {
	labels := make(map[string]string)
	if v := b.creator(); v != "" {
		labels["creator"] = v
	}
	if v := b.title(); v != "" {
		labels["title"] = v
	}
	if v := b.date(); v != "" {
		labels["date"] = v
	}
	if v := b.language(); v != "" {
		labels["language"] = v
	}
	if v := b.translator(); v != "" {
		labels["translator"] = v
	}
	return labels
}

// Text extracts the plain text content of the EPUB in reading order.
func (b *Book) Text() (string, error) {
	// Map Manifest IDs to Hrefs
	idToHref := make(map[string]string)
	for _, item := range b.Package.Manifest.Items {
		idToHref[item.ID] = item.Href
	}

	var fullText strings.Builder

	for _, itemRef := range b.Package.Spine.ItemRefs {
		href, ok := idToHref[itemRef.IDRef]
		if !ok {
			continue
		}

		// Resolve the absolute path within the logical ZIP archive.
		// We use the 'path' package (not 'filepath') because ZIP files use
		// forward slashes regardless of the host operating system.
		fullPath := path.Join(b.opfDir, href)

		content, err := b.getContent(fullPath)
		if err != nil {
			// Try fallback (logic from script)
			content, err = b.getContent(href)
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
		// Use RawToken instead of Token to bypass the strict stack validation
		// (matching start/end tags) which causes errors like "unexpected end element"
		// This often happens in older or poorly generated EPUBs where:
		// * A tag is closed without being opened (orphaned </font>).
		// * Tags are improperly nested (e.g., <b><i>...</b></i>).
		// * The file claims to be XHTML (strict XML) but contains "tag soup" HTML.
		// in malformed XHTML/HTML.
		// RawToken returns the next token from the stream regardless of
		// nesting correctness. For the purpose of extracting plain text
		// (<p>Hello</p> -> "Hello"), structural validation is unnecessary; we
		// only need the character data and the knowledge of block-level tags
		// for spacing.
		token, err := decoder.RawToken()
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
			if blockTags[name] || name == "br" {
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

// FindOPFPath locates the OPF file in the EPUB archive by reading META-INF/container.xml.
func FindOPFPath(r *zip.Reader) (path string, err error) {
	for _, f := range r.File {
		if f.Name == "META-INF/container.xml" {
			rc, openErr := f.Open()
			if openErr != nil {
				return "", openErr
			}
			defer func() {
				err = errors.Join(err, rc.Close())
			}()

			var container ContainerXML
			if decodeErr := xml.NewDecoder(rc).Decode(&container); decodeErr != nil {
				return "", decodeErr
			}

			if len(container.Rootfiles) > 0 {
				return container.Rootfiles[0].FullPath, nil
			}
			break
		}
	}
	return "", fmt.Errorf("could not find OPF path in container.xml")
}

func parseOPF(r *zip.Reader, opfPath string) (pkg PackageXML, err error) {
	for _, f := range r.File {
		if f.Name == opfPath {
			rc, openErr := f.Open()
			if openErr != nil {
				return pkg, openErr
			}
			defer func() {
				err = errors.Join(err, rc.Close())
			}()

			if decodeErr := xml.NewDecoder(rc).Decode(&pkg); decodeErr != nil {
				return pkg, decodeErr
			}
			return pkg, nil
		}
	}
	return pkg, fmt.Errorf("OPF file not found in zip archive")
}

func normalizeDate(val string) string {
	if val == "" {
		return ""
	}
	re := regexp.MustCompile(`\d{4}`)
	return re.FindString(val)
}

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
