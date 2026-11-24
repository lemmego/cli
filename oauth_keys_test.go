package cli

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestOAuthKeysAreGeneratedForAnOAuth2Project is the guard for a generated
// project that does not boot.
//
// The oauth2 provider fails when the signing key is absent, and a failing
// provider panics out of registerProviders — so selecting the module and
// running `lemmego dev` ended in a stack trace. The module's own error names
// the command that fixes it, and that command does work; a scaffolded project
// should still run rather than hand somebody a recovery step.
func TestOAuthKeysAreGeneratedForAnOAuth2Project(t *testing.T) {
	dir := t.TempDir()
	generateOAuthKeys(dir)

	private := filepath.Join(dir, "storage", "oauth", "private.key")
	info, err := os.Stat(private)
	if err != nil {
		t.Fatalf("no signing key was written: %v", err)
	}

	// Mode 0600 is not cosmetic. oauth2.FileKeySource refuses a key with any
	// group or other permission bit set, so a relaxed mode here produces a
	// project that fails to boot complaining about permissions instead of
	// about a missing file — a worse error for the same outcome.
	//
	// Windows does not carry Unix permission bits, so the assertion is
	// skipped there rather than made to pass by weakening it.
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm&0o077 != 0 {
			t.Errorf("private.key is mode %#o; oauth2 refuses any key readable "+
				"by group or other", perm)
		}
	}

	// PKCS#8 PEM, which is what the reader parses.
	body, err := os.ReadFile(private)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(body)
	if block == nil {
		t.Fatal("private.key is not PEM")
	}
	if block.Type != "PRIVATE KEY" {
		t.Errorf("PEM type = %q, want %q — PKCS#1 would say RSA PRIVATE KEY", block.Type, "PRIVATE KEY")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("private.key does not parse as PKCS#8: %v", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("private.key is %T, want *rsa.PrivateKey", key)
	}
	if bits := rsaKey.N.BitLen(); bits < 2048 {
		t.Errorf("key is %d bits; the module's own default is 2048", bits)
	}

	// The public half is served in the JWKS, so its absence is a broken
	// discovery document rather than a broken boot.
	publicPEM, err := os.ReadFile(filepath.Join(dir, "storage", "oauth", "public.key"))
	if err != nil {
		t.Fatalf("no public key was written: %v", err)
	}
	pubBlock, _ := pem.Decode(publicPEM)
	if pubBlock == nil || pubBlock.Type != "PUBLIC KEY" {
		t.Error("public.key is not a PKIX PEM public key")
	}
}

// TestOAuthKeysAreNotReplaced keeps scaffolding from destroying a key.
//
// Replacing the signer invalidates every access token in flight and every
// refresh token ever issued, so an existing key is left alone rather than
// resolved — which is why the write uses O_EXCL.
func TestOAuthKeysAreNotReplaced(t *testing.T) {
	dir := t.TempDir()
	keyDir := filepath.Join(dir, "storage", "oauth")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	existing := []byte("-----BEGIN PRIVATE KEY-----\nnot a real key\n-----END PRIVATE KEY-----\n")
	if err := os.WriteFile(filepath.Join(keyDir, "private.key"), existing, 0o600); err != nil {
		t.Fatal(err)
	}

	generateOAuthKeys(dir)

	after, err := os.ReadFile(filepath.Join(keyDir, "private.key"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(existing) {
		t.Error("an existing signing key was replaced; every token ever issued " +
			"would have been invalidated")
	}
}
