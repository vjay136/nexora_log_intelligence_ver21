package engine

import (
	"nexora.local/nli-v20/internal/fieldmapping"
	"nexora.local/nli-v20/internal/registry"
	"testing"
)

type legacySourceRecorder struct{ input string }

func (p *legacySourceRecorder) TryParseV5(input string, source Source) *V5ParseResult {
	p.input = input
	return &V5ParseResult{}
}
func TestNFIRawOptInPreservesLegacyV5Trimming(t *testing.T) {
	recorder := &legacySourceRecorder{}
	deps := Default()
	deps.SetV5Parser(recorder)
	Parse("  opaque source  ", Source{Service: "fixture"}, deps)
	if recorder.input != "opaque source" {
		t.Fatalf("legacy input changed: %q", recorder.input)
	}
}
func TestNFISelectedRegistryMappingPreservesOriginalSource(t *testing.T) {
	deps := Default()
	deps.Registry = registry.New()
	deps.Registry.Register(&registry.ParserDef{ID: "raw-map", Version: 1, PatternType: "mapping", MappingPlan: &fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "prefix", Method: "character", Source: "1", SourceEnd: "2"}}}})
	result, err := ParseWithSelectedParser("  abc ", Source{}, deps, "raw-map")
	if err != nil || result.Data["prefix"] != "  " {
		t.Fatalf("selected mapping drift: %+v %v", result, err)
	}
}
