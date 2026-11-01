// Package canonicaljson provides the storage- and transport-stable JSON
// representation used by both the controller and client for signed digests.
package canonicaljson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Normalize returns a deterministic semantic representation. Object keys are
// sorted, array order is preserved, and equivalent JSON numbers share a single
// coefficient-and-exponent spelling without float conversion.
func Normalize(raw []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return nil, errors.New("multiple JSON values")
	} else if !errors.Is(err, io.EOF) {
		return nil, err
	}
	var out bytes.Buffer
	if err := writeValue(&out, value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func writeValue(out *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		out.WriteString("null")
	case bool:
		if typed {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case string:
		encoded, _ := json.Marshal(typed)
		out.Write(encoded)
	case json.Number:
		normalized, err := normalizeNumber(typed.String())
		if err != nil {
			return err
		}
		out.WriteString(normalized)
	case []any:
		out.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				out.WriteByte(',')
			}
			if err := writeValue(out, item); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				out.WriteByte(',')
			}
			encoded, _ := json.Marshal(key)
			out.Write(encoded)
			out.WriteByte(':')
			if err := writeValue(out, typed[key]); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	default:
		return fmt.Errorf("unsupported JSON value %T", value)
	}
	return nil
}

func normalizeNumber(input string) (string, error) {
	text := strings.ToLower(strings.TrimSpace(input))
	negative := strings.HasPrefix(text, "-")
	if negative {
		text = strings.TrimPrefix(text, "-")
	}
	exponent := int64(0)
	if index := strings.IndexByte(text, 'e'); index >= 0 {
		parsed, err := strconv.ParseInt(text[index+1:], 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid JSON number %q", input)
		}
		if parsed < -100_000 || parsed > 100_000 {
			return "", errors.New("JSON number exponent is outside the supported canonical range")
		}
		exponent = parsed
		text = text[:index]
	}
	fractionDigits := int64(0)
	if index := strings.IndexByte(text, '.'); index >= 0 {
		fractionDigits = int64(len(text) - index - 1)
		text = text[:index] + text[index+1:]
	}
	if text == "" {
		return "", fmt.Errorf("invalid JSON number %q", input)
	}
	for _, char := range text {
		if char < '0' || char > '9' {
			return "", fmt.Errorf("invalid JSON number %q", input)
		}
	}
	text = strings.TrimLeft(text, "0")
	if text == "" {
		return "0", nil
	}
	power := exponent - fractionDigits
	for strings.HasSuffix(text, "0") {
		text = strings.TrimSuffix(text, "0")
		power++
	}
	sign := ""
	if negative {
		sign = "-"
	}
	const maxCanonicalNumberBytes = 1 << 20
	if power >= 0 {
		if int64(len(text))+power > maxCanonicalNumberBytes {
			return "", errors.New("canonical JSON number exceeds size bound")
		}
		return sign + text + strings.Repeat("0", int(power)), nil
	}
	point := int64(len(text)) + power
	if point > 0 {
		return sign + text[:point] + "." + text[point:], nil
	}
	if 2-point+int64(len(text)) > maxCanonicalNumberBytes {
		return "", errors.New("canonical JSON number exceeds size bound")
	}
	return sign + "0." + strings.Repeat("0", int(-point)) + text, nil
}
