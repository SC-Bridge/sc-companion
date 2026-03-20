package grpcproxy

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// MethodInfo describes a single gRPC method's input/output descriptors.
type MethodInfo struct {
	Input       protoreflect.MessageDescriptor
	Output      protoreflect.MessageDescriptor
	IsStreaming  bool // true if client or server streaming
	ServiceName string // short service name, e.g. "ledger"
}

// Registry indexes all gRPC methods from a compiled FileDescriptorSet.
type Registry struct {
	methods map[string]MethodInfo // key: "/package.Service/Method"
}

// NewRegistry loads a serialized FileDescriptorSet and builds a method index.
func NewRegistry(data []byte) (*Registry, error) {
	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fds); err != nil {
		return nil, fmt.Errorf("unmarshal descriptor set: %w", err)
	}

	files, err := protodesc.NewFiles(fds)
	if err != nil {
		return nil, fmt.Errorf("build file registry: %w", err)
	}

	r := &Registry{
		methods: make(map[string]MethodInfo),
	}

	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			svc := fd.Services().Get(i)
			for j := 0; j < svc.Methods().Len(); j++ {
				method := svc.Methods().Get(j)
				fullPath := fmt.Sprintf("/%s/%s", svc.FullName(), method.Name())

				r.methods[fullPath] = MethodInfo{
					Input:       method.Input(),
					Output:      method.Output(),
					IsStreaming:  method.IsStreamingClient() || method.IsStreamingServer(),
					ServiceName: extractShortService(string(svc.FullName())),
				}
			}
		}
		return true
	})

	return r, nil
}

// LookupMethod returns method info for a gRPC path like "/pkg.Service/Method".
func (r *Registry) LookupMethod(fullMethod string) (MethodInfo, bool) {
	info, ok := r.methods[fullMethod]
	return info, ok
}

// MethodCount returns the number of indexed methods.
func (r *Registry) MethodCount() int {
	return len(r.methods)
}

// extractShortService pulls the domain-specific service name from a fully
// qualified protobuf service name. For "sc.external.services.ledger.v1.LedgerService"
// it returns "ledger".
func extractShortService(fullName string) string {
	parts := strings.Split(fullName, ".")
	// Look for the name after "services." — typically index 3 in
	// "sc.external.services.{name}.v1.ServiceName"
	for i, p := range parts {
		if p == "services" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	// Fallback: use second-to-last part
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return fullName
}
