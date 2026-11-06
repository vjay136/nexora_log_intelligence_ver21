package aiprovider

import (
	"fmt"
	"strings"
	"testing"
)

func TestInspectStreamProbe(t *testing.T) {
	cases := []struct {
		name, body  string
		usable, ack bool
		code        string
	}{
		{"fragmented final answer", "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\ndata: {\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"private reasoning\"}}]}\n\ndata: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hel\"}}]}\n\ndata: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"lo\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n", true, true, ""},
		{"stop without done", "data: {\"choices\":[{\"delta\":{\"content\":\"ready\"},\"finish_reason\":\"stop\"}]}\n\n", true, true, ""},
		{"done without stop", "data: {\"choices\":[{\"delta\":{\"content\":\"ready\"}}]}\n\ndata: [DONE]\n\n", true, true, ""},
		{"reasoning only", "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"thinking\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n", false, true, "incomplete_output"},
		{"length despite done", "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"},\"finish_reason\":\"length\"}]}\n\ndata: [DONE]\n\n", false, true, "incomplete_output"},
		{"missing terminal", "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n", false, true, "incomplete_output"},
		{"role only done", "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\ndata: [DONE]\n\n", false, true, "incomplete_output"},
		{"whitespace final", "data: {\"choices\":[{\"delta\":{\"content\":\" \\t\"},\"finish_reason\":\"stop\"}]}\n\n", false, true, "incomplete_output"},
		{"malformed first", "data: {broken}\n\n", false, false, "invalid_protocol_response"},
		{"malformed after valid", "data: {\"choices\":[{\"delta\":{\"content\":\"ready\"},\"finish_reason\":\"stop\"}]}\n\ndata: broken\n\n", false, true, "invalid_protocol_response"},
		{"provider error payload", "data: {\"error\":{\"message\":\"secret must not escape\"}}\n\n", false, true, "provider_stream_error"},
		{"provider error event", "event: error\ndata: {\"message\":\"secret must not escape\"}\n\n", false, true, "provider_stream_error"},
		{"finish error", "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"},\"finish_reason\":\"error\"}]}\n\n", false, true, "provider_stream_error"},
		{"usage only", "data: {\"choices\":[],\"usage\":{\"total_tokens\":1}}\n\ndata: [DONE]\n\n", false, false, "invalid_protocol_response"},
		{"keepalive only", ": keepalive\n\nevent: ping\n\n", false, false, "invalid_protocol_response"},
		{"done only", "data: [DONE]\n\n", false, false, "invalid_protocol_response"},
		{"nonstream JSON", `{"choices":[{"message":{"content":"ready"}}]}`, false, false, "invalid_protocol_response"},
		{"wrong delta type", "data: {\"choices\":[{\"delta\":{\"content\":42}}]}\n\n", false, false, "invalid_protocol_response"},
		{"cross choice terminal", "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"},{\"index\":1,\"delta\":{\"content\":\"unfinished\"}}]}\n\n", false, true, "incomplete_output"},
		{"content after done", "data: {\"choices\":[{\"delta\":{\"content\":\"ready\"}}]}\n\ndata: [DONE]\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"late\"}}]}\n\n", false, true, "invalid_protocol_response"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			usable, ack, code := InspectStreamProbe([]byte(tc.body))
			if usable != tc.usable || ack != tc.ack || code != tc.code {
				t.Fatalf("got usable=%v acknowledged=%v code=%q", usable, ack, code)
			}
		})
	}
}

func TestInspectStreamProbeMultilineUsageAndLineEndings(t *testing.T) {
	body := ": keepalive\n\nid: 17\nevent: completion\ndata: {\"choices\":[\ndata: {\"index\":0,\"delta\":{\"content\":\"ready\"},\"finish_reason\":\"stop\"}\ndata: ]}\n\ndata: {\"choices\":[],\"usage\":{\"total_tokens\":2}}\n\ndata: [DONE]"
	for _, ending := range []string{"\n", "\r\n", "\r"} {
		t.Run(fmt.Sprintf("ending-%q", ending), func(t *testing.T) {
			data := []byte("\xef\xbb\xbf" + strings.ReplaceAll(body, "\n", ending))
			usable, ack, code := InspectStreamProbe(data)
			if !usable || !ack || code != "" {
				t.Fatalf("valid multiline stream rejected: usable=%v ack=%v code=%s", usable, ack, code)
			}
		})
	}
}

func TestInspectStreamProbeBoundsAndStaticFailures(t *testing.T) {
	for _, data := range [][]byte{nil, []byte(strings.Repeat("x", maxStreamProbeBytes+1)), []byte{0xff}, []byte("data: {\"choices\":[{\"index\":128,\"delta\":{}}]}\n\n")} {
		usable, _, code := InspectStreamProbe(data)
		if usable || code != "invalid_protocol_response" {
			t.Fatal("invalid bounded input was accepted")
		}
	}
	body := strings.Repeat("data: {\"choices\":[{\"delta\":{}}]}\n\n", maxStreamProbeEvents+1)
	usable, ack, code := InspectStreamProbe([]byte(body))
	if usable || !ack || code != "invalid_protocol_response" {
		t.Fatal("event count bound was not enforced")
	}
}

func FuzzInspectStreamProbeNeverPanics(f *testing.F) {
	for _, input := range []string{"data: [DONE]\n\n", "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n", "event: error\ndata: {}\n\n"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 16<<10 {
			return
		}
		usable, ack, code := InspectStreamProbe([]byte(input))
		if usable && (!ack || code != "") {
			t.Fatal("usable result has contradictory status")
		}
		if code != "" && code != "incomplete_output" && code != "invalid_protocol_response" && code != "provider_stream_error" {
			t.Fatal("nonstatic error code")
		}
	})
}
