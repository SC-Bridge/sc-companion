package grpcproxy

import (
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/SC-Bridge/sc-companion/internal/events"
	"github.com/SC-Bridge/sc-companion/internal/grpcproxy/descriptors"
)

func TestDecodeGetFundsRequest(t *testing.T) {
	registry, err := NewRegistry(descriptors.DescriptorSet)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	bus := events.NewBus()

	var mu sync.Mutex
	var received []events.Event
	bus.Subscribe(func(evt events.Event) {
		mu.Lock()
		received = append(received, evt)
		mu.Unlock()
	})

	decoder := NewDecoder(registry, bus)

	// Build a GetFundsRequest with ledger_name="player_wallet"
	info, ok := registry.LookupMethod("/sc.external.services.ledger.v1.LedgerService/GetFunds")
	if !ok {
		t.Fatal("method not found")
	}

	msg := dynamicpb.NewMessage(info.Input)
	nameField := info.Input.Fields().ByName("ledger_name")
	if nameField == nil {
		t.Fatal("ledger_name field not found")
	}
	msg.Set(nameField, protoreflect.ValueOfString("player_wallet"))

	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	decoder.Decode("/sc.external.services.ledger.v1.LedgerService/GetFunds", DirectionRequest, payload)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) == 0 {
		t.Fatal("no events received")
	}

	evt := received[0]
	if evt.Type != "grpc.ledger.GetFunds" {
		t.Errorf("event type = %q, want %q", evt.Type, "grpc.ledger.GetFunds")
	}
	if evt.Source != "grpc" {
		t.Errorf("source = %q, want %q", evt.Source, "grpc")
	}
	if evt.Data["ledger_name"] != "player_wallet" {
		t.Errorf("ledger_name = %q, want %q", evt.Data["ledger_name"], "player_wallet")
	}
	if evt.Data["direction"] != "request" {
		t.Errorf("direction = %q, want %q", evt.Data["direction"], "request")
	}
}

func TestDecodeBlockedService(t *testing.T) {
	registry, err := NewRegistry(descriptors.DescriptorSet)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	bus := events.NewBus()
	var received []events.Event
	bus.Subscribe(func(evt events.Event) {
		received = append(received, evt)
	})

	decoder := NewDecoder(registry, bus)
	decoder.Decode("/sc.external.services.login.v1.LoginService/Login", DirectionRequest, []byte{})

	if len(received) != 0 {
		t.Error("blocked service should not produce events")
	}
}

func TestDecodeUnknownMethod(t *testing.T) {
	registry, err := NewRegistry(descriptors.DescriptorSet)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	bus := events.NewBus()
	var received []events.Event
	bus.Subscribe(func(evt events.Event) {
		received = append(received, evt)
	})

	decoder := NewDecoder(registry, bus)
	decoder.Decode("/unknown.Service/Method", DirectionRequest, []byte{})

	if len(received) != 0 {
		t.Error("unknown method should not produce events")
	}
}

func TestFormatEventType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/sc.external.services.ledger.v1.LedgerService/GetFunds", "grpc.ledger.GetFunds"},
		{"/sc.internal.services.reputation.v1.ReputationService/Get", "grpc.reputation.Get"},
		{"/invalid", "grpc.unknown"},
	}

	for _, tt := range tests {
		got := formatEventType(tt.input)
		if got != tt.want {
			t.Errorf("formatEventType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
