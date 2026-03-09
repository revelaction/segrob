package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// Labels extracts DC metadata from an epub zip, normalizes values, and
// returns a comma-separated string of labels in "key:value" format.
// Example: "creator:garcia_marquez,title:cien_anos_de_soledad,date:1967,language:es"
func Labels(r *zip.Reader) (string, error) {
	opfPath, err := findOPFPath(r)
	if err != nil {
		return "", err
	}

	pkg, err := parseOPF(r, opfPath)
	if err != nil {
		return "", err
	}

	m := pkg.Metadata
	var labels []string

	if v := normalizeValue(extractCreator(m)); v != "" {
		labels = append(labels, "creator:"+v)
	}
	if v := normalizeValue(extractTitle(m)); v != "" {
		labels = append(labels, "title:"+v)
	}
	if v := normalizeDate(extractDatePublication(m)); v != "" {
		labels = append(labels, "date:"+v)
	}
	if v := normalizeValue(extractLanguage(m)); v != "" {
		labels = append(labels, "language:"+v)
	}
	if v := normalizeValue(extractTranslator(m)); v != "" {
		labels = append(labels, "translator:"+v)
	}

	return strings.Join(labels, ","), nil
}

func findOPFPath(r *zip.Reader) (string, error) {
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

func parseOPF(r *zip.Reader, opfPath string) (PackageXML, error) {
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
