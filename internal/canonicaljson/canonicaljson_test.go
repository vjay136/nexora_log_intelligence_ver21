package canonicaljson

import (
	"strings"
	"testing"
)

func TestNormalizeIsStableAcrossObjectOrderAndEquivalentNumbers(t *testing.T) {
	left, err := Normalize([]byte(`{"b":1.0,"a":2e0,"nested":{"z":-0,"x":1.20}}`))
	if err != nil {
		t.Fatalf("normalize left: %v", err)
	}
	right, err := Normalize([]byte(`{"nested":{"x":1.2,"z":0},"a":2,"b":1}`))
	if err != nil {
		t.Fatalf("normalize right: %v", err)
	}
	if string(left) != string(right) {
		t.Fatalf("semantic JSON did not canonicalize equally:\nleft:  %s\nright: %s", left, right)
	}
	if got, want := string(left), `{"a":2,"b":1,"nested":{"x":1.2,"z":0}}`; got != want {
		t.Fatalf("canonical form = %s, want %s", got, want)
	}
}

func TestNormalizePreservesArrayOrderAndStringSemantics(t *testing.T) {
	got, err := Normalize([]byte(`{"items":[3,2,1],"text":"line\n\u003c"}`))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if want := `{"items":[3,2,1],"text":"line\n\u003c"}`; string(got) != want {
		t.Fatalf("canonical form = %s, want %s", got, want)
	}
}

func TestNormalizeRejectsTrailingValuesAndUnsafeNumberExpansion(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "multiple values", raw: `{}` + "\n" + `{}`, want: "multiple JSON values"},
		{name: "exponent bound", raw: `1e100001`, want: "outside the supported canonical range"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Normalize([]byte(test.raw))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Normalize(%q) error = %v, want substring %q", test.raw, err, test.want)
			}
		})
	}
}
