package mixedkv

import "strings"

// PipeField holds the raw segment and value ranges for one field in a
// pipe-prefixed tail. An empty Key represents an unnamed positional segment.
type PipeField struct {
	Key   string
	Pair  Span
	Value Span
	Text  string
}

var pipeTailParser = func() *Parser {
	config := DefaultConfig()
	config.PairSeparators = []string{"|"}
	config.AssignmentSeparators = []string{"=>", ":=", "=", ":", ";"}
	return MustCompile(config)
}()

// ScanPipeTail recognizes pipe-prefixed mail tails, preserving source offsets
// and each occurrence of a repeated key. Escaped, quoted and nested pipes
// belong to the current field.
func ScanPipeTail(text string) ([]PipeField, error) {
	return pipeTailParser.ScanPipeTail(text)
}

// ScanPipeTail uses the configured assignment, quote, and bracket rules. Its
// pair separator must be the single literal pipe.
func (p *Parser) ScanPipeTail(text string) ([]PipeField, error) {
	if len(text) > p.config.MaxInputBytes {
		return nil, parseError(CodeInputTooLarge, p.config.MaxInputBytes, "", "pipe tail exceeds configured maximum")
	}
	if !strings.HasPrefix(text, "|") {
		return nil, parseError(CodeNoKeyValue, 0, "", "pipe tail must start with a pipe")
	}
	var fields []PipeField
	for start := skipSpace(text, 1); start < len(text); {
		h, named := p.headerAt(text, start)
		valueStart := start
		if named {
			valueStart = h.valueStart
		}
		boundary, err := p.findBoundary(text, valueStart, "", true)
		if err != nil {
			return nil, err
		}
		end, next := len(text), len(text)
		if boundary != nil {
			end, next = boundary.start, boundary.next
		}
		valueStart, valueEnd := p.trimValueBounds(text, valueStart, end)
		if valueEnd-valueStart > p.config.MaxValueBytes {
			return nil, parseError(CodeValueTooLarge, valueStart, h.key, "pipe field value exceeds configured maximum")
		}
		if len(fields) >= p.config.MaxFields {
			return nil, parseError(CodeTooManyFields, start, h.key, "pipe tail exceeds configured field count")
		}
		if valueStart == valueEnd && !named {
			return nil, parseError(CodeEmptyValue, start, "", "empty positional pipe segment is ambiguous")
		}
		if valueStart == valueEnd && p.config.RejectEmptyValues {
			return nil, parseError(CodeEmptyValue, start, h.key, "empty values are disabled")
		}
		decoded, err := p.decodeValue(text[valueStart:valueEnd])
		if err != nil {
			return nil, parseError(CodeDanglingEscape, valueStart, h.key, err.Error())
		}
		fields = append(fields, PipeField{
			Key: h.key, Pair: Span{Start: start, End: end}, Value: Span{Start: valueStart, End: valueEnd}, Text: decoded,
		})
		start = next
	}
	if len(fields) == 0 {
		return nil, parseError(CodeNoKeyValue, 0, "", "pipe tail contains no fields")
	}
	return fields, nil
}
