package fieldmapping

import (
	"fmt"
	"regexp"
	"strings"
)

var mailTail = regexp.MustCompile(`^\d{8}\s+\d{9}[+-]\d{4}\s+\S+\s+\S+\s+\d+\s+\d+\s+\d+\s+[^;\s]+;\S+\s+`)

// ValidateComplete prevents a successful partial mapping from hiding structured
// source data. Hidden/omitted fields retain evidence and are intentional choices.
func (c *Compiled) ValidateComplete(raw string, result Result) error {
	spans := []Span{}
	fields := map[string]Field{}
	for _, cf := range c.fields {
		fields[cf.field.Name] = cf.field
	}
	assignments := []string{"="}
	for _, cf := range c.fields {
		f := cf.field
		if f.Method == "key_value" || f.Method == "kv_all" {
			for _, s := range f.AssignmentSeparators {
				if !contains(assignments, s) {
					assignments = append(assignments, s)
				}
			}
			if f.KVSeparator != "" && !contains(assignments, f.KVSeparator) {
				assignments = append(assignments, f.KVSeparator)
			}
		}
	}
	for name, e := range result.Evidence {
		if e.Source {
			f := fields[name]
			if f.Method != "key_value" && f.Method != "kv_all" && f.OutputPolicy != "omit" && f.OutputPolicy != "intermediate" {
				narrow, omitted := false, false
				for _, op := range f.Operations {
					if contains([]string{"split", "select", "substring", "character", "byte", "line", "regex_extract", "first", "last", "range", "before", "after", "between", "token_match", "split_regex", "split_words", "split_tokens"}, op.Op) {
						narrow = true
					}
					if op.Op == "omit" || op.Op == "exclude" {
						omitted = true
					}
				}
				if narrow && !omitted && e.Start >= 0 && e.End >= e.Start && e.End <= len(raw) {
					value, ok := result.Values[name].(string)
					if !ok || value == "" {
						continue
					}
					at := strings.Index(raw[e.Start:e.End], value)
					if at < 0 {
						continue
					}
					e.Start += at
					e.End = e.Start + len(value)
					e.Spans = nil
				}
			}
			spans = append(spans, Span{e.Start, e.End})
			spans = append(spans, e.Spans...)
		}
	}
	return validateStructuredCoverage(raw, spans, assignments)
}

// ValidateStructuredCoverage checks values, rather than labels, so operators may
// freely rename fields. Unstructured free text is not inferred to be key/value.
func ValidateStructuredCoverage(raw string, spans []Span) error {
	return validateStructuredCoverage(raw, spans, []string{"="})
}
func validateStructuredCoverage(raw string, spans []Span, assignments []string) error {
	covered := func(start, end int) bool {
		for _, s := range spans {
			if s.Start <= start && s.End >= end && s.Start >= 0 {
				return true
			}
		}
		return false
	}
	start := 0
	if covered(0, len(raw)) {
		return nil
	}
	if header := mailTail.FindStringIndex(raw); header != nil {
		start = header[1]
		tail := strings.TrimSpace(raw[start:])
		if strings.ContainsAny(tail, "=:;|") && !covered(start, len(strings.TrimRight(raw, " \t\r\n"))) {
			// Even malformed mixed separators are retained for operator repair.
			observations, err := ScanKeyValues(raw[start:], assignments, ":;,| \t\r\n", true)
			if err != nil || len(observations) == 0 {
				return fmt.Errorf("incomplete_structured_tail: unmapped or ambiguous mail payload at byte %d", start+1)
			}
		}
		if covered(start, len(strings.TrimRight(raw, " \t\r\n"))) {
			return nil
		}
	}
	observations, err := ScanKeyValues(raw[start:], assignments, ":;,| \t\r\n", false)
	if err != nil {
		return fmt.Errorf("incomplete_structured_tail: %w", err)
	}
	for _, v := range observations {
		if !covered(start+v.Start, start+v.End) {
			return fmt.Errorf("incomplete_structured_tail: key %q has no mapped or explicitly omitted source evidence", v.Key)
		}
	}
	return nil
}
