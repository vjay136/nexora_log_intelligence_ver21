package aiprovider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const (
	maxStreamProbeBytes   = 1 << 20
	maxStreamProbeEvents  = 4096
	maxStreamProbeChoices = 128
)

type streamProbeChoice struct {
	Index        *int            `json:"index"`
	Delta        json.RawMessage `json:"delta"`
	FinishReason *string         `json:"finish_reason"`
}
type streamProbeDelta struct {
	Content          *string `json:"content"`
	ReasoningContent *string `json:"reasoning_content"`
	Role             *string `json:"role"`
}
type streamProbeState struct{ text, stop bool }

// InspectStreamProbe checks an OpenAI-compatible SSE connection probe without
// retaining or returning its output. Protocol acknowledgement is deliberately
// separate from usable final text: reasoning, truncation and provider errors
// must never make a connection test appear to have generated a complete answer.
// The HTTP caller already bounds its body; this additional cap also protects
// programmatic callers. All returned error codes are static and body-free.
func InspectStreamProbe(data []byte) (usable, acknowledged bool, errorCode string) {
	if len(data) == 0 || len(data) > maxStreamProbeBytes || !utf8.Valid(data) {
		return false, false, "invalid_protocol_response"
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), maxStreamProbeBytes+1)
	var payload bytes.Buffer
	eventName := ""
	events := 0
	done := false
	failedFinish := false
	states := map[int]streamProbeState{}
	process := func() string {
		defer payload.Reset()
		name := eventName
		eventName = ""
		content := bytes.TrimSpace(payload.Bytes())
		if name == "error" {
			acknowledged = true
			return "provider_stream_error"
		}
		if len(content) == 0 {
			return ""
		}
		events++
		if events > maxStreamProbeEvents {
			return "invalid_protocol_response"
		}
		if bytes.Equal(content, []byte("[DONE]")) {
			done = true
			return ""
		}
		var envelope map[string]json.RawMessage
		if json.Unmarshal(content, &envelope) != nil || envelope == nil {
			return "invalid_protocol_response"
		}
		if failure, exists := envelope["error"]; exists && !bytes.Equal(bytes.TrimSpace(failure), []byte("null")) {
			acknowledged = true
			return "provider_stream_error"
		}
		encodedChoices, exists := envelope["choices"]
		encodedChoices = bytes.TrimSpace(encodedChoices)
		if !exists || len(encodedChoices) == 0 || encodedChoices[0] != '[' {
			return "invalid_protocol_response"
		}
		var choices []streamProbeChoice
		if json.Unmarshal(encodedChoices, &choices) != nil || len(choices) > maxStreamProbeChoices {
			return "invalid_protocol_response"
		}
		// A final usage event may legally contain choices:[]; it does not by
		// itself prove that a model acknowledged or generated this request.
		if len(choices) == 0 {
			return ""
		}
		deltas := make([]streamProbeDelta, len(choices))
		indices := make([]int, len(choices))
		for i, choice := range choices {
			indices[i] = i
			if choice.Index != nil {
				indices[i] = *choice.Index
			}
			if indices[i] < 0 || indices[i] >= maxStreamProbeChoices {
				return "invalid_protocol_response"
			}
			delta := bytes.TrimSpace(choice.Delta)
			if len(delta) == 0 || bytes.Equal(delta, []byte("null")) {
				if choice.FinishReason == nil {
					return "invalid_protocol_response"
				}
			} else if delta[0] != '{' || json.Unmarshal(delta, &deltas[i]) != nil {
				return "invalid_protocol_response"
			}
		}
		acknowledged = true
		if done {
			return "invalid_protocol_response"
		}
		for i, choice := range choices {
			state := states[indices[i]]
			if deltas[i].Content != nil && strings.TrimSpace(*deltas[i].Content) != "" {
				state.text = true
			}
			if choice.FinishReason != nil {
				switch strings.ToLower(strings.TrimSpace(*choice.FinishReason)) {
				case "stop":
					state.stop = true
				case "":
				case "error":
					return "provider_stream_error"
				default:
					failedFinish = true
				}
			}
			states[indices[i]] = state
		}
		return ""
	}
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			if code := process(); code != "" {
				return false, acknowledged, code
			}
			continue
		}
		if line[0] == ':' {
			continue
		}
		field, value, found := bytes.Cut(line, []byte(":"))
		if !found {
			value = nil
		}
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}
		switch string(field) {
		case "data":
			if payload.Len() > 0 {
				payload.WriteByte('\n')
			}
			payload.Write(value)
		case "event":
			eventName = strings.TrimSpace(string(value))
		}
	}
	if scanner.Err() != nil {
		return false, acknowledged, "invalid_protocol_response"
	}
	if code := process(); code != "" {
		return false, acknowledged, code
	}
	if !acknowledged {
		return false, false, "invalid_protocol_response"
	}
	if !failedFinish {
		for _, state := range states {
			if state.text && (done || state.stop) {
				return true, true, ""
			}
		}
	}
	return false, true, "incomplete_output"
}
