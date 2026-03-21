package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// OAuthResult contains the result of an OAuth flow.
type OAuthResult struct {
	Token string
	Error error
}

// OAuthFlow manages the temporary HTTP server for the OAuth callback.
type OAuthFlow struct {
	state    string
	port     int
	listener net.Listener
	server   *http.Server
	resultCh chan OAuthResult
	endpoint string
}

// NewOAuthFlow creates a new OAuth flow with a random state and port.
func NewOAuthFlow(endpoint string) (*OAuthFlow, error) {
	// Generate 32-byte hex state
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Listen on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	flow := &OAuthFlow{
		state:    state,
		port:     port,
		listener: listener,
		resultCh: make(chan OAuthResult, 1),
		endpoint: endpoint,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", flow.handleCallback)

	flow.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return flow, nil
}

// ConnectURL returns the URL to open in the browser.
func (f *OAuthFlow) ConnectURL() string {
	// Strip /api suffix from endpoint to get base URL
	base := f.endpoint
	if len(base) > 4 && base[len(base)-4:] == "/api" {
		base = base[:len(base)-4]
	}
	return fmt.Sprintf("%s/companion/connect?port=%d&state=%s", base, f.port, f.state)
}

// Start begins serving the callback endpoint. Returns when a result is received or timeout.
func (f *OAuthFlow) Start(ctx context.Context) OAuthResult {
	// Start server in background
	go func() {
		if err := f.server.Serve(f.listener); err != http.ErrServerClosed {
			slog.Error("oauth server error", "error", err)
		}
	}()

	// Wait for result or timeout (5 minutes)
	timeout := time.After(5 * time.Minute)
	select {
	case result := <-f.resultCh:
		f.shutdown()
		return result
	case <-timeout:
		f.shutdown()
		return OAuthResult{Error: fmt.Errorf("connection timed out after 5 minutes")}
	case <-ctx.Done():
		f.shutdown()
		return OAuthResult{Error: ctx.Err()}
	}
}

func (f *OAuthFlow) handleCallback(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	state := r.URL.Query().Get("state")

	if state != f.state {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		f.resultCh <- OAuthResult{Error: fmt.Errorf("state mismatch")}
		return
	}

	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		f.resultCh <- OAuthResult{Error: fmt.Errorf("missing token")}
		return
	}

	// Redirect browser to success page
	base := f.endpoint
	if len(base) > 4 && base[len(base)-4:] == "/api" {
		base = base[:len(base)-4]
	}
	http.Redirect(w, r, base+"/companion/connected", http.StatusFound)

	// Send result
	f.resultCh <- OAuthResult{Token: token}
}

func (f *OAuthFlow) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	f.server.Shutdown(ctx)
}
