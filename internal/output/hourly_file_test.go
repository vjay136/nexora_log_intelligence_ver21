package output

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWriterChangesHourlyTemplateAtUTCBoundary(t *testing.T) {
	dir := t.TempDir()
	clock := time.Date(2026, 10, 14, 8, 59, 59, 0, time.UTC)
	w, err := NewFileWriter(Config{Type: "file", PathTemplate: filepath.Join(dir, "imap_normalised_{hour_utc}.jsonl"), Routes: []string{"normal"}})
	if err != nil {
		t.Fatal(err)
	}
	w.now = func() time.Time {
		now := clock
		clock = clock.Add(time.Second)
		return now
	}
	if err := w.Write([]byte(`{"line":1}`), "normal"); err != nil {
		t.Fatal(err)
	}
	if err := w.Write([]byte(`{"line":2}`), "normal"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"imap_normalised_20261014_0800.jsonl", "imap_normalised_20261014_0900.jsonl"} {
		if data, readErr := os.ReadFile(filepath.Join(dir, name)); readErr != nil || len(data) == 0 {
			t.Fatalf("hourly output %s missing: %q, %v", name, data, readErr)
		}
	}
}

func TestResolveFilePathRejectsAmbiguousTemplate(t *testing.T) {
	for _, cfg := range []Config{
		{Path: "/tmp/a.jsonl", PathTemplate: "/tmp/a_{hour_utc}.jsonl"},
		{PathTemplate: "/tmp/a.jsonl"},
		{PathTemplate: "relative_{hour_utc}.jsonl"},
	} {
		if _, err := resolveFilePath(cfg, time.Now()); err == nil {
			t.Fatalf("accepted invalid file config: %#v", cfg)
		}
	}
}

func TestEnsureHourlyFilesPreservesRecordsAndCreatesBothIdleRoutes(t *testing.T) {
	dir := t.TempDir()
	clock := time.Date(2026, 10, 14, 8, 59, 59, 0, time.UTC)
	configs := []Config{
		{Type: "file", PathTemplate: filepath.Join(dir, "smtp_normalised_{hour_utc}.jsonl"), Routes: []string{"normal"}},
		{Type: "file", PathTemplate: filepath.Join(dir, "smtp_quarantine_{hour_utc}.jsonl"), Routes: []string{"quarantine"}},
		{Type: "file", Path: filepath.Join(dir, "static.jsonl")},
	}
	if err := EnsureHourlyFiles(configs, clock); err != nil {
		t.Fatal(err)
	}
	normal, _ := ResolveFilePath(configs[0], clock)
	if err := os.WriteFile(normal, []byte("{\"parsed\":true}\n"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := EnsureHourlyFiles(configs, clock); err != nil {
		t.Fatal(err)
	}
	nextHour := clock.Add(time.Second)
	if err := EnsureHourlyFiles(configs, nextHour); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(normal); err != nil || string(content) != "{\"parsed\":true}\n" {
		t.Fatalf("existing output changed: %q, %v", content, err)
	}
	for _, dest := range []struct {
		config Config
		clock  time.Time
	}{{configs[1], clock}, {configs[0], nextHour}, {configs[1], nextHour}} {
		path, _ := ResolveFilePath(dest.config, dest.clock)
		if content, err := os.ReadFile(path); err != nil || len(content) != 0 {
			t.Fatalf("expected empty hourly file %q: %q, %v", path, content, err)
		}
	}
	if _, err := os.Stat(configs[2].Path); !os.IsNotExist(err) {
		t.Fatalf("static destination was materialized: %v", err)
	}
}
