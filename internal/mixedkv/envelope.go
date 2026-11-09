package mixedkv

import (
	"regexp"
	"strings"
)

var framedAssignment = regexp.MustCompile(`^[ \t]*[A-Za-z_][A-Za-z0-9_.-]*[ \t]*[=:;]`)

// UnwrapAssignmentEnvelope recognizes an outer wrapper on a scoped KV source.
// Offsets are relative to the caller's original source.
func UnwrapAssignmentEnvelope(text string) (body string, offset int, framed bool, err error) {
	text = strings.TrimRight(text, " \t\r\n")
	start := len(text) - len(strings.TrimLeft(text, " \t"))
	if start >= len(text) {
		return "", 0, false, nil
	}
	wrapper := strings.IndexByte("[{(<", text[start])
	if wrapper < 0 || !framedAssignment.MatchString(text[start+1:]) {
		return "", 0, false, nil
	}
	if text[len(text)-1] != "]})>"[wrapper] {
		return "", 0, true, parseError(CodeMalformedBracket, len(text), "", "key/value envelope is missing its closing wrapper")
	}
	return text[start+1 : len(text)-1], start + 1, true, nil
}
