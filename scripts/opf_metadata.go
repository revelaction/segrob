package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/revelaction/segrob/epub"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file.epub>\n", os.Args[0])
		os.Exit(1)
	}

	epubPath := os.Args[1]

	z, err := zip.OpenReader(epubPath)
	if err != nil {
		fmt.Printf("[%s]\n", filepath.Base(epubPath))
		fmt.Println("  Error: Not a valid zip/epub file.")
		fmt.Println(strings.Repeat("-", 40))
		os.Exit(1)
	}
	defer z.Close()

	opfPath, err := epub.FindOPFPath(&z.Reader)
	if err != nil {
		// Fallback: search for first .opf file
		for _, f := range z.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".opf") {
				opfPath = f.Name
				break
			}
		}
	}

	if opfPath == "" {
		fmt.Println("  Error: Could not locate OPF file.")
		fmt.Println(strings.Repeat("-", 40))
		os.Exit(1)
	}

	var opfContent []byte
	for _, f := range z.File {
		if f.Name == opfPath {
			rc, openErr := f.Open()
			if openErr != nil {
				fmt.Printf("  Error reading OPF file: %v\n", openErr)
				fmt.Println(strings.Repeat("-", 40))
				os.Exit(1)
			}
			opfContent, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				fmt.Printf("  Error reading OPF file content: %v\n", err)
				fmt.Println(strings.Repeat("-", 40))
				os.Exit(1)
			}
			break
		}
	}

	if len(opfContent) == 0 {
		fmt.Println("  Error: OPF file empty or not found in zip structure.")
		fmt.Println(strings.Repeat("-", 40))
		os.Exit(1)
	}

	// We can simply unmarshal the parts of the OPF we care about,
	// and capture the raw inner XML of the <metadata> block.
	type OPF struct {
		Version  string `xml:"version,attr"`
		Metadata struct {
			Inner string `xml:",innerxml"`
		} `xml:"metadata"`
	}

	var opf OPF
	if err := xml.Unmarshal(opfContent, &opf); err != nil {
		fmt.Printf("  XML Parsing Error: %v\n", err)
		os.Exit(1)
	}

	version := opf.Version
	if version == "" {
		version = "Unknown"
	}
	fmt.Printf("%s - epub version: %s\n", filepath.Base(epubPath), version)

	if opf.Metadata.Inner != "" {
		// Output the Inner XML directly. We wrap it in `<metadata>` for standard viewing formatting.
		// NOTE: Any original namespace attributes on <metadata> (like xmlns:dc) are omitted here for simplicity.
		fmt.Printf("<metadata>\n%s\n</metadata>\n", strings.TrimSpace(opf.Metadata.Inner))
	}
}
