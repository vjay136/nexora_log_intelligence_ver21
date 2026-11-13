package fieldmapping

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestAdvancedKVSharedHeaderTemplate(t *testing.T) {
	ctx := context.Background()
	fields := make([]Field, 0, 10)
	for i, name := range []string{"date", "timestamp", "hostname", "service", "pid", "minor_pid", "thread_id"} {
		fields = append(fields, Field{Name: name, Type: "string", Method: "position", Source: fmt.Sprint(i + 1)})
	}
	fields = append(fields,
		Field{Name: "event_type", Type: "string", Method: "delimiter", Source: "8", Delimiter: ";", Part: "1", Confirmed: true},
		Field{Name: "event", Type: "string", Method: "delimiter", Source: "8", Delimiter: ";", Part: "2", Confirmed: true},
	)
	tail := Field{Name: "kv_pair", Type: "object", Policy: "optional", Method: "kv_all", Source: "9",
		KVProfile: "configured", EndOfLine: true, KeyOptions: KeyOptions{RequireComplete: true},
		KVLiteralBoundaries: true, KVBareField: "messages", KVEmptyKeyField: "unnamed"}
	tail.KVPairSeparators = []string{":", ";", "|", "||"}
	tail.AssignmentSeparators = []string{"=", "=>"}
	plan, err := Compile(Plan{Version: 1, Fields: append(fields, tail)})
	if err != nil {
		t.Fatal(err)
	}
	headerOnly := strings.TrimSpace(kvieMailHeader)
	large := make([]string, 120)
	wantLarge := make(map[string]any, len(large))
	for i := range large {
		key := fmt.Sprintf("key_%03d", i)
		large[i] = key + "=value"
		wantLarge[key] = "value"
	}
	for _, tt := range []struct {
		name, raw, service, event string
		want                      map[string]any
	}{
		{"header only", headerOnly, "smtp", "SieveRuleMsg", nil},
		{"semicolon", kvieMailHeader + "status=failed;reason=not needed", "smtp", "SieveRuleMsg", map[string]any{"status": "failed", "reason": "not needed"}},
		{"colon and bare message", "20260401 134601279+0100 server001 pabd 250410 0 139764638705408 Note;CARDDAV id=002fac3e:HTTP CHALLENGE REQUEST:method=1:realm=mx:address=11.22.33.44", "pabd", "CARDDAV", map[string]any{"id": "002fac3e", "messages": []any{"HTTP CHALLENGE REQUEST"}, "method": "1", "realm": "mx", "address": "11.22.33.44"}},
		{"empty key and value", kvieMailHeader + "=unknown||state=>ready;note=|detail=\"x:y\"", "smtp", "SieveRuleMsg", map[string]any{"unnamed": []any{"unknown"}, "state": "ready", "note": "", "detail": "x:y"}},
		{"many dynamic names", kvieMailHeader + strings.Join(large, ";"), "smtp", "SieveRuleMsg", wantLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := plan.Execute(ctx, tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			if err := plan.ValidateComplete(tt.raw, result); err != nil {
				t.Fatal(err)
			}
			if result.Values["service"] != tt.service || result.Values["event"] != tt.event || result.Values["event_type"] != "Note" {
				t.Fatalf("unexpected header %v", result.Values)
			}
			if tt.want == nil {
				if _, found := result.Values["kv_pair"]; found {
					t.Fatal("header-only input produced a tail")
				}
			} else if !reflect.DeepEqual(result.Values["kv_pair"], tt.want) {
				t.Fatalf("got %v, want %v", result.Values["kv_pair"], tt.want)
			}
		})
	}

	preview, err := InspectKeyValues(ctx, kvieMailHeader+"id=foo:HTTP CHALLENGE REQUEST:method=1", tail)
	if err != nil || len(preview) != 1 || len(preview[0].Observations) != 3 {
		t.Fatalf("configured preview = %v, %v", preview, err)
	}
	for _, sample := range []string{
		"status=ok;invalid key=hidden", "status=ok;status=again", "status=ok;note=\"unterminated",
		"status=ok;;reason=missing", "status=ok;", "status=ok;messages=collision",
	} {
		if _, err := plan.Execute(ctx, kvieMailHeader+sample); !errors.Is(err, ErrIncompleteKV) {
			t.Errorf("accepted malformed tail %q or lost its diagnostic: %v", sample, err)
		}
	}
}

func TestAdvancedKVUnnamedSegmentsRequireExplicitOutput(t *testing.T) {
	tail := Field{Name: "kv_pair", Type: "object", Method: "kv_all", Source: "9",
		KVProfile: "configured", KeyOptions: KeyOptions{RequireComplete: true}, KVLiteralBoundaries: true}
	tail.PairDelimiters = ":;"
	plan, err := Compile(Plan{Version: 1, Fields: []Field{tail}})
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []string{"status=ok:unassigned:reason=failed", "status=ok;=hidden"} {
		if _, err := plan.Execute(context.Background(), kvieMailHeader+sample); !errors.Is(err, ErrIncompleteKV) {
			t.Errorf("accepted unnamed segment %q: %v", sample, err)
		}
	}
}

func TestAdvancedKVFlattensLargeRecordsWithinBudget(t *testing.T) {
	field := Field{Name: "tail", Type: "object", Method: "kv_all", Source: "9",
		KVProfile: "configured", KVLiteralBoundaries: true, KVMaxPairs: 512,
		OutputPolicy: "flatten", PairDelimiters: ";", KeyOptions: KeyOptions{RequireComplete: true}}
	plan, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "service", Method: "position", Source: "4"}, field}})
	if err != nil {
		t.Fatal(err)
	}
	pairs := make([]string, 513)
	for i := range pairs {
		pairs[i] = fmt.Sprintf("key_%03d=value", i)
	}
	raw := kvieMailHeader + strings.Join(pairs[:512], ";")
	result, err := plan.Execute(context.Background(), raw)
	if err != nil || len(result.Values) != 513 || result.Values["key_511"] != "value" {
		t.Fatalf("large flatten has %d fields, err=%v", len(result.Values), err)
	}
	if err := plan.ValidateComplete(raw, result); err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Execute(context.Background(), kvieMailHeader+strings.Join(pairs, ";")); !errors.Is(err, ErrIncompleteKV) {
		t.Fatalf("expected limit rejection, got %v", err)
	}
}
