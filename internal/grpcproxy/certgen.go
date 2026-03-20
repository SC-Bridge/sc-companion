package grpcproxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EnsureCA loads or generates a CA certificate and key in the given directory.
// Returns the CA cert, whether it was newly created, and any error.
func EnsureCA(dir string) (tls.Certificate, bool, error) {
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	// Try loading existing CA
	if cert, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		slog.Info("loaded existing CA", "path", certPath)
		return cert, false, nil
	}

	// Generate new CA
	if err := os.MkdirAll(dir, 0700); err != nil {
		return tls.Certificate{}, false, fmt.Errorf("create CA directory: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return tls.Certificate{}, false, fmt.Errorf("generate CA key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, false, fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"SC Bridge Companion"},
			CommonName:   "SC Bridge Local CA",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, false, fmt.Errorf("create CA certificate: %w", err)
	}

	// Write cert
	certFile, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return tls.Certificate{}, false, fmt.Errorf("write CA cert: %w", err)
	}
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile.Close()

	// Write key (restricted permissions)
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return tls.Certificate{}, false, fmt.Errorf("write CA key: %w", err)
	}
	pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	keyFile.Close()

	slog.Info("generated new CA certificate", "path", certPath)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, false, fmt.Errorf("reload CA: %w", err)
	}

	return cert, true, nil
}

// CACertPath returns the expected CA certificate path for a given data directory.
func CACertPath(dir string) string {
	return filepath.Join(dir, "ca.crt")
}

// LeafCertCache generates and caches per-host TLS certificates signed by the CA.
type LeafCertCache struct {
	ca    tls.Certificate
	caCert *x509.Certificate
	cache sync.Map // hostname → *tls.Certificate
}

// NewLeafCertCache creates a certificate cache backed by the given CA.
func NewLeafCertCache(ca tls.Certificate) (*LeafCertCache, error) {
	caCert, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}
	return &LeafCertCache{
		ca:     ca,
		caCert: caCert,
	}, nil
}

// GetCertificate implements tls.Config.GetCertificate. It returns a cached
// leaf certificate for the requested hostname, generating one if needed.
func (c *LeafCertCache) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := hello.ServerName
	if host == "" {
		host = "localhost"
	}

	if cached, ok := c.cache.Load(host); ok {
		return cached.(*tls.Certificate), nil
	}

	cert, err := c.generateLeaf(host)
	if err != nil {
		return nil, err
	}

	c.cache.Store(host, cert)
	return cert, nil
}

func (c *LeafCertCache) generateLeaf(hostname string) (*tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate leaf key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"SC Bridge Companion"},
			CommonName:   hostname,
		},
		DNSNames:  []string{hostname},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour), // Short-lived leaf certs
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, c.caCert, &key.PublicKey, c.ca.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("create leaf certificate: %w", err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{certDER, c.ca.Certificate[0]},
		PrivateKey:  key,
	}, nil
}
