package grpcproxy

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureCA(t *testing.T) {
	dir := t.TempDir()

	// First call should generate
	cert, created, err := EnsureCA(dir)
	if err != nil {
		t.Fatalf("EnsureCA: %v", err)
	}
	if !created {
		t.Error("expected created=true on first call")
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("no certificates in TLS cert")
	}

	// Files should exist
	if _, err := os.Stat(filepath.Join(dir, "ca.crt")); err != nil {
		t.Errorf("ca.crt not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ca.key")); err != nil {
		t.Errorf("ca.key not found: %v", err)
	}

	// Key file should have restricted permissions
	info, err := os.Stat(filepath.Join(dir, "ca.key"))
	if err == nil && info.Mode().Perm()&0077 != 0 {
		t.Errorf("ca.key permissions too open: %o", info.Mode().Perm())
	}

	// Second call should load, not regenerate
	cert2, created2, err := EnsureCA(dir)
	if err != nil {
		t.Fatalf("EnsureCA (reload): %v", err)
	}
	if created2 {
		t.Error("expected created=false on reload")
	}
	if len(cert2.Certificate) == 0 {
		t.Fatal("no certificates on reload")
	}

	// Verify CA properties
	caCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}
	if !caCert.IsCA {
		t.Error("certificate is not a CA")
	}
	if caCert.Subject.CommonName != "SC Bridge Local CA" {
		t.Errorf("CN = %q, want %q", caCert.Subject.CommonName, "SC Bridge Local CA")
	}
}

func TestLeafCertCache(t *testing.T) {
	dir := t.TempDir()
	ca, _, err := EnsureCA(dir)
	if err != nil {
		t.Fatalf("EnsureCA: %v", err)
	}

	cache, err := NewLeafCertCache(ca)
	if err != nil {
		t.Fatalf("NewLeafCertCache: %v", err)
	}

	// Generate a leaf cert
	hello := &tls.ClientHelloInfo{
		ServerName: "test.cloudimperiumgames.com",
	}
	leaf, err := cache.GetCertificate(hello)
	if err != nil {
		t.Fatalf("GetCertificate: %v", err)
	}
	if len(leaf.Certificate) != 2 {
		t.Fatalf("expected 2 certs in chain (leaf + CA), got %d", len(leaf.Certificate))
	}

	// Verify the leaf chains to the CA
	leafCert, err := x509.ParseCertificate(leaf.Certificate[0])
	if err != nil {
		t.Fatalf("parse leaf cert: %v", err)
	}
	if leafCert.Subject.CommonName != "test.cloudimperiumgames.com" {
		t.Errorf("leaf CN = %q", leafCert.Subject.CommonName)
	}
	if len(leafCert.DNSNames) == 0 || leafCert.DNSNames[0] != "test.cloudimperiumgames.com" {
		t.Error("leaf missing DNS SAN")
	}

	// Verify leaf is signed by CA
	caCert, _ := x509.ParseCertificate(ca.Certificate[0])
	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	if _, err := leafCert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
		t.Errorf("leaf does not chain to CA: %v", err)
	}

	// Cache hit
	leaf2, err := cache.GetCertificate(hello)
	if err != nil {
		t.Fatalf("GetCertificate (cached): %v", err)
	}
	if leaf2 != leaf {
		t.Error("expected same pointer for cached cert")
	}
}
