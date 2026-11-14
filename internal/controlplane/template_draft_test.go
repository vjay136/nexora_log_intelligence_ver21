package controlplane

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/fieldmapping"
	"nexora.local/nli-v20/internal/mixedkv"
)

func TestTemplateDraftPreservesTagsOnBothSidesOfLevel(t *testing.T) {
	f := newAPIFixture(t)
	const head = "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 "
	const record = head + "123456;edge;Note;efgh;12345;SieveRuleMsg status=failed;reason=not needed"
	status, payload := f.request(http.MethodPost, "/api/v1/field-mapping/template-draft", map[string]any{
		"kind": "template", "samples": []string{record},
	})
	if status.StatusCode != http.StatusOK {
		t.Fatalf("template draft returned %d: %s", status.StatusCode, payload)
	}
	body := decodeMap(t, payload)
	if body["passed"] != true || body["complete_records"] != false || body["records_requiring_kv_template"] != float64(1) {
		t.Fatalf("incorrect header and KV status: %s", payload)
	}
	config := body["config"].(map[string]any)
	if len(config["optional_fields"].([]any)) != 0 {
		t.Fatalf("present fields must be required: %v", config["optional_fields"])
	}
	preview := body["examples"].([]any)[0].(map[string]any)["actual"].(map[string]any)
	for key, want := range map[string]string{
		"extra_tag1": "edge", "event_type": "Note", "extra_tag2": "efgh", "extra_tag3": "12345", "event": "SieveRuleMsg",
	} {
		if preview[key] != want {
			t.Errorf("%s = %v, want %s", key, preview[key], want)
		}
	}
	if _, err := validateAIWorkspaceAsset("template", head+"123456;edge;Note;efgh;SieveRuleMsg", config); err == nil {
		t.Fatal("template passed with a required tag missing")
	}
	headerFields := make(map[string]string)
	for name, value := range config["fields"].(map[string]any) {
		headerFields[name] = value.(string)
	}
	plan, err := composeTemplateMapping(templateComposeHeader{
		Engine: "re2", Pattern: config["pattern"].(string), Fields: headerFields,
	}, mixedkv.Config{PairSeparators: []string{";"}, AssignmentSeparators: []string{"="}})
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := fieldmapping.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := compiled.Execute(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.ValidateComplete(record, parsed); err != nil {
		t.Fatal(err)
	}
	tail, ok := parsed.Values["kv_pair"].(map[string]any)
	if !ok || tail["reason"] != "not needed" || tail["status"] != "failed" || parsed.Values["extra_tag3"] != "12345" {
		t.Fatalf("wrong combined preview: %v", parsed.Values)
	}
	config["fields"].(map[string]any)["event_type"], config["fields"].(map[string]any)["extra_tag2"] = "extra_tag2", "event_type"
	if _, err := validateAIWorkspaceAsset("template", record, config); err == nil {
		t.Fatal("template passed with level and tag reversed")
	}
}

func TestTemplateDraftFromSamples(t *testing.T) {
	f := newAPIFixture(t)
	smtp := "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 123456;abcd;Note;SieveRuleMsg status=failed;reason=not needed"
	card := "20260401 134601279+0100 server001 pabd 250410 0 139764638705408 Note;CARDDAV id=002fac3e:HTTP CHALLENGE REQUEST:method=1:realm=mx:address=11.22.33.44"
	const generated = "20261113 120101234+0530 mail-qa-002 smtp 501 7 1234567890 98765;edge-west;tenant-blue;Warn;MailDeferred(450/1) status=queued;reason=retry later;server=mx-qa-02"
	samples := []string{smtp, card, strings.Replace(smtp, "abcd;", "abcd;efgh;", 1), generated}
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
	if got := examples[0].(map[string]any)["tail_start_byte"]; got != float64(strings.Index(smtp, "status=")) {
		t.Fatalf("KV boundary does not point to the first assignment: %v", got)
	}
	prefix := examples[0].(map[string]any)["actual"].(map[string]any)
	if prefix["extra_tag1"] != "abcd" || prefix["random_number"] != "123456" || prefix["event_type"] != "Note" || prefix["event"] != "SieveRuleMsg" {
		t.Fatalf("incorrect prefix: %v", prefix)
	}
	if tagged := examples[2].(map[string]any)["actual"].(map[string]any); tagged["extra_tag1"] != "abcd" || tagged["extra_tag2"] != "efgh" || tagged["event"] != "SieveRuleMsg" {
		t.Fatalf("incorrect numbered fields: %v", tagged)
	}
	if fresh := examples[3].(map[string]any)["actual"].(map[string]any); fresh["extra_tag1"] != "edge-west" || fresh["extra_tag2"] != "tenant-blue" || fresh["event_code"] != "450/1" {
		t.Fatalf("new header failed: %v", fresh)
	}
	custom := header["config"].(map[string]any)
	custom["fields"].(map[string]any)["extra_tag1"] = "delivery_tag"
	custom["optional_fields"] = []any{"random_number", "event_code", "delivery_tag", "extra_tag2"}
	if verified, err := validateAIWorkspaceAsset("template", smtp, custom); err != nil || verified["actual"].(map[string]any)["delivery_tag"] != "abcd" {
		t.Fatalf("renamed header field failed: %v, %v", verified, err)
	}
	legacy := aiNativeMailAsset("template", card)
	delete(legacy["fields"].(map[string]any), "extra_tag1")
	legacy["optional_fields"] = []any{"random_number", "event_code"}
	if _, err := validateAIWorkspaceAsset("template", smtp, legacy); err == nil {
		t.Fatal("a template missing extra_tag passed on the tagged sample")
	}
	kv := check("kv_template")
	config := kv["config"].(map[string]any)
	config["key_aliases"] = map[string]any{"reason": "failure_reason"}
	response, data := f.request(http.MethodPost, path, map[string]any{
		"kind": "kv_template", "samples": []string{smtp}, "config": config,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("KV alias preview failed: %d %s", response.StatusCode, data)
	}
	alias := decodeMap(t, data)["examples"].([]any)[0].(map[string]any)["actual"].(map[string]any)
	if alias["failure_reason"] != "not needed" || alias["reason"] != nil {
		t.Fatalf("renamed KV output: %v", alias)
	}
	examples = kv["examples"].([]any)
	first := examples[0].(map[string]any)["actual"].(map[string]any)
	second := examples[1].(map[string]any)["actual"].(map[string]any)
	if fresh := examples[3].(map[string]any)["actual"].(map[string]any); fresh["status"] != "queued" || fresh["reason"] != "retry later" || fresh["server"] != "mx-qa-02" {
		t.Fatalf("new KV output failed: %v", fresh)
	}
	if first["status"] != "failed" || first["reason"] != "not needed" ||
		second["address"] != "11.22.33.44" || second["messages"].([]any)[0] != "HTTP CHALLENGE REQUEST" {
		t.Fatalf("incorrect KV values: %v / %v", first, second)
	}
	response, data = f.request(http.MethodPost, path, map[string]any{
		"kind": "kv_template", "samples": []string{"status=failed;status=again"}, "config": kv["config"],
	})
	if response.StatusCode != http.StatusOK || decodeMap(t, data)["failed"] != float64(1) {
		t.Fatalf("duplicate source key was not rejected: %d %s", response.StatusCode, data)
	}
}
