package controlplane

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	fleetGrantPrefix = "fleet."
	fleetNodePrefix  = "nli-fleet-"
)

func enrollmentDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

// acceptFleetGrant runs under enrollmentMu. The root-installed grant contains
// only an expiry and its digest is the filename. The claim binds the grant to
// one stable node identity before registration, so interrupted requests retry
// safely without allowing the same grant to create a second node.
func (s *Server) acceptFleetGrant(token, nodeID, proof string) (bool, error) {
	if !strings.HasPrefix(nodeID, fleetNodePrefix) || len(nodeID) != len(fleetNodePrefix)+64 {
		return false, nil
	}
	if _, err := hex.DecodeString(nodeID[len(fleetNodePrefix):]); err != nil || nodeID != strings.ToLower(nodeID) {
		return false, nil
	}
	if len(token) != len(fleetGrantPrefix)+64 {
		return false, nil
	}
	if _, err := hex.DecodeString(proof); len(proof) != 64 || err != nil {
		return false, nil
	}
	digest := enrollmentDigest(token)
	root := filepath.Join(s.cfg.StateDir, "enrollment")
	expires, err := os.ReadFile(filepath.Join(root, "grants", digest))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	grant := strings.Fields(string(expires))
	if len(grant) == 0 || len(grant) > 2 || (len(grant) == 2 && grant[1] != nodeID) {
		return false, nil
	}
	deadline, err := time.Parse(time.RFC3339, grant[0])
	if err != nil || !time.Now().Before(deadline) {
		return false, nil
	}

	claimsPath := filepath.Join(s.cfg.StateDir, "fleet-enrollment-claims.json")
	type enrollmentClaim struct {
		NodeID    string `json:"node_id"`
		ProofHash string `json:"proof_hash"`
		Replaced  bool   `json:"replaced,omitempty"`
	}
	claims := make(map[string]enrollmentClaim)
	if data, err := os.ReadFile(claimsPath); err == nil {
		if err := json.Unmarshal(data, &claims); err != nil {
			return false, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	proofHash := enrollmentDigest(proof)
	if claim, claimed := claims[digest]; claimed {
		return !claim.Replaced && claim.NodeID == nodeID && subtle.ConstantTimeCompare([]byte(claim.ProofHash), []byte(proofHash)) == 1, nil
	}
	replacement := false
	for previousDigest, claim := range claims {
		if !claim.Replaced && claim.NodeID == nodeID {
			if len(grant) != 2 || subtle.ConstantTimeCompare([]byte(claim.ProofHash), []byte(proofHash)) != 1 {
				return false, nil
			}
			claim.Replaced = true
			claims[previousDigest] = claim
			replacement = true
			break
		}
	}
	if !replacement {
		if len(grant) == 2 {
			return false, nil
		}
		if _, exists := s.fleet.Get(nodeID); exists {
			return false, nil
		}
	}
	claims[digest] = enrollmentClaim{NodeID: nodeID, ProofHash: proofHash}
	data, err := json.Marshal(claims)
	if err != nil {
		return false, err
	}
	return true, writeEnrollmentClaim(claimsPath, string(data))
}

func writeEnrollmentClaim(path, value string) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".enrollment-claim-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString(value); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
