package controlplane

import (
	"nexora.local/nli-v20/internal/fieldmapping"
	"strings"
	"testing"
)

func TestNFIScopeIsolationAndExplicitGlobalGuard(t *testing.T) {
	definition, desired := nfiExampleDefinition()
	cases := []assuranceRequest{{Raw: nfiExample, Service: "auto", Desired: desired}}
	pack, err := assuranceMappingRuntimePack("nfi-scope", "1.0.0", "admin", definition, cases)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(pack.Rules[0].ServiceSelectors, ",") != "smtp-in" || pack.RegressionCases[0].Service != "smtp-in" {
		t.Fatalf("inferred scope: %+v", pack.Rules[0])
	}
	definition["service_scope"] = "imap"
	if validateParserCandidate(definition, cases)["status"] != "invalid_definition" {
		t.Fatal("cross-service case accepted")
	}
	definition["service_scope"] = "*"
	pack, err = assuranceMappingRuntimePack("nfi-fallback", "1.0.0", "admin", definition, cases)
	if err != nil || pack.Metadata["selection_policy"] != "" {
		t.Fatalf("unguarded wildcard became authoritative: %v", err)
	}
	definition["reviewed_global_scope"] = true
	if _, err = assuranceMappingRuntimePack("nfi-global", "1.0.0", "admin", definition, cases); err == nil {
		t.Fatal("unguarded global override accepted")
	}
	definition["scope_pattern"] = `^20250811 .* smtp .*Note;SieveRuleMsg `
	pack, err = assuranceMappingRuntimePack("nfi-global", "1.0.0", "admin", definition, cases)
	if err != nil || pack.Metadata["selection_policy"] != "reviewed_mapping" || len(pack.Rules[0].IncludePredicates) != 1 {
		t.Fatalf("guarded override: %v", err)
	}
	cases[0].Raw = "different record"
	if validateParserCandidate(definition, cases)["status"] != "invalid_definition" {
		t.Fatal("out-of-pattern case accepted")
	}
}

func TestNFIConstantPlanGuardAndUnknownFutureService(t *testing.T) {
	plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "label", Method: "constant", Type: "string", DefaultValue: "operator label"}}}
	cases := []assuranceRequest{{Raw: "future unclassified format", Service: "auto"}}
	selectors, pattern, authoritative, err := nfiMappingScope(map[string]any{}, plan, cases)
	if err != nil || len(selectors) != 1 || selectors[0] != "__unclassified__" || pattern != "^(?:future unclassified format)$" || !authoritative {
		t.Fatalf("constant scope: %v %q %v %v", selectors, pattern, authoritative, err)
	}
}

func TestNFIScopeAutoInferenceMatchesRuntimeFamilies(t *testing.T) {
	cases := []struct{ raw, service, want string }{
		{"20250811 080053105+0100 synthetic-rgi009 smtp 12 0 42 Note;Event status=failed", "auto", "smtp-in"},
		{"20250811 080053105+0100 synthetic-rgo009 smtp 12 0 42 Note;Event status=sent", "auto", "smtp-out"},
		{"2025-08-11 08:00:53:105+0100 action=login session=example", "auto", "webtop"},
		{"2025-08-11T08:00:53Z action=login", "auto", "__unclassified__"},
		{"opaque", "imapserv", "imap"},
		{"opaque", "popserv", "pop"},
	}
	for _, item := range cases {
		if got := nfiCaseService(assuranceRequest{Raw: item.raw, Service: item.service}); got != item.want {
			t.Errorf("service for %q: got %q want %q", item.raw, got, item.want)
		}
	}
}

func TestNFIScopeExplicitDirectionalAndBroadSMTP(t *testing.T) {
	plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "first", Method: "position", Source: "1"}}}
	cases := []assuranceRequest{{Raw: nfiExample, Service: "auto"}}
	for _, scope := range []string{"smtp-in", "smtp"} {
		selectors, _, authoritative, err := nfiMappingScope(map[string]any{"service_scope": scope}, plan, cases)
		if err != nil || !authoritative {
			t.Fatalf("matching %s scope rejected: %v %v", scope, selectors, err)
		}
		if scope == "smtp" && strings.Join(selectors, ",") != "smtp,smtp-in,smtp-out" {
			t.Fatalf("broad SMTP not expanded: %v", selectors)
		}
	}
}

func TestNFIScopeRejectsMalformedOptionalTypes(t *testing.T) {
	plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "first", Method: "position", Source: "1"}}}
	for _, definition := range []map[string]any{{"scope_pattern": 123}, {"scope_pattern": []any{".*"}}, {"reviewed_global_scope": "false"}, {"reviewed_global_scope": 1}} {
		if _, _, _, err := nfiMappingScope(definition, plan, []assuranceRequest{{Raw: "observed", Service: "fixture"}}); err == nil {
			t.Fatalf("malformed scope silently accepted: %+v", definition)
		}
	}
}

func TestNFIScopePatternBoundAndCaseMembership(t *testing.T) {
	plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "first", Method: "position", Source: "1"}}}
	for _, pattern := range []string{strings.Repeat("x", 1025), "[", "^other$"} {
		if _, _, _, err := nfiMappingScope(map[string]any{"scope_pattern": pattern}, plan, []assuranceRequest{{Raw: "observed", Service: "fixture"}}); err == nil {
			t.Fatalf("invalid or mismatched pattern accepted: %q", pattern)
		}
	}
}
