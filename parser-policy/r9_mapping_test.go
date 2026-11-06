package parserpolicy

import (
	"nexora.local/nli-v20/internal/fieldmapping"
	"testing"
)

func TestR9MappingDynamicKeysSurviveFixedSchema(t *testing.T) {
	plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "date", Method: "position", Source: "1"}, {Name: "tail_fields", Method: "kv_all", Type: "object", Source: "9", OutputPolicy: "flatten", KeyOptions: fieldmapping.KeyOptions{RequireComplete: true}}}}
	pack := productionTestPack([]Stage{{ID: "mapping", Name: "Mapping", Type: StageTypeMapping, Config: plan, FailurePolicy: FailurePolicyRejectRule}})
	pack.Rules[0].CountPolicy = CountPolicyAllowDerived
	pack.Rules[0].FieldSchema = map[string]FieldDef{"date": {Type: FieldTypeString, Required: true}}
	raw := "20250811 080053105+0100 host smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:future=0042"
	result := executeProductionTestPack(t, pack, raw)
	if result.EventPayload == nil || result.Quarantined || result.EventPayload.Fields["future"] != "0042" {
		t.Fatalf("dynamic schema dropped: %+v", result.ParseTelemetry)
	}
	plan.Fields = plan.Fields[:1]
	pack.Rules[0].Stages[0].Config = plan
	result = executeProductionTestPack(t, pack, raw)
	if result.EventPayload != nil || !result.Quarantined {
		t.Fatal("header-only mapping normalized structured suffix")
	}
}
