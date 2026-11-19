package auth

import (
	"encoding/hex"
	"testing"
)

func TestPasswordDerivationCompatibility(t *testing.T) {
	for _, vector := range []struct {
		iterations int
		expected   string
	}{
		{1, "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b"},
		{2, "ae4d0c95af6b46d32d0adff928f06dd02a303f8ef3c251dfd6e2d85a95474c43"},
	} {
		got := hex.EncodeToString(pbkdf2SHA256([]byte("password"), []byte("salt"), vector.iterations, 32))
		if got != vector.expected {
			t.Fatalf("password derivation mismatch for %d iterations: %s", vector.iterations, got)
		}
	}
}
