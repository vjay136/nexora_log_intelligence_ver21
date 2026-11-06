package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"nexora.local/nli-v20/internal/output"
)

func countAccountingRows(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var value map[string]any
		if err = json.Unmarshal([]byte(line), &value); err != nil {
			t.Fatal("output contains incomplete JSONL")
		}
		count++
	}
	return count
}

func Test49PhysicalRecordsRequiredRemoteRetryPreservesLocalCounts(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%v", legacy), func(t *testing.T) {
			root := t.TempDir()
			input := filepath.Join(root, "input.log")
			normal := filepath.Join(root, "normal.jsonl")
			quarantine := filepath.Join(root, "quarantine.jsonl")
			var raw strings.Builder
			normalCount, quarantineCount := 0, 0
			for i := 0; i < 49; i++ {
				if i%3 == 0 {
					fmt.Fprintf(&raw, "rejected-input-%d\x00\n", i)
					quarantineCount++
				} else {
					raw.WriteString("{\"service\":\"auto\",\"message\":\"intentionally identical\"}\n")
					normalCount++
				}
			}
			if err := os.WriteFile(input, []byte(raw.String()), 0600); err != nil {
				t.Fatal(err)
			}
			var fail atomic.Bool
			fail.Store(true)
			var accepted atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if fail.Load() {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				var push struct {
					Streams []struct {
						Values [][2]string `json:"values"`
					} `json:"streams"`
				}
				if err := json.NewDecoder(r.Body).Decode(&push); err != nil {
					t.Error(err)
				}
				for _, stream := range push.Streams {
					accepted.Add(int64(len(stream.Values)))
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			cfg := Config{Version: 2, Inputs: []InputConfig{{Path: input, Service: "auto"}}, StateDir: filepath.Join(root, "state"), PollMS: 10, DisableV5Parser: true, EnableLegacyRouting: legacy, Outputs: []output.Config{{Type: "file", Path: normal, Routes: []string{"normal", "warning"}}, {Type: "file", Path: quarantine, Routes: []string{"quarantine", "unknown"}}, {Type: "loki", URL: server.URL, Required: true, RetryMax: 1, RetryBackoffMS: 1, BatchSize: 1000, Routes: []string{"normal", "warning"}}}}
			if err := Run(context.Background(), cfg, true); err == nil {
				t.Fatal("required remote outage was acknowledged")
			}
			if got := countAccountingRows(t, normal); got != normalCount {
				t.Fatalf("normal rows after failure=%d want%d", got, normalCount)
			}
			if got := countAccountingRows(t, quarantine); got != quarantineCount {
				t.Fatalf("raw quarantine rows after failure=%d want%d", got, quarantineCount)
			}
			fail.Store(false)
			for i := 0; i < 2; i++ {
				if err := Run(context.Background(), cfg, true); err != nil {
					t.Fatal(err)
				}
			}
			n, q := countAccountingRows(t, normal), countAccountingRows(t, quarantine)
			if n != normalCount || q != quarantineCount || n+q != 49 {
				t.Fatalf("retry accounting normal=%d quarantine=%d total=%d want49", n, q, n+q)
			}
			if accepted.Load() != int64(normalCount) {
				t.Fatalf("successful remote rows=%d want%d", accepted.Load(), normalCount)
			}
			data, err := os.ReadFile(filepath.Join(cfg.StateDir, "offsets.json"))
			if err != nil {
				t.Fatal(err)
			}
			var states map[string]cursor
			if err = json.Unmarshal(data, &states); err != nil {
				t.Fatal(err)
			}
			if states[input].Events != 49 || states[input].Line != 49 || states[input].Offset != int64(raw.Len()) {
				t.Fatal("durable input accounting does not match all49 source records")
			}
			journals, _ := filepath.Glob(filepath.Join(cfg.StateDir, "delivery", "*.wal"))
			for _, path := range journals {
				info, err := os.Stat(path)
				if err != nil || info.Size() != 0 {
					t.Fatal("acknowledged receipts accumulated after successful checkpoint")
				}
			}
		})
	}
}

func TestDuplicatePhysicalInputAliasesAreRejected(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "input.log")
	_ = os.WriteFile(path, []byte("same input\n"), 0600)
	if err := validateDistinctInputs([]InputConfig{{Path: path}, {Path: filepath.Join(root, ".", "input.log")}}); err == nil {
		t.Fatal("duplicate path accepted")
	}
	alias := filepath.Join(root, "hardlink.log")
	if err := os.Link(path, alias); err == nil {
		if err = validateDistinctInputs([]InputConfig{{Path: path}, {Path: alias}}); err == nil {
			t.Fatal("physical hardlink aliases accepted")
		}
	}
}

func TestSourceGenerationSeparatesSameBytesAfterRotationAndTruncation(t *testing.T) {
	for _, rotation := range []bool{false, true} {
		t.Run(fmt.Sprintf("rotation=%v", rotation), func(t *testing.T) {
			root := t.TempDir()
			input := filepath.Join(root, "input.log")
			quarantine := filepath.Join(root, "quarantine.jsonl")
			line := "same-rejected-input\x00\n"
			_ = os.WriteFile(input, []byte(strings.Repeat(line, 49)), 0600)
			cfg := Config{Inputs: []InputConfig{{Path: input, Service: "auto"}}, StateDir: filepath.Join(root, "state"), PollMS: 10, DisableV5Parser: true, Outputs: []output.Config{{Type: "file", Path: quarantine, Routes: []string{"quarantine", "unknown"}}}}
			if err := Run(context.Background(), cfg, true); err != nil {
				t.Fatal(err)
			}
			if rotation {
				if err := os.Rename(input, input+".old"); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(input, []byte(line), 0600); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if err := Run(context.Background(), cfg, true); err != nil {
					t.Fatal(err)
				}
			}
			if countAccountingRows(t, quarantine) != 50 {
				t.Fatal("rotation/copytruncate collapsed legitimate same-byte occurrences or replayed old records")
			}
			data, _ := os.ReadFile(quarantine)
			seen := map[string]bool{}
			for _, row := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var value map[string]any
				_ = json.Unmarshal([]byte(row), &value)
				id, _ := value["id"].(string)
				if seen[id] {
					t.Fatal("quarantine occurrence ID reused across positions/generations")
				}
				seen[id] = true
			}
		})
	}
}

func TestAgentStateLockPreventsConcurrentCursorOwners(t *testing.T) {
	root := t.TempDir()
	unlock, err := lockAgentState(root)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := lockAgentState(root); err == nil {
		_ = second()
		t.Fatal("concurrent agent state owner admitted")
	}
	if err = unlock(); err != nil {
		t.Fatal(err)
	}
	again, err := lockAgentState(root)
	if err != nil {
		t.Fatal("state lock did not release on close")
	}
	_ = again()
}

func Test49CombinedLocalDestinationHasOneRowPerOutcome(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.log")
	target := filepath.Join(root, "all.jsonl")
	var raw strings.Builder
	for i := 0; i < 49; i++ {
		if i%3 == 0 {
			raw.WriteString("rejected\x00\n")
		} else {
			raw.WriteString("{\"message\":\"identical\"}\n")
		}
	}
	_ = os.WriteFile(input, []byte(raw.String()), 0600)
	cfg := Config{Inputs: []InputConfig{{Path: input, Service: "auto"}}, StateDir: filepath.Join(root, "state"), PollMS: 10, DisableV5Parser: true, Outputs: []output.Config{{Type: "file", Path: target}}}
	for i := 0; i < 2; i++ {
		if err := Run(context.Background(), cfg, true); err != nil {
			t.Fatal(err)
		}
	}
	if countAccountingRows(t, target) != 49 {
		t.Fatal("combined normal/quarantine destination overcounted or dropped inputs")
	}
}
