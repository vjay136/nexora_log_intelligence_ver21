package fieldmapping

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// KeyValueObservation is source-backed discovery data, never a guessed value.
type KeyValueObservation struct {
	Key        string
	Value      string
	Start, End int
}

// ScanKeyValues exposes the exact native scanner to authoring suggestions.
// complete rejects unexplained prefix/fragments instead of silently skipping.
func ScanKeyValues(raw string, assignments []string, delimiters string, complete bool) ([]KeyValueObservation, error) {
	if len(raw) > MaxInputBytes || !utf8.ValidString(raw) || strings.ContainsRune(raw, 0) {
		return nil, fmt.Errorf("key/value input violates bounds or UTF-8 contract")
	}
	if len(assignments) == 0 {
		assignments = []string{"="}
	}
	if len(assignments) > 16 || len(delimiters) > 64 {
		return nil, fmt.Errorf("key/value separators exceed bounds")
	}
	assignments = append([]string(nil), assignments...)
	for _, s := range assignments {
		if s == "" || len(s) > 8 {
			return nil, fmt.Errorf("invalid assignment separator")
		}
	}
	sort.SliceStable(assignments, func(i, j int) bool { return len(assignments[i]) > len(assignments[j]) })
	if delimiters == "" {
		delimiters = ":;, \t\r\n"
	}
	pairs, err := scanPairsManyMode(raw, assignments, delimiters, complete)
	if err != nil {
		return nil, err
	}
	result := []KeyValueObservation{}
	for key, values := range pairs {
		for _, v := range values {
			result = append(result, KeyValueObservation{key, v.value, v.start, v.end})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Start < result[j].Start })
	return result, nil
}

var foreignAssignment = regexp.MustCompile(`(?:^|[:;|\s])([A-Za-z_][A-Za-z0-9_.-]*)[:;]`)

func strictPairValue(value, delimiters string) error {
	if value == "" {
		return nil
	}
	if value[0] == '\'' || value[0] == '"' {
		if len(value) < 2 || value[len(value)-1] != value[0] {
			return fmt.Errorf("quoted value has trailing unconsumed data")
		}
		return nil
	}
	if value[0] == '{' || value[0] == '[' {
		return nil
	}
	if net.ParseIP(value) != nil || strings.Contains(value, "://") {
		return nil
	}
	var plain strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			i++
			plain.WriteByte(' ')
			continue
		}
		plain.WriteByte(value[i])
	}
	text := plain.String()
	ambiguous := false
	for _, c := range ";|\r\n" {
		if strings.ContainsRune(delimiters, c) && strings.ContainsRune(text, c) {
			ambiguous = true
		}
	}
	for _, match := range foreignAssignment.FindAllString(text, -1) {
		if strings.ContainsRune(delimiters, rune(match[len(match)-1])) {
			ambiguous = true
		}
	}
	if ambiguous {
		return fmt.Errorf("unconsumed or ambiguous key/value fragment; configure its separators or quote the literal value")
	}
	return nil
}

func (r *record) keySource(cf compiledField) (string, int, int, bool, error) {
	f := cf.field
	if f.Source == "" || f.Source == "raw" {
		end := len(r.raw)
		if f.EndOfLine {
			if i := strings.IndexAny(r.raw, "\r\n"); i >= 0 {
				end = i
			}
		}
		return r.raw[:end], 0, end, true, nil
	}
	start, _ := index(f.Source, 1)
	parts, err := units(r.raw, f.Selection, cf.tokenPattern, f.QuoteAware)
	if err != nil {
		return "", 0, 0, false, err
	}
	if start > len(parts) {
		return "", 0, 0, false, nil
	}
	a, b := parts[start-1].start, len(r.raw)
	if f.SourceEnd != "" {
		end, _ := index(f.SourceEnd, start)
		if end > len(parts) {
			return "", 0, 0, false, nil
		}
		b = parts[end-1].end
	}
	selected := r.raw[a:b]
	if f.EndOfLine {
		if i := strings.IndexAny(selected, "\r\n"); i >= 0 {
			b = a + i
			selected = r.raw[a:b]
		}
	}
	if !utf8.ValidString(selected) {
		return "", 0, 0, false, fmt.Errorf("key/value source cuts a Unicode character")
	}
	return selected, a, b, true, nil
}
