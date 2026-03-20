package grpcproxy

import "testing"

func TestIsBlocked(t *testing.T) {
	tests := []struct {
		service string
		want    bool
	}{
		{"login", true},
		{"eatransaction", true},
		{"ledger", false},
		{"reputation", false},
		{"push", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsBlocked(tt.service)
		if got != tt.want {
			t.Errorf("IsBlocked(%q) = %v, want %v", tt.service, got, tt.want)
		}
	}
}

func TestRedactFields(t *testing.T) {
	data := map[string]string{
		"email":      "user@example.com",
		"account_id": "12345",
		"jwt":        "secret-token",
		"handle":     "player1",
	}

	RedactFields("identity", data)

	if _, ok := data["email"]; ok {
		t.Error("email should be redacted")
	}
	if _, ok := data["account_id"]; ok {
		t.Error("account_id should be redacted")
	}
	if _, ok := data["jwt"]; ok {
		t.Error("jwt should be redacted")
	}
	if _, ok := data["handle"]; !ok {
		t.Error("handle should NOT be redacted")
	}
}

func TestRedactFieldsNoMatch(t *testing.T) {
	data := map[string]string{
		"funds": "1000",
	}
	RedactFields("ledger", data)
	if data["funds"] != "1000" {
		t.Error("non-matching service should not redact")
	}
}
