package grpcproxy

import (
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/SC-Bridge/sc-companion/internal/events"
)

// Direction indicates whether a message is a request or response.
type Direction int

const (
	DirectionRequest  Direction = 0
	DirectionResponse Direction = 1
)

func (d Direction) String() string {
	if d == DirectionRequest {
		return "request"
	}
	return "response"
}

// Decoder decodes raw protobuf payloads into typed events using the registry.
type Decoder struct {
	registry *Registry
	bus      *events.Bus
}

// NewDecoder creates a decoder backed by a proto registry and event bus.
func NewDecoder(registry *Registry, bus *events.Bus) *Decoder {
	return &Decoder{
		registry: registry,
		bus:      bus,
	}
}

// Decode attempts to decode a raw protobuf payload for the given gRPC method
// and publishes the result as an event. Designed to be called from a goroutine
// so it never blocks the forwarding path.
func (d *Decoder) Decode(fullMethod string, dir Direction, payload []byte) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("decoder panic recovered", "method", fullMethod, "panic", r)
		}
	}()
	info, ok := d.registry.LookupMethod(fullMethod)
	if !ok {
		return
	}

	if IsBlocked(info.ServiceName) {
		return
	}

	// Pick the right descriptor based on direction
	var desc protoreflect.MessageDescriptor
	if dir == DirectionRequest {
		desc = info.Input
	} else {
		desc = info.Output
	}

	msg := dynamicpb.NewMessage(desc)
	if err := proto.Unmarshal(payload, msg); err != nil {
		slog.Debug("proto unmarshal failed", "method", fullMethod, "dir", dir, "error", err)
		return
	}

	// Flatten to map[string]string
	data := flattenMessage(msg)
	RedactFields(info.ServiceName, data)

	// Build event type: grpc.{service}.{Method}
	eventType := formatEventType(fullMethod)
	data["direction"] = dir.String()
	data["method"] = fullMethod

	d.bus.Publish(events.Event{
		Type:   eventType,
		Source: "grpc",
		Data:   data,
	})

	// Special handling for PushService: unwrap google.protobuf.Any
	d.tryUnwrapPush(fullMethod, info.ServiceName, msg, data)
}

// tryUnwrapPush checks if this is a PushService message containing a
// google.protobuf.Any envelope. If so, it tries to decode the inner
// message and emit a more specific event.
func (d *Decoder) tryUnwrapPush(fullMethod, serviceName string, msg *dynamicpb.Message, parentData map[string]string) {
	if !strings.Contains(strings.ToLower(serviceName), "push") {
		return
	}

	// Look for an Any-typed field (skip list/map fields)
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() != protoreflect.MessageKind {
			return true
		}
		if fd.IsList() || fd.IsMap() {
			return true
		}
		innerMsg, ok := v.Message().Interface().(*dynamicpb.Message)
		if !ok {
			return true
		}

		// Try to interpret as Any
		anyMsg := &anypb.Any{}
		innerBytes, err := proto.Marshal(innerMsg)
		if err != nil {
			return true
		}
		if err := proto.Unmarshal(innerBytes, anyMsg); err != nil || anyMsg.TypeUrl == "" {
			return true
		}

		// Extract the type name from the Any URL
		typeName := anyMsg.TypeUrl
		if idx := strings.LastIndex(typeName, "/"); idx >= 0 {
			typeName = typeName[idx+1:]
		}

		// Emit a typed push event
		pushType := fmt.Sprintf("grpc.push.%s", strings.ReplaceAll(typeName, ".", "_"))
		pushData := make(map[string]string)
		for k, v := range parentData {
			pushData[k] = v
		}
		pushData["any_type_url"] = anyMsg.TypeUrl
		pushData["any_type"] = typeName

		d.bus.Publish(events.Event{
			Type:   pushType,
			Source: "grpc",
			Data:   pushData,
		})

		return false // stop after first Any
	})
}

// formatEventType converts "/sc.external.services.ledger.v1.LedgerService/GetFunds"
// to "grpc.ledger.GetFunds".
func formatEventType(fullMethod string) string {
	// Split "/package.Service/Method"
	parts := strings.Split(strings.TrimPrefix(fullMethod, "/"), "/")
	if len(parts) != 2 {
		return "grpc.unknown"
	}

	serviceFull := parts[0]
	method := parts[1]

	// Extract short service name
	short := extractShortService(serviceFull)

	return fmt.Sprintf("grpc.%s.%s", short, method)
}

// flattenMessage converts a dynamic protobuf message to a flat map[string]string.
// Nested messages are flattened with dot-separated keys.
func flattenMessage(msg *dynamicpb.Message) map[string]string {
	result := make(map[string]string)
	flattenFields(msg, "", result)
	return result
}

func flattenFields(msg *dynamicpb.Message, prefix string, out map[string]string) {
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		key := string(fd.Name())
		if prefix != "" {
			key = prefix + "." + key
		}

		switch fd.Kind() {
		case protoreflect.MessageKind, protoreflect.GroupKind:
			if fd.IsList() {
				list := v.List()
				for i := 0; i < list.Len(); i++ {
					itemKey := fmt.Sprintf("%s.%d", key, i)
					if inner, ok := list.Get(i).Message().Interface().(*dynamicpb.Message); ok {
						flattenFields(inner, itemKey, out)
					}
				}
			} else if inner, ok := v.Message().Interface().(*dynamicpb.Message); ok {
				flattenFields(inner, key, out)
			}
		case protoreflect.EnumKind:
			out[key] = fmt.Sprintf("%d", v.Enum())
		default:
			out[key] = fmt.Sprintf("%v", v.Interface())
		}
		return true
	})
}
