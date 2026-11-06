package controlplane

import (
	"context"
	"nexora.local/nli-v20/internal/fieldmapping"
	"testing"
)

func TestR9SuggestionsDiscoverEntireMailKVTail(t *testing.T) {
	prefix := "20250811 080053105+0100 host smtp 3692 0 140026854221568 Note;SieveRuleMsg "
	for _, tail := range []string{"status=failed;reason=not needed", "status=failed:reason=notneeded:server=cloud", "|status:failed|reason:notneeded|server:cloud"} {
		raw := prefix + tail
		_, fields, err := nfiSuggestRecord(context.Background(), raw, "")
		if err != nil {
			t.Fatal(err)
		}
		for i := range fields {
			fields[i].Confirmed = true
		}
		p, err := fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: fields})
		if err != nil {
			t.Fatal(err)
		}
		r, err := p.Execute(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		if r.Values["status"] != "failed" || r.Values["reason"] == nil {
			t.Fatalf("suffix fields missing: %+v", r.Values)
		}
		if err := p.ValidateComplete(raw, r); err != nil {
			t.Fatalf("suggested incomplete: %v", err)
		}
	}
	_, fields, err := nfiSuggestRecord(context.Background(), prefix+"status:failed reason;notneeded server;cloud", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fields {
		if f.Method == "key_value" {
			t.Fatal("mixed ambiguous assignments guessed")
		}
	}
}
