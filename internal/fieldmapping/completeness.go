package fieldmapping

import (
	"fmt"
	"regexp"
	"strings"

	"nexora.local/nli-v20/internal/mixedkv"
)

var mailTail = regexp.MustCompile(`^\s*\d{8}\s+\d{9}[+-]\d{4}\s+\S+\s+\S+\s+\d+\s+\d+\s+\d+\s+(?:\d+;)?[^;\s]+;\S+\s+`)

// ValidateComplete prevents a successful partial mapping from hiding structured
// source data. Hidden/omitted fields retain evidence and are intentional choices.
func (c *Compiled) ValidateComplete(raw string, result Result) error {
	if HasMailHeaderExtension(raw) {
		if err := c.validateMailHeader(raw, result); err != nil {
			return err
		}
	}
	spans := []Span{}
	fields := map[string]Field{}
	for _, cf := range c.fields {
		fields[cf.field.Name] = cf.field
	}
	assignments := []string{"="}
	for _, cf := range c.fields {
		f := cf.field
		if f.KVProfile == "" && (f.Method == "key_value" || f.Method == "kv_all") {
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
		f := fields[name]
		if e.Source || f.KVProfile != "" && e.Start >= 0 {
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
	return validateStructuredCoverage(raw, spans, assignments, result)
}

func (c *Compiled) validateMailHeader(raw string, result Result) error {
	evidence := make(map[string]Evidence, len(c.fields))
	for _, compiled := range c.fields {
		field := compiled.field
		source, exists := result.Evidence[field.Name]
		if !exists || field.OutputPolicy == "intermediate" || field.Method == "kv_all" || field.Method == "key_value" ||
			field.Method == "derived" || field.Method == "field" || field.Method == "constant" {
			continue
		}
		for _, operation := range field.Operations {
			if !contains([]string{"split", "select", "substring", "character", "byte", "line", "regex_extract", "first", "last", "range", "before", "after", "between", "token_match", "split_regex", "split_words", "split_tokens"}, operation.Op) {
				continue
			}
			value, ok := result.Values[field.Name].(string)
			if !ok || value == "" || source.Start < 0 || source.End > len(raw) || source.Start > source.End {
				source = Evidence{Start: -1, End: -1}
				break
			}
			region := raw[source.Start:source.End]
			offset := strings.Index(region, value)
			if offset < 0 || offset != strings.LastIndex(region, value) {
				source = Evidence{Start: -1, End: -1}
				break
			}
			source.Start += offset
			source.End = source.Start + len(value)
			source.Spans = nil
			break
		}
		if field.Method == "regex_groups" && field.OutputPolicy == "flatten" {
			for _, key := range result.DynamicFields {
				if compiled.pattern.SubexpIndex(key) > 0 {
					evidence[key] = result.Evidence[key]
				}
			}
			continue
		}
		evidence[field.Name] = source
	}
	return ValidateMailHeader(raw, evidence)
}

// HeaderOmissions returns source evidence for fields whose approved mapping
// explicitly omits them from production output.
func (c *Compiled) HeaderOmissions(result Result) map[string]Evidence {
	var omissions map[string]Evidence
	for _, compiled := range c.fields {
		field := compiled.field
		evidence, exists := result.Evidence[field.Name]
		if !exists || field.Method == "kv_all" || field.Method == "key_value" ||
			field.Method == "derived" || field.Method == "field" || field.Method == "constant" {
			continue
		}
		omitted := field.OutputPolicy == "omit"
		for _, operation := range field.Operations {
			omitted = omitted || operation.Op == "omit" || operation.Op == "exclude"
		}
		if omitted {
			if omissions == nil {
				omissions = make(map[string]Evidence)
			}
			omissions[field.Name] = evidence
		}
	}
	return omissions
}

// ValidateStructuredCoverage checks values, rather than labels, so operators may
// freely rename fields. Unstructured free text is not inferred to be key/value.
func ValidateStructuredCoverage(raw string, spans []Span) error {
	return validateStructuredCoverage(raw, spans, []string{"="}, Result{})
}
func validateStructuredCoverage(raw string, spans []Span, assignments []string, result Result) error {
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
	end := len(strings.TrimRight(raw, " \t\r\n"))
	if header := mailTail.FindStringIndex(raw); header != nil {
		start = header[1]
	}
	if covered(start, end) && start < end {
		return nil
	}
	if start > 0 && strings.HasPrefix(raw[start:], "|") {
		segments, err := mixedkv.ScanPipeTail(raw[start:end])
		if err != nil {
			return fmt.Errorf("%w: %w", ErrIncompleteKV, err)
		}
		evidence := make(map[Span]bool, len(spans))
		for _, span := range spans {
			evidence[span] = true
		}
		for _, segment := range segments {
			value := Span{start + segment.Value.Start, start + segment.Value.End}
			pair := Span{start + segment.Pair.Start, start + segment.Pair.End}
			if !evidence[value] && !evidence[pair] {
				if segment.Key != "" {
					return fmt.Errorf("%w: key %q has no mapped or explicitly omitted pipe segment at byte %d", ErrIncompleteKV, segment.Key, pair.Start+1)
				}
				return fmt.Errorf("%w: no mapped or explicitly omitted pipe segment at byte %d", ErrIncompleteKV, pair.Start+1)
			}
		}
		return nil
	}
	if observations, ok := result.profileObservations(start, end); ok {
		for _, v := range observations {
			if !covered(v.Start, v.End) {
				return fmt.Errorf("incomplete_structured_tail: key %q has no mapped or explicitly omitted source evidence", v.Key)
			}
		}
		return nil
	}
	if start > 0 {
		tail := strings.TrimSpace(raw[start:])
		if strings.ContainsAny(tail, "=:;|") {
			// Even malformed mixed separators are retained for operator repair.
			observations, err := ScanKeyValues(raw[start:], assignments, ":;,| \t\r\n", true)
			if err != nil || len(observations) == 0 {
				return fmt.Errorf("incomplete_structured_tail: unmapped or ambiguous mail payload at byte %d", start+1)
			}
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
