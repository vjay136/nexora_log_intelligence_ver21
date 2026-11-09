package parserpolicy

import (
	"regexp"
	"strings"

	"nexora.local/nli-v20/internal/mixedkv"
)

var structuredMailHeader = regexp.MustCompile(`^\s*\d{8}\s+\d{9}[+-]\d{4}\s+\S+\s+\S+\s+\d+\s+\d+\s+\d+\s+[^;\s]+;\S+\s+`)

func framedKVBody(text string) (body string, offset int, framed bool, err error) {
	body, offset, framed, err = mixedkv.UnwrapAssignmentEnvelope(text)
	if err != nil {
		err = fail("incomplete_structured_tail", err.Error())
	}
	return
}

func newCoverageKV(assignments []string) *mixedkv.Parser {
	config := mixedkv.DefaultConfig()
	config.AssignmentSeparators = assignments
	config.DuplicatePolicy = mixedkv.DuplicateArray
	config.BracketPairs["<"] = ">"
	return mixedkv.MustCompile(config)
}

var (
	coverageKV = newCoverageKV([]string{"=", ";"})
	coverageColonKV = newCoverageKV([]string{"=", ":", ";"})
)

// HasFramedMailTail also recognizes a malformed envelope so a fallback
// cannot normalize it after all governed candidates reject the record.
func HasFramedMailTail(input string) bool {
	header := structuredMailHeader.FindStringIndex(input)
	if header == nil {
		return false
	}
	_, _, framed, _ := mixedkv.UnwrapAssignmentEnvelope(input[header[1]:])
	return framed
}

// Validate against raw value spans, so a field name or a similar-looking
// value elsewhere in the record cannot hide an omitted structured pair.
func validateFramedMailTail(input string, fields workingSet) error {
	header := structuredMailHeader.FindStringIndex(input)
	if header == nil {
		return nil
	}
	body, offset, framed, err := framedKVBody(input[header[1]:])
	if err != nil || !framed {
		return err
	}
	scanner := coverageKV
	if first := strings.IndexAny(body, "=:;"); first >= 0 && body[first] == ':' {
		scanner = coverageColonKV
	}
	parsed, err := scanner.Parse(body)
	if err != nil || parsed.RemainderSpan != nil || len(parsed.Entries) == 0 {
		return fail("incomplete_structured_tail", "could not verify complete coverage of the key/value envelope")
	}
	base := header[1] + offset
	captured := make(map[mixedkv.Span]bool, len(fields))
	for name, field := range fields {
		if field.Source && !strings.HasPrefix(name, "_") {
			captured[mixedkv.Span{Start: field.Start, End: field.End}] = true
		}
	}
	for _, entry := range parsed.Entries {
		value := mixedkv.Span{Start: base + entry.ValueSpan.Start, End: base + entry.ValueSpan.End}
		pair := mixedkv.Span{Start: base + entry.PairSpan.Start, End: base + entry.PairSpan.End}
		if !captured[value] && !captured[pair] {
			return fail("incomplete_structured_tail", "key/value envelope contains a value without a corresponding captured source span")
		}
	}
	return nil
}
