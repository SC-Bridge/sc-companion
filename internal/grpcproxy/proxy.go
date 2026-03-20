package grpcproxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"

	"google.golang.org/grpc"

	"github.com/SC-Bridge/sc-companion/internal/events"
	"github.com/SC-Bridge/sc-companion/internal/grpcproxy/descriptors"
)

// Proxy intercepts gRPC traffic via TLS MITM. Supports two modes:
//   - HTTP CONNECT mode (proxy env var approach)
//   - Direct TLS mode (hosts file redirect approach)
type Proxy struct {
	listenAddr  string
	directMode  bool
	backendAddr string // only used in direct mode
	bus         *events.Bus
	certCache   *LeafCertCache
	handler     *Handler

	mu      sync.RWMutex
	running bool
}

// ProxyConfig holds proxy configuration.
type ProxyConfig struct {
	ListenAddr string // e.g. "127.0.0.1:8443"
	CADir      string // directory containing ca.crt and ca.key

	// Direct TLS mode (hosts file redirect approach):
	// Instead of HTTP CONNECT, listen as a raw TLS server and forward
	// all connections to BackendAddr.
	DirectMode  bool   // if true, use direct TLS instead of HTTP CONNECT
	BackendAddr string // real backend IP:port, e.g. "34.11.124.51:443"
}

// NewProxy creates a new HTTP CONNECT proxy with gRPC interception.
func NewProxy(cfg ProxyConfig, bus *events.Bus) (*Proxy, error) {
	// Load or generate CA
	ca, created, err := EnsureCA(cfg.CADir)
	if err != nil {
		return nil, fmt.Errorf("ensure CA: %w", err)
	}
	if created {
		slog.Info("new CA certificate generated — install it to intercept gRPC traffic",
			"path", CACertPath(cfg.CADir),
			"install", fmt.Sprintf("certutil -addstore Root \"%s\"", CACertPath(cfg.CADir)),
		)
	}

	certCache, err := NewLeafCertCache(ca)
	if err != nil {
		return nil, fmt.Errorf("create cert cache: %w", err)
	}

	// Load proto registry
	registry, err := NewRegistry(descriptors.DescriptorSet)
	if err != nil {
		return nil, fmt.Errorf("load proto registry: %w", err)
	}
	slog.Info("proto registry loaded", "methods", registry.MethodCount())

	// Build decoder + handler
	decoder := NewDecoder(registry, bus)
	handler := NewHandler(decoder)

	p := &Proxy{
		listenAddr:  cfg.ListenAddr,
		directMode:  cfg.DirectMode,
		backendAddr: cfg.BackendAddr,
		bus:         bus,
		certCache:   certCache,
		handler:     handler,
	}

	return p, nil
}

// Run starts the proxy. Blocks until ctx is cancelled.
func (p *Proxy) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", p.listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	p.mu.Lock()
	p.running = true
	p.mu.Unlock()

	mode := "HTTP CONNECT"
	if p.directMode {
		mode = fmt.Sprintf("direct TLS → %s", p.backendAddr)
	}
	slog.Info("gRPC proxy listening", "addr", p.listenAddr, "mode", mode)

	go func() {
		<-ctx.Done()
		p.handler.Close()
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
		if p.directMode {
			go p.handleDirect(conn)
		} else {
			go p.handleConnect(conn)
		}
	}
}

// IsRunning returns whether the proxy is active.
func (p *Proxy) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// handleDirect handles a connection in direct TLS mode (hosts file redirect).
// The client connects directly to us thinking we're the CIG server.
// No HTTP CONNECT — we immediately do TLS and forward to the real backend.
func (p *Proxy) handleDirect(conn net.Conn) {
	defer conn.Close()

	slog.Debug("direct connection from", "remote", conn.RemoteAddr())

	// Wrap in TLS — we present a cert for whatever hostname the client expects
	tlsConfig := &tls.Config{
		GetCertificate: p.certCache.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
	}

	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		slog.Warn("direct TLS handshake failed", "error", err)
		return
	}
	defer tlsConn.Close()

	slog.Info("direct TLS connection",
		"sni", tlsConn.ConnectionState().ServerName,
		"alpn", tlsConn.ConnectionState().NegotiatedProtocol,
	)

	negotiatedProto := tlsConn.ConnectionState().NegotiatedProtocol

	if negotiatedProto == "h2" {
		p.serveGRPC(tlsConn, p.backendAddr)
	} else {
		p.passthrough(tlsConn, p.backendAddr)
	}
}

// handleConnect processes an HTTP CONNECT request.
func (p *Proxy) handleConnect(conn net.Conn) {
	defer conn.Close()

	br := bufio.NewReader(conn)
	req, err := http.ReadRequest(br)
	if err != nil {
		slog.Debug("failed to read CONNECT request", "error", err)
		return
	}

	if req.Method != http.MethodConnect {
		resp := &http.Response{
			StatusCode: http.StatusMethodNotAllowed,
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
		resp.Write(conn)
		return
	}

	targetHost := req.Host
	slog.Debug("CONNECT", "target", targetHost)

	// Respond 200 to establish the tunnel
	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Wrap the connection in TLS, presenting a leaf cert for the target host
	tlsConfig := &tls.Config{
		GetCertificate: p.certCache.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
	}

	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		slog.Debug("TLS handshake failed", "target", targetHost, "error", err)
		return
	}
	defer tlsConn.Close()

	negotiatedProto := tlsConn.ConnectionState().NegotiatedProtocol

	if negotiatedProto == "h2" {
		// HTTP/2 → hand to gRPC server for interception
		p.serveGRPC(tlsConn, targetHost)
	} else {
		// Not h2 → plain passthrough to real backend
		p.passthrough(tlsConn, targetHost)
	}
}

// serveGRPC creates a per-connection gRPC server and forwards all calls
// to the real backend while decoding payloads asynchronously.
func (p *Proxy) serveGRPC(conn net.Conn, targetHost string) {
	// Ensure the target has a port for dialing
	backendAddr := targetHost
	if !strings.Contains(backendAddr, ":") {
		backendAddr = backendAddr + ":443"
	}

	// Create a per-connection gRPC server with the target baked in
	srv := grpc.NewServer(
		grpc.ForceServerCodec(RawCodec{}),
		grpc.UnknownServiceHandler(p.handler.TransparentHandler(backendAddr)),
	)

	// Serve on a single-connection listener
	ln := &singleConnListener{conn: conn}
	srv.Serve(ln)
}

// passthrough does a bidirectional TCP copy to the real backend for non-gRPC traffic.
func (p *Proxy) passthrough(clientConn net.Conn, targetHost string) {
	if !strings.Contains(targetHost, ":") {
		targetHost = targetHost + ":443"
	}

	backendConn, err := tls.Dial("tcp", targetHost, &tls.Config{})
	if err != nil {
		slog.Debug("passthrough dial failed", "target", targetHost, "error", err)
		return
	}
	defer backendConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(backendConn, clientConn)
	}()
	go func() {
		defer wg.Done()
		io.Copy(clientConn, backendConn)
	}()
	wg.Wait()
}

// singleConnListener is a net.Listener that serves exactly one connection.
type singleConnListener struct {
	conn net.Conn
	done bool
	mu   sync.Mutex
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	if l.done {
		l.mu.Unlock()
		// Block until the connection closes — gRPC server will notice
		// when it tries to read from the closed connection.
		select {}
	}
	l.done = true
	conn := l.conn
	l.mu.Unlock()
	return conn, nil
}

func (l *singleConnListener) Close() error   { return nil }
func (l *singleConnListener) Addr() net.Addr { return l.conn.LocalAddr() }

// Context key for target host (used if we need context-based lookup later).
type contextKey string

const targetKey contextKey = "grpc-proxy-target"

// targetFromContext extracts the backend target host from a context.
func targetFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(targetKey)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
