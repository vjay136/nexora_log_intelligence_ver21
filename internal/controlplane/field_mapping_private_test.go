package controlplane

import (
	"context"
	"os"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/fieldmapping"
)

// Private operator evidence is used only on explicit opt-in. Raw records and
// field values are never printed, committed or included in release artifacts.
func TestNFIPrivateOperatorCorpus(t *testing.T) {
	path := os.Getenv("NLI_NFI_PRIVATE_CORPUS")
	if path == "" {
		t.Skip("operator corpus is private and opt-in")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	samples := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			samples = append(samples, line)
		}
	}
	profiles, diagnostics := nfiSuggestCorpus(context.Background(), samples, "")
	if len(profiles) == 0 {
		t.Fatal("private corpus produced no structural profiles")
	}
	// Qualify the actual merged profile the operator will save, not only the
	// narrower per-record suggestions. Optional nested keys may occur in some
	// members of a structural family and be absent in the others.
	merged := map[string]*fieldmapping.Compiled{}
	for _, profile := range profiles {
		fields := append([]fieldmapping.Field(nil), profile.Fields...)
		for i := range fields {
			if fields[i].Method == "delimiter" {
				fields[i].Confirmed = true
			}
		}
		compiled, err := fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: fields})
		if err != nil {
			t.Fatalf("private merged profile %s is invalid: %v", profile.ID, err)
		}
		if _, exists := merged[profile.Family]; exists {
			t.Fatal("private corpus emitted duplicate structural family profiles")
		}
		merged[profile.Family] = compiled
	}
	checked := 0
	for index, raw := range samples {
		family, fields, err := nfiSuggestRecord(context.Background(), raw, "")
		if err != nil {
			t.Fatalf("private sample %d could not be inspected: %v", index+1, err)
		}
		for i := range fields {
			if fields[i].Method == "delimiter" {
				fields[i].Confirmed = true
			}
		}
		compiled, err := fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: fields})
		if err != nil {
			t.Fatalf("private sample %d plan invalid: %v", index+1, err)
		}
		result, err := compiled.Execute(context.Background(), raw)
		if err != nil {
			t.Fatalf("private sample %d extraction failed: %v", index+1, err)
		}
		if len(result.Values) == 0 {
			t.Fatalf("private sample %d produced no observed values", index+1)
		}
		profile := merged[family]
		if profile == nil {
			t.Fatalf("private sample %d has no matching merged structural profile", index+1)
		}
		mergedResult, err := profile.Execute(context.Background(), raw)
		if err != nil {
			t.Fatalf("private sample %d merged profile extraction failed: %v", index+1, err)
		}
		if len(mergedResult.Values) == 0 {
			t.Fatalf("private sample %d merged profile produced no observed values", index+1)
		}
		checked++
	}
	t.Logf("Private corpus acceptance: %d records execute through individual and merged plans, %d structural profiles, %d diagnostic messages; no raw evidence persisted", checked, len(profiles), len(diagnostics))
}
