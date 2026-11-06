package fieldmapping

import (
	"fmt"
	"strings"
)

// splitParts is a bounded literal-delimiter scanner. Quoted fields permit
// delimiters and doubled quotation marks; backslash-escaped boundaries are
// preserved as values. Offsets always refer to the original input bytes.
func splitParts(text, delimiter string, quoted bool) ([]pair, error) {
	if delimiter == "" {
		return nil, fmt.Errorf("delimiter must be nonempty")
	}
	parts := []pair{}
	start := 0
	quote := byte(0)
	closed := false
	appendPart := func(end int) error {
		if len(parts) >= MaxTokens {
			return fmt.Errorf("split exceeds part limit")
		}
		value := text[start:end]
		if quoted {
			trimmed := strings.TrimSpace(value)
			if len(trimmed) >= 2 && (trimmed[0] == '"' || trimmed[0] == '\'') && trimmed[len(trimmed)-1] == trimmed[0] {
				q := trimmed[0]
				value = trimmed[1 : len(trimmed)-1]
				value = strings.ReplaceAll(value, string([]byte{q, q}), string(q))
			}
			var b strings.Builder
			for i := 0; i < len(value); i++ {
				if value[i] == '\\' && i+1 < len(value) && (value[i+1] == '\\' || value[i+1] == '\'' || value[i+1] == '"' || strings.HasPrefix(value[i+1:], delimiter)) {
					i++
				}
				b.WriteByte(value[i])
			}
			value = b.String()
		}
		parts = append(parts, pair{value, start, end})
		return nil
	}
	for i := 0; i < len(text); i++ {
		if quoted && text[i] == '\\' && i+1 < len(text) {
			i++
			continue
		}
		if quote != 0 {
			if text[i] == quote {
				if i+1 < len(text) && text[i+1] == quote {
					i++
					continue
				}
				quote = 0
				closed = true
			}
			continue
		}
		if strings.HasPrefix(text[i:], delimiter) {
			if err := appendPart(i); err != nil {
				return nil, err
			}
			i += len(delimiter) - 1
			start = i + 1
			closed = false
			continue
		}
		if quoted && (text[i] == '"' || text[i] == '\'') && strings.TrimSpace(text[start:i]) == "" {
			quote = text[i]
			continue
		}
		if closed && text[i] != ' ' && text[i] != '\t' && text[i] != '\r' && text[i] != '\n' {
			return nil, fmt.Errorf("unexpected data after closing delimiter quote")
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated delimiter quote")
	}
	if err := appendPart(len(text)); err != nil {
		return nil, err
	}
	return parts, nil
}
