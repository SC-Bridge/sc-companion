package grpcproxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// RawCodec is a gRPC codec that passes bytes through without protobuf parsing.
// This lets UnknownServiceHandler forward messages without knowing the schema.
type RawCodec struct{}

// rawFrame wraps raw bytes for the codec.
type rawFrame struct {
	payload []byte
}

func (RawCodec) Marshal(v interface{}) ([]byte, error) {
	f, ok := v.(*rawFrame)
	if !ok {
		return nil, fmt.Errorf("rawcodec: expected *rawFrame, got %T", v)
	}
	return f.payload, nil
}

func (RawCodec) Unmarshal(data []byte, v interface{}) error {
	f, ok := v.(*rawFrame)
	if !ok {
		return fmt.Errorf("rawcodec: expected *rawFrame, got %T", v)
	}
	f.payload = make([]byte, len(data))
	copy(f.payload, data)
	return nil
}

func (RawCodec) Name() string { return "raw" }

// Handler manages gRPC forwarding and decoding.
type Handler struct {
	decoder  *Decoder
	backends sync.Map // hostname → *grpc.ClientConn
}

// NewHandler creates a gRPC forwarding handler.
func NewHandler(decoder *Decoder) *Handler {
	return &Handler{
		decoder: decoder,
	}
}

// TransparentHandler returns a grpc.StreamHandler for use with
// grpc.UnknownServiceHandler. It forwards all calls to the real backend
// while capturing payloads for async decoding.
func (h *Handler) TransparentHandler(backendAddr string) grpc.StreamHandler {
	return func(srv interface{}, serverStream grpc.ServerStream) error {
		fullMethod, ok := grpc.Method(serverStream.Context())
		if !ok {
			return fmt.Errorf("no method in stream context")
		}

		slog.Debug("forwarding gRPC call", "method", fullMethod, "backend", backendAddr)

		// Get or create backend connection
		cc, err := h.getBackend(backendAddr)
		if err != nil {
			return fmt.Errorf("dial backend: %w", err)
		}

		// Forward metadata from client
		md, _ := metadata.FromIncomingContext(serverStream.Context())
		ctx := metadata.NewOutgoingContext(serverStream.Context(), md)

		// Open stream to backend
		desc := &grpc.StreamDesc{
			ServerStreams: true,
			ClientStreams: true,
		}
		clientStream, err := cc.NewStream(ctx, desc, fullMethod, grpc.ForceCodec(RawCodec{}))
		if err != nil {
			return fmt.Errorf("open backend stream: %w", err)
		}

		// Bidirectional forwarding with two goroutines
		var wg sync.WaitGroup
		var forwardErr error

		// Client → Backend (requests)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				f := &rawFrame{}
				if err := serverStream.RecvMsg(f); err != nil {
					if err == io.EOF {
						clientStream.CloseSend()
						return
					}
					slog.Debug("client recv error", "method", fullMethod, "error", err)
					clientStream.CloseSend()
					return
				}

				// Async decode request
				payload := f.payload
				go h.decoder.Decode(fullMethod, DirectionRequest, payload)

				if err := clientStream.SendMsg(f); err != nil {
					slog.Debug("backend send error", "method", fullMethod, "error", err)
					return
				}
			}
		}()

		// Backend → Client (responses)
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Forward response headers
			if header, err := clientStream.Header(); err == nil {
				serverStream.SendHeader(header)
			}

			for {
				f := &rawFrame{}
				if err := clientStream.RecvMsg(f); err != nil {
					if err == io.EOF {
						// Forward trailers
						serverStream.SetTrailer(clientStream.Trailer())
						return
					}
					forwardErr = err
					return
				}

				// Async decode response
				payload := f.payload
				go h.decoder.Decode(fullMethod, DirectionResponse, payload)

				if err := serverStream.SendMsg(f); err != nil {
					slog.Debug("client send error", "method", fullMethod, "error", err)
					return
				}
			}
		}()

		wg.Wait()
		return forwardErr
	}
}

// getBackend returns a cached or new gRPC client connection to the backend.
func (h *Handler) getBackend(addr string) (*grpc.ClientConn, error) {
	if cc, ok := h.backends.Load(addr); ok {
		return cc.(*grpc.ClientConn), nil
	}

	cc, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(RawCodec{})),
	)
	if err != nil {
		return nil, err
	}

	actual, loaded := h.backends.LoadOrStore(addr, cc)
	if loaded {
		// Another goroutine beat us — close our duplicate
		cc.Close()
	}
	return actual.(*grpc.ClientConn), nil
}

// Close shuts down all backend connections.
func (h *Handler) Close() {
	h.backends.Range(func(key, value interface{}) bool {
		value.(*grpc.ClientConn).Close()
		h.backends.Delete(key)
		return true
	})
}
