package controlplane

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"nexora.local/nli-v20/internal/fieldmapping"
	metastore "nexora.local/nli-v20/internal/store"
)

func mappingAIProviderFixture(t *testing.T, f *apiFixture, reply string, inspect func(map[string]any), status ...int) *atomic.Int32 {
	t.Helper()
	t.Setenv("NLI_AI_ALLOW_LOOPBACK", "true")
	calls := new(atomic.Int32)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := int(calls.Add(1))
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if inspect != nil {
			inspect(request)
		}
		w.Header().Set("Content-Type", "application/json")
		if len(status) > 0 {
			w.WriteHeader(status[min(attempt-1, len(status)-1)])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": reply}, "finish_reason": "stop"}}})
	}))
	t.Cleanup(provider.Close)
	if err := f.server.storeAIProvider(context.Background(), aiProviderConfig{ID: "mapping-assistant", Provider: "openai_compatible", Model: "synthetic-mapping-model", BaseURL: provider.URL, Enabled: true, APIKey: "synthetic-provider-key-1234"}, "admin"); err != nil {
		t.Fatal(err)
	}
	return calls
}

func TestFieldMappingAICompoundProposalUsesNativeSchemaAndStaysEphemeral(t *testing.T) {
	f := newAPIFixture(t)
	definition, desired := nfiExampleDefinition()
	// Private unrequested values are intentionally omitted, rather than being
	// silently dropped by an otherwise successful AI proposal.
	fixturePlan := definition["mapping_plan"].(map[string]any)
	fixturePlan["fields"] = append(fixturePlan["fields"].([]any), map[string]any{"name": "ignored_owner", "method": "key_value", "source": "owner", "output_policy": "omit"}, map[string]any{"name": "ignored_password", "method": "key_value", "source": "password", "output_policy": "omit"})
	encoded, _ := json.Marshal(map[string]any{"plan": definition["mapping_plan"]})
	manual := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "password", Method: "constant", Source: "manual-private-source", DefaultValue: "manual-private-default", Preview: "manual-private-preview", Evidence: "manual-private-evidence", Example: "manual-private-example", Description: "manual-private-description", Operations: []fieldmapping.Operation{{Op: "replace", Find: "manual-private-find", Replace: "manual-private-replace", Value: "manual-private-value"}}}}}
	manualBefore, _ := json.Marshal(manual)
	calls := mappingAIProviderFixture(t, f, string(encoded), func(request map[string]any) {
		body, _ := json.Marshal(request)
		for _, private := range []string{"manual-private-", "private@example.test", "raw-private-password", "synthetic-provider-key-1234"} {
			if bytes.Contains(body, []byte(private)) {
				t.Error("AI prompt contained private manual/source/credential values")
			}
		}
		messages, _ := request["messages"].([]any)
		if len(messages) < 2 {
			t.Error("missing schema system and masked evidence messages")
			return
		}
		system, _ := messages[0].(map[string]any)
		text, _ := system["content"].(string)
		for _, required := range []string{"conditional_merge", "assignment_separators", "quote_aware", "before", "key_value", "additionalProperties", "source_end"} {
			if !strings.Contains(text, required) {
				t.Errorf("actual native schema omitted %s", required)
			}
		}
		if request["max_tokens"] != float64(8192) {
			t.Errorf("mapping generation budget = %v, want 8192", request["max_tokens"])
		}
	})
	before, _, err := f.server.loadAIProviderByID(context.Background(), "mapping-assistant")
	if err != nil {
		t.Fatal(err)
	}
	providerDoc, _ := f.server.metadata.Get(context.Background(), "ai-provider", "mapping-assistant", 0)
	raw := nfiExample + " owner=private@example.test password=raw-private-password"
	response, body := f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": raw, "service": "smtp", "desired": desired, "plan": manual})
	if response.StatusCode != 200 {
		t.Fatalf("AI mapping failed: %d %s", response.StatusCode, body)
	}
	result := decodeMap(t, body)
	validation := result["validation"].(map[string]any)
	if validation["syntax_valid"] != true || validation["extraction_valid"] != true || validation["desired_match"] != true || validation["ready_for_publish"] != false || validation["publication_validated"] != false {
		t.Fatalf("false validation claims: %s", body)
	}
	if result["review_only"] != true || result["persisted"] != false || result["activated"] != false || result["operator_confirmation_required"] != true || calls.Load() != 1 {
		t.Fatal("AI suggestion changed state, claimed approval, or duplicated the provider request")
	}
	digest := sha256.Sum256([]byte(raw))
	if result["raw_sha256"] != hex.EncodeToString(digest[:]) {
		t.Fatal("source hash did not preserve exact evidence")
	}
	proposal := result["proposal"].(map[string]any)
	planJSON, _ := json.Marshal(proposal["plan"])
	var proposed fieldmapping.Plan
	if err = json.Unmarshal(planJSON, &proposed); err != nil {
		t.Fatal(err)
	}
	for _, field := range proposed.Fields {
		if field.Confirmed {
			t.Fatal("AI self-confirmed operator review")
		}
	}
	if _, err = fieldmapping.Compile(proposed); err == nil {
		t.Fatal("unconfirmed delimiter proposal bypassed normal native confirmation")
	}
	// Explicitly model the operator confirming the two delimiter selections,
	// then verify this canonical skeleton through the existing candidate path.
	for i := range proposed.Fields {
		if proposed.Fields[i].Method == "delimiter" {
			proposed.Fields[i].Confirmed = true
		}
	}
	parser := proposal["parser"].(map[string]any)
	parser["mapping_plan"] = proposed
	candidate := validateParserCandidateContext(context.Background(), parser, []assuranceRequest{{Raw: raw, Service: "smtp", Desired: desired, PreserveSource: true}})
	if candidate["failed"] != 0 {
		t.Fatalf("canonical proposal is incompatible with existing native candidate validation: %#v", candidate)
	}
	for _, collection := range []string{"parser-packs", "ai-remediation-proposals", "assurance-cases"} {
		docs, err := f.server.metadata.List(context.Background(), collection, metastore.Query{Limit: 100})
		if err != nil || len(docs) != 0 {
			t.Fatalf("AI suggestion persisted %s metadata", collection)
		}
	}
	after, _, _ := f.server.loadAIProviderByID(context.Background(), "mapping-assistant")
	afterDoc, _ := f.server.metadata.Get(context.Background(), "ai-provider", "mapping-assistant", 0)
	if !sameAIProfile(before, after) || afterDoc.Revision != providerDoc.Revision {
		t.Fatal("generation mutated provider credentials/readiness metadata")
	}
	manualAfter, _ := json.Marshal(manual)
	if !bytes.Equal(manualBefore, manualAfter) {
		t.Fatal("manual context was mutated")
	}
}

func TestFieldMappingAIAlreadyParsedSampleAndDesiredMismatchRemainReviewOnly(t *testing.T) {
	for _, desired := range []map[string]any{nil, {"message": "different"}} {
		f := newAPIFixture(t)
		plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "message", Method: "json_path", Source: "$.message"}}}
		reply, _ := json.Marshal(plan)
		mappingAIProviderFixture(t, f, string(reply), nil)
		response, body := f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": `{"service":"synthetic","message":"already parsed"}`, "service": "auto", "desired": desired})
		if response.StatusCode != 200 {
			t.Fatalf("successful native sample was incorrectly excluded: %d %s", response.StatusCode, body)
		}
		validation := decodeMap(t, body)["validation"].(map[string]any)
		if validation["extraction_valid"] != true || validation["ready_for_publish"] != false || validation["publication_validated"] != false {
			t.Fatal("single-sample extraction bypassed publication workflow")
		}
		if desired == nil {
			if validation["desired_match"] != nil || validation["desired_provided"] != false {
				t.Fatal("missing desired JSON was asserted to match")
			}
		} else if validation["desired_match"] != false || len(validation["diff"].([]any)) == 0 {
			t.Fatal("mismatch was hidden instead of exposed alongside the proposal")
		}
	}
}

func TestFieldMappingAIRejectsInvalidLanguagesPropertiesAndExtraction(t *testing.T) {
	for _, reply := range []string{`{broken`, `{"plan":{"version":1,"fields":[{"name":"message","method":"raw","script":"return process.env"}]}}`, `{"version":1,"fields":[{"name":"message","method":"javascript","source":"exec('bad')"}]}`, `{"version":1,"fields":[{"name":"message","method":"regex","source":"("}]}`, `{"version":1,"fields":[{"name":"message","method":"position","source":"999"}]}`, `{"version":1,"fields":[{"name":"message","method":"raw"}]} {"version":1}`, `{"plan":{"version":1,"fields":[{"name":"message","method":"raw"}]},"activate":true}`} {
		f := newAPIFixture(t)
		mappingAIProviderFixture(t, f, reply, nil)
		response, body := f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": "one two", "service": "custom"})
		if response.StatusCode != 422 {
			t.Fatalf("invalid AI plan accepted: status%d", response.StatusCode)
		}
		result := decodeMap(t, body)
		if result["proposal"] != nil || result["activated"] != false || result["persisted"] != false {
			t.Fatal("invalid AI output became an accepted/active proposal")
		}
	}
}

func TestFieldMappingAIUnavailableBoundsAndAdmission(t *testing.T) {
	f := newAPIFixture(t)
	response, _ := f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": "source"})
	if response.StatusCode != 503 {
		t.Fatal("unconfigured AI did not remain unavailable")
	}
	reply := `{"version":1,"fields":[{"name":"message","method":"raw"}]}`
	calls := mappingAIProviderFixture(t, f, reply, nil)
	response, _ = f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": strings.Repeat("a", (64<<10)+1)})
	if response.StatusCode != 400 || calls.Load() != 0 {
		t.Fatal("oversized source reached the provider")
	}
	aiReviewSlots <- struct{}{}
	aiReviewSlots <- struct{}{}
	response, _ = f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": "source"})
	<-aiReviewSlots
	<-aiReviewSlots
	if response.StatusCode != 429 || calls.Load() != 0 {
		t.Fatal("AI slot admission failed")
	}
	cfg, _, _ := f.server.loadAIProviderByID(context.Background(), "mapping-assistant")
	cfg.Enabled = false
	if err := f.server.storeAIProvider(context.Background(), cfg, "admin"); err != nil {
		t.Fatal(err)
	}
	response, _ = f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": "source"})
	if response.StatusCode != 503 || calls.Load() != 0 {
		t.Fatal("disabled provider was invoked")
	}
}

func TestFieldMappingAIPromptSchemaExamplesAndManualPrivacy(t *testing.T) {
	for _, example := range mappingAIPublicExamples() {
		plan := example["plan"].(fieldmapping.Plan)
		compiled, err := fieldmapping.Compile(plan)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = compiled.Execute(context.Background(), example["raw"].(string)); err != nil {
			t.Fatal(err)
		}
	}
	schema := mappingAIJSONSchema(reflect.TypeOf(fieldmapping.Field{}))
	properties := schema["properties"].(map[string]any)
	for _, name := range []string{"operations", "source_end", "assignment_separators", "boundary_start", "custom_pattern", "null_policy", "timezone_name"} {
		if properties[name] == nil {
			t.Fatalf("native schema omitted %s", name)
		}
	}
	if properties["tokenPattern"] != nil {
		t.Fatal("unexported implementation leaked into AI schema")
	}
	if len(mappingAISystemPrompt()) > 32<<10 {
		t.Fatal("system context exceeded its bounded budget")
	}
}

func TestFieldMappingAISeedFallbackOnInvalidProviderReply(t *testing.T) {
	const raw = "20250812 130035121+0100 aaa-gmailrgo009 smtp 3609 0 140293780948736 Note;SmtpServerEHLO(2001/4) " +
		`|GW-EU-07|DEFER|sid=5f9e\|2c1a|reason=4.7.1 greylisted; retry after 300s|hex=doted, key=normal, fan=texe`
	for _, tc := range []struct{ name, reply, origin, field string }{
		{"invalid plan", `{"plan":{"unknown":true}}`, "native_fallback", "field1"},
		{"invalid extraction", `{"plan":{"version":1,"fields":[{"name":"missing","method":"position","source":"99"}]}}`, "native_fallback", "field1"},
		{"seed rename", `{"renames":{"field1":"gateway"}}`, "ai", "gateway"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAPIFixture(t)
			calls := mappingAIProviderFixture(t, f, tc.reply, func(request map[string]any) {
				encoded, _ := json.Marshal(request)
				if !bytes.Contains(encoded, []byte("Native FieldSense seed: verified")) {
					t.Error("provider did not receive native mapping structure")
				}
			})
			response, body := f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": raw, "service": "smtp", "native_fallback": true})
			if response.StatusCode != http.StatusOK {
				t.Fatalf("suggestion status %d: %s", response.StatusCode, body)
			}
			result := decodeMap(t, body)
			validation := result["validation"].(map[string]any)
			actual := validation["actual"].(map[string]any)
			if calls.Load() != 1 || result["proposal_origin"] != tc.origin || validation["extraction_valid"] != true ||
				result["activated"] != false || result["persisted"] != false || actual[tc.field] != "GW-EU-07" ||
				actual["sid"] != "5f9e|2c1a" || actual["reason"] != "4.7.1 greylisted; retry after 300s" {
				t.Fatalf("incorrect proposal: %s", body)
			}
		})
	}
}

func TestFieldMappingAIProviderNumericSelectorsUseExactNativePreview(t *testing.T) {
	f := newAPIFixture(t)
	const raw = "node Note;Delivery(2001/4)"
	const reply = `{"plan":{"version":1,"fields":[{"name":"node","method":"position","source":1,"source_end":1},{"name":"level","method":"delimiter","source":2,"delimiter":";","part":1},{"name":"event_code","method":"regex","source":"[(]([^)]*)[)]","part":1}]}}`
	calls := mappingAIProviderFixture(t, f, reply, nil)
	response, body := f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{
		"raw": raw, "desired": map[string]any{"node": "node", "level": "Note", "event_code": "2001/4"},
	})
	if response.StatusCode != http.StatusOK || calls.Load() != 1 {
		t.Fatalf("provider numeric selectors failed: %d %s", response.StatusCode, body)
	}
	result := decodeMap(t, body)
	validation := result["validation"].(map[string]any)
	if result["proposal_origin"] != "ai" || validation["desired_match"] != true || validation["publication_validated"] != false {
		t.Fatalf("unexpected native validation: %s", body)
	}
	if _, err := decodeAIMappingPlan([]byte(reply)); err == nil {
		t.Fatal("stored mapping plans must still use the native selector types")
	}
}

func TestFieldMappingAINumericCoercionRejectsInvalidTypesAndUnknownFields(t *testing.T) {
	for _, field := range []string{
		`{"name":"event","method":"regex","source":123,"part":1}`,
		`{"name":"event","method":"regex","source":"(x)","part":1.5}`,
		`{"name":"event","method":"regex","source":"(x)","part":true}`,
		`{"name":"event","method":"position","source":1,"unknown":true}`,
	} {
		input := []byte(`{"version":1,"fields":[` + field + `]}`)
		if _, err := decodeAISeedOrMapping(input, fieldmapping.Plan{}); err == nil {
			t.Errorf("accepted malformed provider field %s", field)
		}
	}
}

func TestFieldMappingAIProviderFailureKeepsVerifiedNativePreview(t *testing.T) {
	f := newAPIFixture(t)
	const raw = "20250811 080053105+0100 mail.example.test smtp 3692 0 140026854221568 Note;SieveRuleMsg |status:failed|reason:notneeded|server:cloud"
	calls := mappingAIProviderFixture(t, f, "", nil, http.StatusServiceUnavailable)
	response, body := f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{
		"raw": raw, "service": "smtp", "native_fallback": true,
	})
	if response.StatusCode != http.StatusOK || calls.Load() != 2 {
		t.Fatalf("provider failure did not return native fallback: %d %s", response.StatusCode, body)
	}
	result := decodeMap(t, body)
	validation := result["validation"].(map[string]any)
	actual := validation["actual"].(map[string]any)
	if result["proposal_origin"] != "native_fallback" || validation["extraction_valid"] != true ||
		actual["status"] != "failed" || actual["reason"] != "notneeded" || actual["server"] != "cloud" {
		t.Fatalf("unverified or mislabeled fallback: %s", body)
	}
}

func TestFieldMappingAIRecoversFromTransientProviderFailure(t *testing.T) {
	f := newAPIFixture(t)
	calls := mappingAIProviderFixture(t, f, `{"renames":{}}`, nil, http.StatusServiceUnavailable, http.StatusOK)
	response, body := f.request(http.MethodPost, "/api/v1/field-mapping/ai-suggest", map[string]any{
		"raw": "status=failed reason=notneeded", "native_fallback": true,
	})
	if response.StatusCode != http.StatusOK || calls.Load() != 2 {
		t.Fatalf("provider retry failed: %d %s", response.StatusCode, body)
	}
	result := decodeMap(t, body)
	if result["proposal_origin"] != "ai" || result["provider_attempts"] != float64(2) {
		t.Fatalf("retry outcome was not reported accurately: %s", body)
	}
}
