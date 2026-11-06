package parserpolicy

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/fieldmapping"
)

func TestNFIMappingStageRoundTripAndProductionExecution(t *testing.T) {
	plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{
		{Name: "loglevel", Method: "delimiter", Source: "2", Delimiter: ";", Part: "1", Confirmed: true},
		{Name: "event", Method: "delimiter", Source: "2", Delimiter: ";", Part: "2", Confirmed: true},
		{Name: "status", Method: "key_value", Source: "status"},
		{Name: "pid", Method: "position", Source: "1", Type: "integer"},
		{Name: "summary", Method: "derived", Source: "loglevel,event", JoinSeparator: " / "},
		{Name: "extra", Method: "constant", DefaultValue: `{"enabled":true}`, Type: "object"},
		{Name: "parts", Method: "constant", DefaultValue: `["a",2]`, Type: "array"},
		{Name: "missing", Method: "key_value", Source: "missing", Policy: "nullable"},
	}}
	pack := productionTestPack([]Stage{{ID: "nfi", Name: "Native Field Intelligence", Type: StageTypeMapping, Config: plan, FailurePolicy: FailurePolicyRejectRule}})
	pack.Rules[0].CountPolicy = CountPolicyProjection
	pack.Rules[0].FieldSchema = map[string]FieldDef{"loglevel": {Type: FieldTypeString, Required: true}, "event": {Type: FieldTypeString, Required: true}, "status": {Type: FieldTypeString, Required: true}, "pid": {Type: FieldTypeInt, Required: true}, "summary": {Type: FieldTypeString, Required: true}, "extra": {Type: FieldTypeObject, Required: true}, "parts": {Type: FieldTypeArray, Required: true}, "missing": {Type: FieldTypeString, Nullable: true}}
	if err := SealParserPack(pack); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadParserPackBytes(encoded)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := NewParserPackExecutor(loaded)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"123 Note;SieveRuleMsg status=failed:server=cloud", "123 Note;SieveRuleMsg server=cloud:status=failed"} {
		result, err := executor.Execute(&ExecutionRequest{Input: raw, Service: "smtp", Context: context.Background()})
		if err != nil || result.Quarantined || result.EventPayload == nil {
			t.Fatalf("runtime failed: %+v %v", result, err)
		}
		fields := result.EventPayload.Fields
		if fields["loglevel"] != "Note" || fields["event"] != "SieveRuleMsg" || fields["status"] != "failed" || fields["pid"] != 123 || fields["summary"] != "Note / SieveRuleMsg" {
			t.Fatal(fields)
		}
		if value, ok := fields["missing"]; !ok || value != nil {
			t.Fatal("nullable lost")
		}
	}
	result, err := executor.Execute(&ExecutionRequest{Input: "invalid Note;SieveRuleMsg status=failed", Service: "smtp", Context: context.Background()})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Quarantined || result.EventPayload != nil {
		t.Fatal("invalid required type emitted normalized output")
	}
}

func TestNFIMappingStageRejectsUnknownConfigAndFalseCountPolicy(t *testing.T) {
	pack := productionTestPack([]Stage{{ID: "nfi", Name: "NFI", Type: StageTypeMapping, Config: map[string]any{"version": 1, "fields": []map[string]any{{"name": "x", "method": "constant", "default_value": "safe"}}, "execute": "forbidden"}, FailurePolicy: FailurePolicyRejectRule}})
	pack.Rules[0].CountPolicy = CountPolicyProjection
	if err := SealParserPack(pack); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(pack)
	if _, err := LoadParserPackBytes(encoded); err == nil {
		t.Fatal("arbitrary code property accepted")
	}
	pack.Rules[0].Stages[0].Config = fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "x", Method: "constant", DefaultValue: "safe"}}}
	pack.Rules[0].CountPolicy = CountPolicyExactSource
	if validation := ValidateParserPack(pack); validation.Valid || !strings.Contains(strings.Join(validation.Errors, " "), "count policy") {
		t.Fatalf("mapping derived values accepted under exact_source: %+v", validation)
	}
}
