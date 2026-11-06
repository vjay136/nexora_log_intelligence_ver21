package adapter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/fieldmapping"
	parserpolicy "nexora.local/nli-v20/parser-policy"
)

func sourceParityAdapter(t *testing.T, raw string, plan fieldmapping.Plan) (*ParserPackAdapter, map[string]interface{}) {
	t.Helper()
	compiled, err := fieldmapping.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := compiled.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, f := range plan.Fields {
		names = append(names, f.Name)
	}
	pack := &parserpolicy.ParserPack{ID: "nfi-source-parity", Version: "1.0.0", Revision: 1, SourceType: "manual", Author: "administrator", ReviewState: parserpolicy.ReviewStateApproved, CreatedAt: time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC), ResourceLimits: parserpolicy.DefaultResourceLimits(), Metadata: map[string]string{"mapping_engine": "nfi-v1", "selection_policy": "reviewed_mapping"}, Rules: []parserpolicy.ParserRule{{ID: "mapping", Name: "Reviewed source mapping", Priority: 1, ServiceSelectors: []string{"nfi-fixture"}, Stages: []parserpolicy.Stage{{ID: "map", Name: "Mapping", Type: parserpolicy.StageTypeMapping, Config: plan, FailurePolicy: parserpolicy.FailurePolicyRejectRule}}, FailurePolicy: parserpolicy.FailurePolicyQuarantine, ProjectionMode: parserpolicy.ProjectionModeAllowlist, AllowedFields: names, CountPolicy: parserpolicy.CountPolicyProjection}}, RegressionCases: []parserpolicy.RegressionCase{{Name: "source", Input: raw, Service: "nfi-fixture", SourceType: "file", ExpectedPayload: preview.Values, ExpectedParser: "nfi-source-parity/mapping", ExpectedRoute: "normal"}}}
	pack.ResourceLimits.MaxAttempts = 1
	if err := parserpolicy.SealParserPack(pack); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, pack.ID+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	adapter, err := NewParserPackAdapterFromDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	return adapter, preview.Values
}

func TestNFIOriginalSourcePreviewAdapterAndAgentParity(t *testing.T) {
	raw := "  abc \nsecond  "
	plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "leading", Method: "character", Source: "1", SourceEnd: "2"}, {Name: "bytes", Method: "byte", Source: "3", SourceEnd: "6"}, {Name: "last_line", Method: "line", Source: "2"}, {Name: "trailing", Method: "character", Source: "14", SourceEnd: "15"}}}
	adapter, want := sourceParityAdapter(t, raw, plan)
	source := engine.Source{Service: "nfi-fixture"}
	direct := adapter.TryParseV5Production(raw, source)
	if !direct.Success || !reflect.DeepEqual(direct.EventPayload.Fields, want) {
		t.Fatalf("adapter drift: %+v want %+v", direct, want)
	}
	deps := engine.Default()
	deps.SetV5Parser(adapter)
	actual := engine.Parse(raw, source, deps)
	if actual.NLI.Parser != "nfi-source-parity/mapping" || !reflect.DeepEqual(actual.Data, want) {
		t.Fatalf("agent drift: %+v want %+v", actual, want)
	}
}

func TestNFIReviewedJSONMappingPrecedesGenericJSONOnlyInScope(t *testing.T) {
	raw := ` {"request":{"code":"0042"},"unused":true} `
	plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "operator_code", Method: "json_path", Source: "request.code"}}}
	adapter, want := sourceParityAdapter(t, raw, plan)
	deps := engine.Default()
	deps.SetV5Parser(adapter)
	actual := engine.Parse(raw, engine.Source{Service: "nfi-fixture"}, deps)
	if actual.NLI.Parser != "nfi-source-parity/mapping" || !reflect.DeepEqual(actual.Data, want) {
		t.Fatalf("JSON mapping bypassed: %+v", actual)
	}
	truncated := engine.ParseWithTruncated(raw, engine.Source{Service: "nfi-fixture"}, deps, true)
	if truncated.Meta.Assurance.Verified || string(truncated.Meta.Route) != "quarantine" || truncated.NLI.Confidence > 0.79 {
		t.Fatalf("truncated mapping falsely verified: %+v", truncated)
	}
	outside := engine.Parse(raw, engine.Source{Service: "unrelated-service"}, deps)
	if outside.NLI.Parser == "nfi-source-parity/mapping" || outside.Data["unused"] != true {
		t.Fatalf("unrelated JSON fast path changed: %+v", outside)
	}
}
