package grpcproxy

// blockedServices are excluded entirely — never decoded, never logged.
// These carry PII or real-money transaction data.
var blockedServices = map[string]bool{
	"login":         true,
	"eatransaction": true,
}

// redactedFields lists fields to strip from decoded messages, keyed by short
// service name. These fields may contain PII that isn't needed for game data.
var redactedFields = map[string][]string{
	"identity": {"email", "account_id", "jwt", "token", "refresh_token"},
	"friends":  {"email"},
}

// IsBlocked returns true if the service should not be intercepted at all.
func IsBlocked(shortService string) bool {
	return blockedServices[shortService]
}

// RedactFields removes sensitive fields from a decoded event's data map.
func RedactFields(shortService string, data map[string]string) {
	fields, ok := redactedFields[shortService]
	if !ok {
		return
	}
	for _, f := range fields {
		delete(data, f)
	}
}
