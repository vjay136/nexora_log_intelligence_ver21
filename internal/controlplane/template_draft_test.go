package controlplane

import (
	"net/http"
	"testing"
)

func TestTemplateDraftFromSamples(t *testing.T) {
	f := newAPIFixture(t)
	smtp := "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 123456;Note;SieveRuleMsg status=failed;reason=not needed"
	card := "20260401 134601279+0100 server001 pabd 250410 0 139764638705408 Note;CARDDAV id=002fac3e:HTTP CHALLENGE REQUEST:method=1:realm=mx:address=11.22.33.44"
	samples := []string{smtp, card}
	const path = "/api/v1/field-mapping/template-draft"
	check := func(kind string) map[string]any {
		t.Helper()
		response, data := f.request(http.MethodPost, path, map[string]any{"kind": kind, "samples": samples})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s returned status %d: %s", kind, response.StatusCode, data)
		}
		body := decodeMap(t, data)
		if body["passed"] != true || body["persisted"] != false || body["activated"] != false {
			t.Fatalf("%s did not return a validated draft: %s", kind, data)
		}
		return body
	}
	header := check("template")
	examples := header["examples"].([]any)
	prefix := examples[0].(map[string]any)["actual"].(map[string]any)
	if prefix["random_number"] != "123456" || prefix["event_type"] != "Note" || prefix["event"] != "SieveRuleMsg" {
		t.Fatalf("incorrect prefix: %v", prefix)
	}
	kv := check("kv_template")
	examples = kv["examples"].([]any)
	first := examples[0].(map[string]any)["actual"].(map[string]any)
	second := examples[1].(map[string]any)["actual"].(map[string]any)
	if first["status"] != "failed" || first["reason"] != "not needed" ||
		second["address"] != "11.22.33.44" || second["messages"].([]any)[0] != "HTTP CHALLENGE REQUEST" {
		t.Fatalf("incorrect KV values: %v / %v", first, second)
	}
	response, data := f.request(http.MethodPost, path, map[string]any{
		"kind": "kv_template", "samples": []string{"status=failed;status=again"}, "config": kv["config"],
	})
	if response.StatusCode != http.StatusOK || decodeMap(t, data)["failed"] != float64(1) {
		t.Fatalf("duplicate source key was not rejected: %d %s", response.StatusCode, data)
	}
}
