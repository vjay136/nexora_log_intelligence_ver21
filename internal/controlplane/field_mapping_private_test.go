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
	checked := 0
	for index, raw := range samples {
		_, fields, err := nfiSuggestRecord(context.Background(), raw, "")
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
		checked++
	}
	t.Logf("Private corpus acceptance: %d records, %d structural profiles, %d diagnostic messages; no raw evidence persisted", checked, len(profiles), len(diagnostics))
}
