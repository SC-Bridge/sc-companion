package grpcproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/SC-Bridge/sc-companion/internal/events"
)

// blockedServices are excluded from capture (PII / real money).
var blockedServices = map[string]bool{
	"login":          true,
	"eatransaction":  true,
}

// Proxy is a transparent TLS MITM proxy that intercepts gRPC traffic
// to CIG's backend and extracts game data from protobuf payloads.
type Proxy struct {
	listenAddr string
	targetHost string
	bus        *events.Bus
	caCert     tls.Certificate
	mu         sync.RWMutex
	running    bool
}

// Config holds gRPC proxy configuration.
type Config struct {
	ListenAddr string // local address to listen on (e.g. ":8443")
	TargetHost string // CIG backend (e.g. "pub-sc-alpha-460-11135423.test1.cloudimperiumgames.com:443")
	CACertFile string // path to CA certificate for MITM
	CAKeyFile  string // path to CA private key for MITM
}

// New creates a new gRPC MITM proxy.
func New(cfg Config, bus *events.Bus) (*Proxy, error) {
	caCert, err := tls.LoadX509KeyPair(cfg.CACertFile, cfg.CAKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load CA cert: %w", err)
	}

	return &Proxy{
		listenAddr: cfg.ListenAddr,
		targetHost: cfg.TargetHost,
		bus:        bus,
		caCert:     caCert,
	}, nil
}

// Run starts the proxy listener. Blocks until ctx is cancelled.
func (p *Proxy) Run(ctx context.Context) error {
	// Generate a TLS config that presents certs signed by our CA
	tlsConfig := &tls.Config{
		GetCertificate: p.getCertificate,
	}

	listener, err := tls.Listen("tcp", p.listenAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	p.mu.Lock()
	p.running = true
	p.mu.Unlock()

	slog.Info("gRPC proxy listening", "addr", p.listenAddr, "target", p.targetHost)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("accept error", "error", err)
			continue
		}
		go p.handleConn(ctx, conn)
	}
}

// IsRunning returns whether the proxy is active.
func (p *Proxy) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

func (p *Proxy) handleConn(ctx context.Context, clientConn net.Conn) {
	defer clientConn.Close()

	// Connect to the real CIG backend
	backendConn, err := tls.Dial("tcp", p.targetHost, &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         stripPort(p.targetHost),
	})
	if err != nil {
		slog.Error("backend dial failed", "error", err)
		return
	}
	defer backendConn.Close()

	// Bidirectional copy with sniffing
	var wg sync.WaitGroup
	wg.Add(2)

	// Client → Backend (requests)
	go func() {
		defer wg.Done()
		p.copyWithSniff(backendConn, clientConn, "request")
	}()

	// Backend → Client (responses)
	go func() {
		defer wg.Done()
		p.copyWithSniff(clientConn, backendConn, "response")
	}()

	wg.Wait()
}

func (p *Proxy) copyWithSniff(dst, src net.Conn, direction string) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			// Sniff the gRPC frame for service/method info
			p.sniffFrame(buf[:n], direction)

			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// sniffFrame attempts to extract gRPC metadata from an HTTP/2 frame.
// This is a best-effort parser — full HTTP/2 frame parsing would be more robust
// but this catches the :path header which contains the service/method.
func (p *Proxy) sniffFrame(data []byte, direction string) {
	// Look for gRPC :path header in HTTP/2 HEADERS frame
	// Format: /package.ServiceName/MethodName
	content := string(data)

	// Find service paths like "/sc.external.services.ledger.v1.LedgerService/GetFunds"
	for i := 0; i < len(content)-1; i++ {
		if content[i] == '/' && i+1 < len(content) && content[i+1] == 's' {
			// Look for "sc.external.services." or "sc.internal.services."
			end := strings.IndexByte(content[i+1:], 0)
			if end < 0 {
				end = len(content) - i - 1
			}
			path := content[i : i+1+end]

			if strings.Contains(path, "sc.external.services.") || strings.Contains(path, "sc.internal.services.") {
				parts := strings.Split(path, "/")
				if len(parts) >= 3 {
					serviceFull := parts[1]
					method := parts[2]

					// Extract short service name for filtering
					serviceParts := strings.Split(serviceFull, ".")
					shortName := ""
					if len(serviceParts) >= 4 {
						shortName = serviceParts[3] // e.g. "ledger", "reputation"
					}

					if blockedServices[shortName] {
						return
					}

					p.bus.Publish(events.Event{
						Type:   "grpc_call",
						Source: "grpc",
						Data: map[string]string{
							"service":   serviceFull,
							"method":    method,
							"direction": direction,
							"short":     shortName,
						},
					})
				}
				return
			}
		}
	}
}

func (p *Proxy) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// Generate a certificate for the requested hostname, signed by our CA
	cert, err := generateCert(hello.ServerName, &p.caCert)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func stripPort(hostPort string) string {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	return host
}

// generateCert creates a TLS certificate for the given hostname, signed by the CA.
func generateCert(hostname string, ca *tls.Certificate) (*tls.Certificate, error) {
	caCert, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, err
	}

	// For now, return the CA cert itself — a proper implementation would
	// generate a per-host certificate signed by the CA.
	// TODO: implement proper per-host cert generation
	_ = caCert
	_ = hostname
	return ca, nil
}
