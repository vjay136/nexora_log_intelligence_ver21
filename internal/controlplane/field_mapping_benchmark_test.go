package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	parserpolicy "nexora.local/nli-v20/parser-policy"
)

// Compare the same record and exact output through the preserved R6 regex
// runtime path and NFI. This is a focused workload, not a capacity claim.
func BenchmarkNFIAndR6RegexSameOutput(b *testing.B) {
	native, desired := nfiExampleDefinition()
	regex := map[string]any{"pattern_type": "regex", "pattern": `^(?P<date>\S+)\s+(?P<time_timezone>\S+)\s+(?P<hostname>\S+)\s+(?P<service>\S+)\s+(?P<pid>\S+)\s+(?P<minor_pid>\S+)\s+(?P<thread_id>\S+)\s+(?P<loglevel>[^;\s]+);(?P<event>\S+)\s+status=(?P<status>[^:]+):reason=(?P<reason>[^:]+):server=(?P<server>\S+)$`}
	fields := []any{}
	for _, name := range []string{"date", "time_timezone", "hostname", "service", "pid", "minor_pid", "thread_id", "loglevel", "event", "status", "reason", "server"} {
		fields = append(fields, map[string]any{"name": name, "source": "group:" + name, "type": "string", "required": true})
	}
	regex["fields"] = fields
	cases := []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}}
	for _, item := range []struct {
		name       string
		definition map[string]any
	}{{"R6_regex", regex}, {"NFI_mapping", native}} {
		pack, err := assuranceRuntimePack("bench-mapping", "1.0.0", "benchmark", item.definition, cases)
		if err != nil {
			b.Fatal(err)
		}
		executor, err := parserpolicy.NewParserPackExecutor(pack)
		if err != nil {
			b.Fatal(err)
		}
		request := &parserpolicy.ExecutionRequest{Input: nfiExample, Service: nfiCaseService(cases[0]), SourceType: "file", Context: context.Background()}
		result, err := executor.Execute(request)
		if err != nil || result.Quarantined {
			detail,_:=json.Marshal(result);b.Fatalf("benchmark output did not pass reviewed regression: err=%v result=%s",err,detail)
		}
		if diff, _ := structuralDiff(result.EventPayload.Fields, desired); len(diff) != 0 {
			b.Fatal("benchmark outputs differ")
		}
		b.Run(item.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(nfiExample)))
			for i := 0; i < b.N; i++ {
				result, err := executor.Execute(request)
				if err != nil || result.Quarantined {
					b.Fatal("execution failed")
				}
			}
		})
	}
}
