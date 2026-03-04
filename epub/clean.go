package epub

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// output of pandoc can contain: 
//
// —Si hombres salvajes venir —<2060>dijo<2060>— ellos comerme a mí, vos salvaros.
//
// Those <2060> characters are U+2060 WORD JOINER — an invisible zero-width
// character used in typesetting to prevent line breaks around em-dashes 
//  
// Unicode defines U+2060
//     ↓
// HTML/XHTML can express it as &#x2060;
//     ↓
// Epub (XHTML inside a zip) inherits it
//     ↓
// Pandoc converts to plain text, keeps the raw character
//     ↓
// Your terminal/vim shows it as <2060> because it has no glyph 
//
// There are several Unicode categories of invisible/typesetting characters
// that are all noise for NLP. Unicode defines a formal category Cf (Format
// characters) that covers most of these. You can use it directly
func cleanForNLP(s string) string {
	s = norm.NFC.String(s)

	var b strings.Builder
	for _, r := range s {
        // Strip all Unicode Cf (format) characters - the general solution
        // This covers word joiners, directional marks, soft hyphens, BOM, etc.
        if unicode.Is(unicode.Cf, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
