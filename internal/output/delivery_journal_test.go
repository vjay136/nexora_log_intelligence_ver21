package output

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func deliveryTestID(text string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(text))) }

func TestIdentifiedLocalDeliverySurvivesRetryAndRetiresOnlyAfterCheckpoint(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "events.jsonl")
	state := filepath.Join(root, "delivery.wal")
	cfg := Config{Type: "file", Path: path, DeliveryStatePath: state}
	first, _ := NewFileWriter(cfg)
	for _, id := range []string{"source1-occurrence1", "source1-occurrence2"} {
		if err := first.WriteIdentified([]byte(`{"message":"identical"}`), "normal", deliveryTestID(id)); err != nil {
			t.Fatal(err)
		}
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	retry, _ := NewFileWriter(cfg)
	for _, id := range []string{"source1-occurrence1", "source1-occurrence2"} {
		if err := retry.WriteIdentified([]byte(`{"message":"identical"}`), "normal", deliveryTestID(id)); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(path)
	if strings.Count(string(data), "\n") != 2 {
		t.Fatal("retry duplicated records or identical legitimate occurrences collapsed")
	}
	if err := retry.AcknowledgeDeliveries([]string{deliveryTestID("source1-occurrence1"), deliveryTestID("source1-occurrence2")}); err != nil {
		t.Fatal(err)
	}
	if err := retry.Close(); err != nil {
		t.Fatal(err)
	}
	receipts, err := os.ReadFile(state)
	if err != nil || len(receipts) != 0 {
		t.Fatal("checkpointed delivery receipts were not retired")
	}
}

func TestIdentifiedDeliveryRecoversInterruptedAppend(t *testing.T) {
	for _, kind := range []string{"intent_only", "partial", "complete", "torn_journal"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "events.jsonl")
			state := filepath.Join(root, "delivery.wal")
			prefix := []byte("{\"previous\":1}\n")
			original := []byte("{\"value\":\"original\"}\n")
			id := deliveryTestID("source-generation-1-occurrence-2")
			hash := fmt.Sprintf("%x", sha256.Sum256(original))
			j, err := openDeliveryJournal(state)
			if err != nil {
				t.Fatal(err)
			}
			entry := deliveryEntry{Kind: "intent", ID: id, Path: path, Offset: int64(len(prefix)), Length: len(original), SHA256: hash}
			if err = j.append(entry); err != nil {
				t.Fatal(err)
			}
			if err = j.close(); err != nil {
				t.Fatal(err)
			}
			initial := append([]byte(nil), prefix...)
			if kind == "partial" {
				initial = append(initial, original[:8]...)
			}
			if kind == "complete" || kind == "torn_journal" {
				initial = append(initial, original...)
			}
			if err = os.WriteFile(path, initial, 0600); err != nil {
				t.Fatal(err)
			}
			if kind == "torn_journal" {
				f, _ := os.OpenFile(state, os.O_APPEND|os.O_WRONLY, 0600)
				_, _ = f.WriteString(`{"kind":"comm`)
				_ = f.Close()
			}
			w, err := NewFileWriter(Config{Type: "file", Path: path, DeliveryStatePath: state})
			if err != nil {
				t.Fatal(err)
			}
			if err = w.WriteIdentified([]byte(`{"value":"reconstructed"}`), "normal", id); err != nil {
				t.Fatal(err)
			}
			if err = w.Close(); err != nil {
				t.Fatal(err)
			}
			data, _ := os.ReadFile(path)
			if strings.Count(string(data), "\n") != 2 || !strings.HasPrefix(string(data), string(prefix)) {
				t.Fatal("interrupted append damaged previous data or duplicated the event")
			}
			if (kind == "complete" || kind == "torn_journal") && !strings.Contains(string(data), "original") {
				t.Fatal("durable original append was overwritten")
			}
		})
	}
}

func TestIdentifiedDeliveryRejectsConflictingCompletedAppend(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "events.jsonl")
	state := filepath.Join(root, "delivery.wal")
	j, _ := openDeliveryJournal(state)
	id := deliveryTestID("event")
	payload := []byte("{\"x\":1}\n")
	if err := j.append(deliveryEntry{Kind: "intent", ID: id, Path: path, Length: len(payload), SHA256: fmt.Sprintf("%x", sha256.Sum256(payload))}); err != nil {
		t.Fatal(err)
	}
	_ = j.close()
	_ = os.WriteFile(path, []byte("{\"x\":2}\n"), 0600)
	w, _ := NewFileWriter(Config{Type: "file", Path: path, DeliveryStatePath: state})
	if err := w.WriteIdentified(payload[:len(payload)-1], "normal", id); err == nil {
		t.Fatal("conflicting external bytes were silently replaced")
	}
	_ = w.Close()
}

func TestSourceAcknowledgementPrunesAfterCheckpointBeforeCleanupCrash(t *testing.T) {
	root := t.TempDir()
	cfg := Config{Type: "file", Path: filepath.Join(root, "events.jsonl"), DeliveryStatePath: filepath.Join(root, "delivery.wal")}
	source := strings.Repeat("a", 48)
	other := strings.Repeat("b", 48)
	w, _ := NewFileWriter(cfg)
	for _, id := range []string{source + "0000000000000001", source + "0000000000000002", other + "0000000000000001"} {
		if err := w.WriteIdentified([]byte(`{"value":1}`), "normal", id); err != nil {
			t.Fatal(err)
		}
	}
	_ = w.Close()
	// The caller's cursor already acknowledges only source A occurrence1.
	restart, _ := NewFileWriter(cfg)
	if err := restart.AcknowledgeSource(source, 1); err != nil {
		t.Fatal(err)
	}
	if restart.delivery.committed[source+"0000000000000001"] || !restart.delivery.committed[source+"0000000000000002"] || !restart.delivery.committed[other+"0000000000000001"] {
		t.Fatal("startup pruning crossed source/cursor boundary")
	}
	_ = restart.Close()
}

func BenchmarkFileDeliveryDurability(b *testing.B) {
	for _, identified := range []bool{false, true} {
		name := "buffered_append_locked_reference"
		if identified {
			name = "R8_identified_durable"
		}
		b.Run(name, func(b *testing.B) {
			root := b.TempDir()
			w, _ := NewFileWriter(Config{Type: "file", Path: filepath.Join(root, "events.jsonl"), DeliveryStatePath: filepath.Join(root, "delivery.wal")})
			defer w.Close()
			payload := []byte(`{"service":"synthetic","message":"durability accounting"}`)
			source := strings.Repeat("a", 48)
			b.ReportAllocs()
			b.SetBytes(int64(1000 * len(payload)))
			b.ResetTimer()
			for batch := 0; batch < b.N; batch++ {
				for i := 0; i < 1000; i++ {
					var err error
					if identified {
						err = w.WriteIdentified(payload, "normal", fmt.Sprintf("%s%016x", source, uint64(batch*1000+i+1)))
					} else {
						err = w.Write(payload, "normal")
					}
					if err != nil {
						b.Fatal(err)
					}
				}
				if err := w.Flush(); err != nil {
					b.Fatal(err)
				}
				if identified {
					if err := w.AcknowledgeSource(source, uint64((batch+1)*1000)); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func TestRouterRejectsDuplicateLocalFileOwners(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if router, err := NewRouter([]Config{{Type: "file", Path: path}, {Type: "file", Path: path}}); err == nil {
		_ = router.Close()
		t.Fatal("two independent queues admitted for one local output file")
	}
}

func TestIdentifiedWriterFollowsAtomicQuarantineReplacement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "queue.jsonl")
	w, _ := NewFileWriter(Config{Type: "file", Path: path, DeliveryStatePath: filepath.Join(root, "delivery.wal")})
	if err := w.WriteIdentified([]byte(`{"id":"resolved"}`), "quarantine", deliveryTestID("first")); err != nil {
		t.Fatal(err)
	}
	replacement := path + ".replay.tmp"
	_ = os.WriteFile(replacement, []byte("{\"id\":\"still-waiting\"}\n"), 0600)
	if err := os.Rename(replacement, path); err != nil {
		t.Skipf("platform does not replace an open queue: %v", err)
	}
	if err := w.WriteIdentified([]byte(`{"id":"new-occurrence"}`), "quarantine", deliveryTestID("second")); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	data, _ := os.ReadFile(path)
	if strings.Count(string(data), "\n") != 2 || !strings.Contains(string(data), "new-occurrence") || strings.Contains(string(data), "resolved") {
		t.Fatal("live writer appended to an unlinked pre-replay queue")
	}
}

func TestConcurrentWriterInstancesShareReceiptsWithoutLosingOtherOwners(t *testing.T) {
	root := t.TempDir()
	cfg := Config{Type: "file", Path: filepath.Join(root, "events.jsonl"), DeliveryStatePath: filepath.Join(root, "delivery.wal")}
	first, _ := NewFileWriter(cfg)
	second, _ := NewFileWriter(cfg)
	var wg sync.WaitGroup
	writers := []*FileWriter{first, second}
	for owner := range writers {
		wg.Add(1)
		go func(owner int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if err := writers[owner].WriteIdentified([]byte(`{"message":"legitimate identical occurrence"}`), "normal", deliveryTestID(fmt.Sprintf("owner%d-occurrence%d", owner, i))); err != nil {
					t.Error(err)
					return
				}
			}
		}(owner)
	}
	wg.Wait()
	ids := []string{}
	for i := 0; i < 50; i++ {
		ids = append(ids, deliveryTestID(fmt.Sprintf("owner0-occurrence%d", i)))
	}
	if err := first.AcknowledgeDeliveries(ids); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		if err := second.WriteIdentified([]byte(`{"message":"legitimate identical occurrence"}`), "normal", deliveryTestID(fmt.Sprintf("owner1-occurrence%d", i))); err != nil {
			t.Fatal(err)
		}
	}
	_ = first.Close()
	_ = second.Close()
	data, _ := os.ReadFile(cfg.Path)
	if strings.Count(string(data), "\n") != 100 {
		t.Fatal("shared receipt reload/compaction duplicated events or erased another writer's receipts")
	}
}
