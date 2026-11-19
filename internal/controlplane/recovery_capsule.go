package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"nexora.local/nli-v20/internal/recovery"
	metastore "nexora.local/nli-v20/internal/store"
	"time"
)

const recoveryCapsuleChunk = 1 << 20

type recoveryCapsuleManifest struct {
	Schema    int       `json:"schema_version"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
	Bytes     int       `json:"bytes"`
	Chunks    []string  `json:"chunks"`
}

func (s *Server) importRecoveryCapsule(ctx context.Context, encoded []byte, actor string) (string, error) {
	if err := recovery.ValidateEnvelope(encoded); err != nil {
		return "", err
	}

	digest := sha256.Sum256(encoded)
	hash := hex.EncodeToString(digest[:])
	id := "host-" + hash
	manifest := recoveryCapsuleManifest{Schema: 1, SHA256: hash, CreatedAt: time.Now().UTC(), Bytes: len(encoded), Chunks: []string{}}
	for offset := 0; offset < len(encoded); offset += recoveryCapsuleChunk {
		end := min(len(encoded), offset+recoveryCapsuleChunk)
		key := fmt.Sprintf("%s-%03d", hash, len(manifest.Chunks))
		payload, _ := json.Marshal(map[string]any{"ciphertext_chunk": encoded[offset:end]})
		if _, err := s.metadata.Put(ctx, metastore.Document{Collection: "recovery-ciphertext", ID: key, State: "encrypted", Payload: payload, CreatedBy: actor}); err != nil {
			return "", err
		}
		manifest.Chunks = append(manifest.Chunks, key)
	}
	payload, _ := json.Marshal(manifest)
	if _, err := s.metadata.Put(ctx, metastore.Document{Collection: "recovery-capsules", ID: id, State: "complete", Payload: payload, CreatedBy: actor}); err != nil {
		return "", err
	}
	return id, nil
}
func (s *Server) readRecoveryCapsule(ctx context.Context, id string) ([]byte, error) {
	doc, err := s.metadata.Get(ctx, "recovery-capsules", id, 0)
	if err != nil || doc.State != "complete" || doc.Deleted {
		return nil, errors.New("completed host recovery capsule not found")
	}
	var manifest recoveryCapsuleManifest
	if json.Unmarshal(doc.Payload, &manifest) != nil || manifest.Schema != 1 || id != "host-"+manifest.SHA256 || manifest.Bytes <= 0 || manifest.Bytes > recovery.MaxBytes*3 || len(manifest.Chunks) > 100 {
		return nil, errors.New("invalid host recovery manifest")
	}
	encoded := make([]byte, 0, manifest.Bytes)
	for index, key := range manifest.Chunks {
		if key != fmt.Sprintf("%s-%03d", manifest.SHA256, index) {
			return nil, errors.New("invalid host recovery chunk identity")
		}
		chunk, err := s.metadata.Get(ctx, "recovery-ciphertext", key, 0)
		if err != nil {
			return nil, err
		}
		var payload struct {
			Data []byte `json:"ciphertext_chunk"`
		}
		if chunk.Deleted || chunk.State != "encrypted" || json.Unmarshal(chunk.Payload, &payload) != nil || len(payload.Data) > recoveryCapsuleChunk {
			return nil, errors.New("invalid host recovery chunk")
		}
		encoded = append(encoded, payload.Data...)
		if len(encoded) > manifest.Bytes {
			return nil, errors.New("host recovery size mismatch")
		}
	}
	digest := sha256.Sum256(encoded)
	if len(encoded) != manifest.Bytes || hex.EncodeToString(digest[:]) != manifest.SHA256 {
		return nil, errors.New("host recovery digest mismatch")
	}
	return encoded, nil
}
func (s *Server) handleRecoveryCapsuleDownload(w http.ResponseWriter, r *http.Request, id string) {
	encoded, err := s.readRecoveryCapsule(r.Context(), id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="nli-controller-`+id+`.json.enc"`)
	_, _ = w.Write(encoded)
}
