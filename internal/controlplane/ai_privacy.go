package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"nexora.local/nli-v20/internal/store"
)

const (
	aiEvidenceProduction = "production"
	aiPrivacyWithheld    = "production_withheld"
	aiPrivacyMasked      = "masked_authoring"
	aiPrivacySampleFree  = "sample_free"
	aiPrivacyHistorySize = 64
	aiKnowledgeListLimit = 250
)

type aiPrivacyEvent struct {
	Time        time.Time `json:"time"`
	Route       string    `json:"route"`
	Disposition string    `json:"disposition"`
}

type aiPrivacyStore struct {
	mu     sync.Mutex
	counts map[string]uint64
	events [aiPrivacyHistorySize]aiPrivacyEvent
	next   int
	size   int
}

// Record only the policy decision. Prompts, identities, raw data, and provider
// credentials are deliberately outside the privacy monitor's schema.
func (s *aiPrivacyStore) record(route, disposition string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counts == nil {
		s.counts = make(map[string]uint64, 3)
	}
	s.counts[disposition]++
	s.events[s.next] = aiPrivacyEvent{Time: time.Now().UTC(), Route: route, Disposition: disposition}
	s.next = (s.next + 1) % len(s.events)
	if s.size < len(s.events) {
		s.size++
	}
}

func (s *aiPrivacyStore) snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := make([]aiPrivacyEvent, 0, s.size)
	for i := 0; i < s.size; i++ {
		index := (s.next - 1 - i + len(s.events)) % len(s.events)
		events = append(events, s.events[index])
	}
	return map[string]any{
		aiPrivacyWithheld:   s.counts[aiPrivacyWithheld],
		aiPrivacyMasked:     s.counts[aiPrivacyMasked],
		aiPrivacySampleFree: s.counts[aiPrivacySampleFree],
		"events":            events,
	}
}

func validAIEvidenceSource(source string) bool {
	return source == "" || source == "manual" || source == aiEvidenceProduction
}

func productionAssuranceEvidence(classification, source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	return classification == aiEvidenceProduction || source == "production" ||
		strings.HasPrefix(source, "quarantine:") || strings.HasPrefix(source, "fixture:") ||
		strings.HasPrefix(source, "live:") || strings.HasPrefix(source, "production:")
}

func (s *Server) recordAIAuthoring(route, raw string) {
	disposition := aiPrivacySampleFree
	if strings.TrimSpace(raw) != "" {
		disposition = aiPrivacyMasked
	}
	s.aiPrivacy.record(route, disposition)
}

type aiKnowledgeWorkspace struct {
	ID       string `json:"id"`
	Guidance string `json:"guidance"`
}

type aiKnowledgeCount struct {
	Count     int  `json:"count"`
	Truncated bool `json:"truncated"`
	Available bool `json:"available"`
}

var aiKnowledgeCollections = map[string]string{
	"parser_packs": "parser-packs", "templates": "templates", "kv_templates": "kv-config",
	"regex_patterns": "regex-patterns", "known_key_catalogs": "known-keys", "static_patterns": "static-patterns",
}

func (s *Server) aiKnowledgeInventory(ctx context.Context) map[string]aiKnowledgeCount {
	inventory := make(map[string]aiKnowledgeCount, len(aiKnowledgeCollections)+2)
	if s.parsers != nil {
		inventory["registered_parsers"] = aiKnowledgeCount{Count: len(s.parsers.All()), Available: true}
	}
	if s.metadata == nil {
		return inventory
	}
	for name, collection := range aiKnowledgeCollections {
		documents, err := s.metadata.List(ctx, collection, store.Query{Limit: aiKnowledgeListLimit})
		if err != nil {
			inventory[name] = aiKnowledgeCount{}
			continue
		}
		truncated := len(documents) == aiKnowledgeListLimit
		inventory[name] = aiKnowledgeCount{Count: len(documents), Truncated: truncated, Available: true}
		if collection == "known-keys" {
			keys := make(map[string]struct{})
			for _, document := range documents {
				var payload struct {
					Config struct {
						Keys []string `json:"keys"`
					} `json:"config"`
				}
				if json.Unmarshal(document.Payload, &payload) == nil {
					for _, key := range payload.Config.Keys {
						if key = strings.TrimSpace(key); key != "" {
							keys[key] = struct{}{}
						}
					}
				}
			}
			inventory["known_key_names"] = aiKnowledgeCount{Count: len(keys), Truncated: truncated, Available: true}
		}
	}
	return inventory
}

func (s *Server) handleAIPrivacy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	workspaces := make([]aiKnowledgeWorkspace, 0, len(aiWorkspaceWorkflows))
	for id, guidance := range aiWorkspaceWorkflows {
		workspaces = append(workspaces, aiKnowledgeWorkspace{ID: id, Guidance: guidance})
	}
	sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].ID < workspaces[j].ID })
	artifacts := make([]string, 0, len(aiWorkspaceCollections))
	for kind := range aiWorkspaceCollections {
		artifacts = append(artifacts, kind)
	}
	slices.Sort(artifacts)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	jsonOK(w, map[string]any{
		"policy": map[string]any{
			"live_ingestion_external": false, "production_evidence_external": false,
			"manual_samples": "bounded_masked", "audit_payloads": false,
			"activity_retention": "controller_memory", "activity_history_limit": aiPrivacyHistorySize,
			"manual_origin": "Pasted text has no trusted origin. Use redacted or synthetic manual examples.",
		},
		"activity": s.aiPrivacy.snapshot(),
		"knowledge": map[string]any{
			"version": 1, "workspaces": workspaces, "artifact_types": artifacts,
			"inventory": s.aiKnowledgeInventory(ctx),
		},
	})
}
