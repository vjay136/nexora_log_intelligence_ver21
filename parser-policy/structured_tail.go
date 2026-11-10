package parserpolicy

import (
	"context"
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
	coverageKV      = newCoverageKV([]string{"=", ";"})
	coverageColonKV = newCoverageKV([]string{"=", ":", ";"})
)

// HasFramedMailTail also recognizes a malformed envelope or pipe tail so a fallback
// cannot normalize it after all governed candidates reject the record.
func HasFramedMailTail(input string) bool {
	header := structuredMailHeader.FindStringIndex(input)
	if header == nil {
		return false
	}
	tail := input[header[1]:]
	if strings.HasPrefix(tail, "|") {
		return true
	}
	_, _, framed, _ := mixedkv.UnwrapAssignmentEnvelope(tail)
	return framed
}

// Validate against raw value spans, so a field name or a similar-looking
// value elsewhere in the record cannot hide an omitted structured pair.
func validateFramedMailTail(input string, fields workingSet, projected map[string]interface{}) error {
	header := structuredMailHeader.FindStringIndex(input)
	if header == nil {
		return nil
	}
	base := header[1]
	tail := input[base:]
	pipe := strings.HasPrefix(tail, "|")
	uncodedSieve := uncodedSieveHeader.MatchString(input) && !pipe && !hasMailContinuation(tail)
	body, offset, framed, err := framedKVBody(tail)
	if !uncodedSieve && (err != nil || !framed && !pipe) {
		return err
	}
	captured := make(map[mixedkv.Span]bool, len(fields))
	for name := range projected {
		field := fields[name]
		if field.Source {
			captured[mixedkv.Span{Start: field.Start - base, End: field.End - base}] = true
			for _, span := range field.SourceSpans {
				captured[mixedkv.Span{Start: span.Start - base, End: span.End - base}] = true
			}
		}
	}
	if uncodedSieve {
		parsed, err := parseUncodedSieveTail(context.Background(), tail)
		if err != nil {
			return fail("incomplete_structured_tail", "could not verify the uncoded mail grammar")
		}
		for _, entry := range parsed.Entries {
			if !captured[entry.ValueSpan] {
				return fail("incomplete_structured_tail", "uncoded mail payload contains an unmapped value")
			}
		}
		return nil
	}
	if pipe {
		segments, err := mixedkv.ScanPipeTail(tail)
		if err != nil {
			return fail("incomplete_structured_tail", err.Error())
		}
		for _, segment := range segments {
			if !captured[segment.Value] && !captured[segment.Pair] {
				return fail("incomplete_structured_tail", "pipe-delimited payload contains a segment without a corresponding captured source span")
			}
		}
		return nil
	}
	scanner := coverageKV
	if first := strings.IndexAny(body, "=:;"); first >= 0 && body[first] == ':' {
		scanner = coverageColonKV
	}
	parsed, err := scanner.Parse(body)
	if err != nil || parsed.RemainderSpan != nil || len(parsed.Entries) == 0 {
		return fail("incomplete_structured_tail", "could not verify complete coverage of the key/value envelope")
	}
	for _, entry := range parsed.Entries {
		value := mixedkv.Span{Start: offset + entry.ValueSpan.Start, End: offset + entry.ValueSpan.End}
		pair := mixedkv.Span{Start: offset + entry.PairSpan.Start, End: offset + entry.PairSpan.End}
		if !captured[value] && !captured[pair] {
			return fail("incomplete_structured_tail", "key/value envelope contains a value without a corresponding captured source span")
		}
	}
	return nil
}
