package controlplane

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAIWorkspaceMappingWithNoPriorPreview(t *testing.T) {
	f := newAPIFixture(t)
	const raw = "severity=warning message=observed"
	const plan = `{"plan":{"version":1,"fields":[{"name":"severity","method":"key_value","source":"severity"},{"name":"message","method":"key_value","source":"message"}]}}`
	calls := mappingAIProviderFixture(t, f, plan, func(req map[string]any) {
		data, _ := json.Marshal(req)
		if !bytes.Contains(data, []byte("Rename the output")) {
			t.Error("operator preferences were not passed to the provider")
		}
	})
	response, body := f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{
		"mode": "fix", "workspace": "quarantine-wizard", "tab": "Mapping",
		"artifact": "parser", "question": "Rename the output", "raw": raw,
		"desired": map[string]any{"severity": "warning", "message": "observed"},
	})
	if response.StatusCode != http.StatusOK || calls.Load() != 1 {
		t.Fatalf("mapping suggestion did not run: %d %s", response.StatusCode, body)
	}
	result := decodeMap(t, body)
	validation := result["validation"].(map[string]any)
	if validation["desired_match"] != true || result["activated"] != false || result["persisted"] != false {
		t.Fatalf("unexpected validation or lifecycle result: %s", body)
	}
}

func TestAIWorkspacePreservesPipeLabelsWhenMasking(t *testing.T) {
	f := newAPIFixture(t)
	calls := mappingAIProviderFixture(t, f, "Select the source format.", func(req map[string]any) {
		messages := req["messages"].([]any)
		prompt := messages[1].(map[string]any)["content"].(string)
		for _, text := range []string{"Input paths", "Choose profile", "|from=", "|to=", "|subj="} {
			if !strings.Contains(prompt, text) {
				t.Errorf("context lost %q", text)
			}
		}
		if strings.Contains(prompt, "private@example.test") {
			t.Error("prompt retained a private email")
		}
	})
	response, body := f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{
		"mode": "ask", "workspace": "input-paths", "tab": "Input paths", "controls": []string{"Choose profile"},
		"question": "Which control should I use?", "raw": "|from=private@example.test|to=buyer@example.test|subj=observed",
	})
	if response.StatusCode != http.StatusOK || calls.Load() != 1 || !bytes.Contains(body, []byte("Select the source format.")) {
		t.Fatalf("workspace guidance failed: %d %s", response.StatusCode, body)
	}
}

func TestAIWorkspaceRejectsRegexThatOnlyMatchesPartOfSample(t *testing.T) {
	f := newAPIFixture(t)
	reply := `{"answer":"Use a named capture.","config":{"pattern":"(?P<value>first)","fields":{"value":"value"}}}`
	mappingAIProviderFixture(t, f, reply, nil)
	response, body := f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{
		"mode": "fix", "workspace": "regex", "artifact": "regex", "raw": "first unmatched",
	})
	if response.StatusCode != http.StatusUnprocessableEntity || !bytes.Contains(body, []byte("entire supplied sample")) {
		t.Fatalf("partial regex was accepted: %d %s", response.StatusCode, body)
	}
}

func TestAIWorkspaceKVTemplatePreservesCompleteValues(t *testing.T) {
	f := newAPIFixture(t)
	reply := `{"answer":"Use pipe boundaries.","config":{"pair_separators":["|"],"assignment_separators":["="],"duplicate_policy":"array","max_fields":256}}`
	mappingAIProviderFixture(t, f, reply, nil)
	response, body := f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{
		"mode": "fix", "workspace": "kv-templates", "artifact": "kv_template", "raw": `|sid=a\|b|hdr=first|hdr=second`,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("KV template %d: %s", response.StatusCode, body)
	}
	draft := decodeMap(t, body)
	values := draft["validation"].(map[string]any)["actual"].(map[string]any)
	hdr, _ := values["hdr"].([]any)
	if values["sid"] != "a|b" || len(hdr) != 2 || hdr[0] != "first" || hdr[1] != "second" || draft["persisted"] != false {
		t.Fatalf("unexpected KV values or lifecycle result: %s", body)
	}
	response, body = f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{
		"mode": "fix", "workspace": "kv-templates", "artifact": "kv_template", "raw": `|sid=a|unnamed-segment`,
	})
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("partial KV template %d: %s", response.StatusCode, body)
	}
}

func TestAIWorkspaceDirectAuthoringRequestsBuildDrafts(t *testing.T) {
	f := newAPIFixture(t)
	calls := mappingAIProviderFixture(t, f, `{"renames":{}}`, nil)
	for _, question := range []string{"Please do field mappings", "Create a parser pack"} {
		response, body := f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{
			"mode": "ask", "workspace": "reference-ai", "question": question,
			"raw": "status=failed reason=queued",
		})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("direct command returned %d: %s", response.StatusCode, body)
		}
		result := decodeMap(t, body)
		if result["proposal"] == nil || result["proposal_origin"] != "ai" || result["activated"] != false {
			t.Fatal("command did not return a verified review-only proposal")
		}
	}
	response, _ := f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{
		"mode": "ask", "workspace": "reference-ai", "question": "Map the fields",
	})
	if response.StatusCode != http.StatusUnprocessableEntity || calls.Load() != 2 {
		t.Fatal("a draft without raw evidence must not call the provider")
	}
	for _, question := range []string{"How do I map fields?", "Deploy a parser pack", "Which regex matches this?"} {
		if aiDirectDraftArtifact(question) != "" {
			t.Fatalf("guidance misclassified: %s", question)
		}
	}
}
