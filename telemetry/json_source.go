package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrTrailingJSON = errors.New("trailing JSON content")

// DecodeJSONObject preserves standard JSON decoding and records the exact value
// offsets of top-level source fields. Repeated keys retain the last occurrence.
func DecodeJSONObject(input string) (map[string]any, map[string]SourceSpan, error) {
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil {
		return nil, nil, err
	}
	if first != json.Delim('{') {
		return nil, nil, fmt.Errorf("JSON log must be an object")
	}
	values := map[string]any{}
	spans := map[string]SourceSpan{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, nil, err
		}
		name, ok := token.(string)
		if !ok {
			return nil, nil, fmt.Errorf("JSON object key must be a string")
		}
		start := int(decoder.InputOffset())
		for start < len(input) && strings.ContainsRune(" :\t\r\n", rune(input[start])) {
			start++
		}
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, nil, err
		}
		values[name] = value
		spans[name] = SourceSpan{Start: start, End: int(decoder.InputOffset())}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, nil, ErrTrailingJSON
	}
	return values, spans, nil
}
