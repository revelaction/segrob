package epub

import "encoding/xml"

// ContainerXML represents META-INF/container.xml
type ContainerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

// PackageXML represents the OPF file content (<package> element).
// It acts as the "brain" of the EPUB, linking everything together.
type PackageXML struct {
	Metadata Metadata `xml:"metadata"`
	Manifest Manifest `xml:"manifest"`
	Spine    Spine    `xml:"spine"`
}

type Metadata struct {
	Titles       []string  `xml:"title"`
	Creators     []Creator `xml:"creator"`
	Contributors []Creator `xml:"contributor"`
	Dates        []Date    `xml:"date"`
	Language     []string  `xml:"language"`
	Description  []string  `xml:"description"`
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

// Manifest lists *all* resources (HTML, CSS, Images, Fonts) in the EPUB.
// Think of it as an inventory. Each item has a unique 'id' and a 'href' (path).
//
// <manifest>
//   <item id="intro" href="intro.xhtml" media-type="application/xhtml+xml"/>
//   <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
//   <item id="css" href="style.css" media-type="text/css"/>
// </manifest>
type Manifest struct {
	Items []Item `xml:"item"`
}

type Item struct {
	ID   string `xml:"id,attr"`
	Href string `xml:"href,attr"` // Path relative to OPF
}

// Spine defines the linear *reading order* of the book.
// It does NOT contain file paths directly. Instead, it contains 'itemref' elements
// that point to 'id's in the Manifest.
//
// This allows the book to re-use the same content file in different places if needed,
// or just separates the "order" logic from the "file" logic.
//
// <spine>
//   <itemref idref="intro"/>  <!-- First, read the item with id="intro" -->
//   <itemref idref="chap1"/>  <!-- Then, read the item with id="chap1" -->
// </spine>
type Spine struct {
	ItemRefs []ItemRef `xml:"itemref"`
}

type ItemRef struct {
	IDRef string `xml:"idref,attr"` // References an ID in the Manifest
}
