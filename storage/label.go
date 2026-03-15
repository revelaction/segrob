package storage

import (
	"maps"
	"slices"
	"strings"
)

// NormalizeLabel sanitizes a single label value for safe storage.
//
// The corpus "labels" column uses comma-separated values (CSV). To prevent
// corruption, commas are stripped. Additionally, labels are lowercased and
// whitespace/hyphens are replaced with underscores for consistent matching
// and display.
//
// This function must be called at every boundary where labels enter storage:
//   - CorpusStore.AddLabel (CLI path)
//   - SerializeLabels (epub ingest path)
func NormalizeLabel(val string) string {
	if val == "" {
		return ""
	}
	val = strings.ReplaceAll(val, ", ", "_") // "márquez, gabriel" → "márquez_gabriel"
	val = strings.ReplaceAll(val, ",", "")   // any remaining bare commas
	val = strings.ToLower(val)
	val = strings.ReplaceAll(val, " ", "_")
	val = strings.ReplaceAll(val, "-", "_")
	return val
}

// SerializeLabels converts a map of prefix → raw value into the
// comma-separated "prefix:normalized_value" format used by the corpus labels
// column. Empty values are skipped. The result is sorted alphabetically for
// deterministic output.
func SerializeLabels(m map[string]string) string {
	var labels []string
	for _, prefix := range slices.Sorted(maps.Keys(m)) {
		val := NormalizeLabel(m[prefix])
		if val != "" {
			labels = append(labels, prefix+":"+val)
		}
	}
	return strings.Join(labels, ",")
}
