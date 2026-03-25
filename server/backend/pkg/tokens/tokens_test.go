package tokens

import "testing"

func TestGenerateSecureTokenLength(t *testing.T) {
	token, err := GenerateSecureToken(16)
	if err != nil {
		t.Fatalf("GenerateSecureToken returned error: %v", err)
	}

	if len(token) != 32 {
		t.Fatalf("expected 32 hex chars, got %d", len(token))
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	left := HashToken("termviewer-token")
	right := HashToken("termviewer-token")
	other := HashToken("different-token")

	if left != right {
		t.Fatalf("expected deterministic hash output")
	}

	if left == other {
		t.Fatalf("expected different inputs to produce different hashes")
	}
}
