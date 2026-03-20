package grpcproxy

import (
	"testing"

	"github.com/SC-Bridge/sc-companion/internal/grpcproxy/descriptors"
)

func TestNewRegistry(t *testing.T) {
	r, err := NewRegistry(descriptors.DescriptorSet)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	if r.MethodCount() == 0 {
		t.Fatal("registry has no methods")
	}
	t.Logf("loaded %d methods", r.MethodCount())
}

func TestLookupMethod(t *testing.T) {
	r, err := NewRegistry(descriptors.DescriptorSet)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	info, ok := r.LookupMethod("/sc.external.services.ledger.v1.LedgerService/GetFunds")
	if !ok {
		t.Fatal("LedgerService/GetFunds not found in registry")
	}

	if info.Input == nil {
		t.Error("input descriptor is nil")
	}
	if info.Output == nil {
		t.Error("output descriptor is nil")
	}
	if info.ServiceName != "ledger" {
		t.Errorf("service name = %q, want %q", info.ServiceName, "ledger")
	}

	// GetFundsResponse has a 'ledgers' repeated field
	ledgersField := info.Output.Fields().ByName("ledgers")
	if ledgersField == nil {
		t.Error("output descriptor missing 'ledgers' field")
	}

	// Input has a 'ledger_name' field
	nameField := info.Input.Fields().ByName("ledger_name")
	if nameField == nil {
		t.Error("input descriptor missing 'ledger_name' field")
	}
}

func TestLookupMethodNotFound(t *testing.T) {
	r, err := NewRegistry(descriptors.DescriptorSet)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	_, ok := r.LookupMethod("/nonexistent.Service/Method")
	if ok {
		t.Error("expected not found for nonexistent method")
	}
}

func TestRegistryInvalidData(t *testing.T) {
	_, err := NewRegistry([]byte("not a valid protobuf"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

func TestExtractShortService(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sc.external.services.ledger.v1.LedgerService", "ledger"},
		{"sc.internal.services.reputation.v1.ReputationService", "reputation"},
		{"sc.external.services.push.v1.PushService", "push"},
		{"SomeService", "SomeService"},
		{"a.b", "a"},
	}

	for _, tt := range tests {
		got := extractShortService(tt.input)
		if got != tt.want {
			t.Errorf("extractShortService(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
