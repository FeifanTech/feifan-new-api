package seat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestEncryptDecryptTokenRoundTrip(t *testing.T) {
	common.CryptoSecret = "test-crypto-secret-1234567890"
	plain := "ghp_xxx_test_token"

	enc, err := encryptToken(plain)
	if err != nil {
		t.Fatalf("encryptToken failed: %v", err)
	}
	if enc == plain {
		t.Fatalf("encryptToken should not return plain text")
	}

	dec, err := decryptToken(enc)
	if err != nil {
		t.Fatalf("decryptToken failed: %v", err)
	}
	if dec != plain {
		t.Fatalf("decryptToken got %q, want %q", dec, plain)
	}
}

func TestNormalizeTenantID(t *testing.T) {
	if got := normalizeTenantID(""); got != "default" {
		t.Fatalf("normalizeTenantID empty got %q", got)
	}
	if got := normalizeTenantID("  "); got != "default" {
		t.Fatalf("normalizeTenantID spaces got %q", got)
	}
	if got := normalizeTenantID("tenant-a"); got != "tenant-a" {
		t.Fatalf("normalizeTenantID got %q", got)
	}
}
