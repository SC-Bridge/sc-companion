package logtailer

import (
	"regexp"
	"strings"
	"time"

	"github.com/SC-Bridge/sc-companion/internal/events"
)

// Parser extracts structured events from raw Game.log lines.
type Parser struct {
	patterns []pattern
	// Multi-line notification accumulator
	pendingNotification string
}

type pattern struct {
	name    string
	re      *regexp.Regexp
	extract func(matches []string) events.Event
}

// NewParser creates a parser with all known log patterns.
func NewParser() *Parser {
	p := &Parser{}
	p.patterns = []pattern{
		// Player identity
		{
			name: "player_login",
			re:   regexp.MustCompile(`nickname="([^"]+)"\s+playerGEID=(\d+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "player_login", Source: "log",
					Data: map[string]string{"handle": m[1], "geid": m[2]},
				}
			},
		},
		// Ship boarding — "You have joined channel 'Ship Name : OwnerHandle'"
		{
			name: "ship_boarded",
			re:   regexp.MustCompile(`Added notification "You have joined channel '([^']+)'`),
			extract: func(m []string) events.Event {
				channel := m[1]
				ship, owner := parseShipChannel(channel)
				return events.Event{
					Type: "ship_boarded", Source: "log",
					Data: map[string]string{"ship": ship, "owner": owner, "raw": channel},
				}
			},
		},
		// Ship exiting
		{
			name: "ship_exited",
			re:   regexp.MustCompile(`Added notification "You have left the channel '([^']+)'`),
			extract: func(m []string) events.Event {
				channel := m[1]
				ship, owner := parseShipChannel(channel)
				return events.Event{
					Type: "ship_exited", Source: "log",
					Data: map[string]string{"ship": ship, "owner": owner, "raw": channel},
				}
			},
		},
		// Contract accepted
		{
			name: "contract_accepted",
			re:   regexp.MustCompile(`Added notification "Contract Accepted:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "contract_accepted", Source: "log",
					Data: map[string]string{"name": strings.TrimSpace(m[1])},
				}
			},
		},
		// Contract completed
		{
			name: "contract_completed",
			re:   regexp.MustCompile(`Added notification "Contract Complete:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "contract_completed", Source: "log",
					Data: map[string]string{"name": strings.TrimSpace(m[1])},
				}
			},
		},
		// Contract failed
		{
			name: "contract_failed",
			re:   regexp.MustCompile(`Added notification "Contract Failed:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "contract_failed", Source: "log",
					Data: map[string]string{"name": strings.TrimSpace(m[1])},
				}
			},
		},
		// Quantum travel target selected
		{
			name: "qt_target_selected",
			re:   regexp.MustCompile(`Player has selected point (\S+) as their destination`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "qt_target_selected", Source: "log",
					Data: map[string]string{"destination": m[1]},
				}
			},
		},
		// Quantum travel arrived
		{
			name: "qt_arrived",
			re:   regexp.MustCompile(`Quantum Drive has arrived at final destination`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "qt_arrived", Source: "log", Data: map[string]string{}}
			},
		},
		// Location change
		{
			name: "location_change",
			re:   regexp.MustCompile(`<RequestLocationInventory> Player\[([^\]]+)\] requested inventory for Location\[([^\]]+)\]`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "location_change", Source: "log",
					Data: map[string]string{"player": m[1], "location": m[2]},
				}
			},
		},
		// Jurisdiction change
		{
			name: "jurisdiction_entered",
			re:   regexp.MustCompile(`Added notification "Entered ([^"]+) Jurisdiction`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "jurisdiction_entered", Source: "log",
					Data: map[string]string{"jurisdiction": m[1]},
				}
			},
		},
		// Armistice zone
		{
			name: "armistice_entered",
			re:   regexp.MustCompile(`Added notification "Entering Armistice Zone`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "armistice_entered", Source: "log", Data: map[string]string{}}
			},
		},
		{
			name: "armistice_exited",
			re:   regexp.MustCompile(`Added notification "Leaving Armistice Zone`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "armistice_exited", Source: "log", Data: map[string]string{}}
			},
		},
		// Monitored space
		{
			name: "monitored_space_entered",
			re:   regexp.MustCompile(`Added notification "Entered Monitored Space`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "monitored_space_entered", Source: "log", Data: map[string]string{}}
			},
		},
		{
			name: "monitored_space_exited",
			re:   regexp.MustCompile(`Added notification "Exited Monitored Space`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "monitored_space_exited", Source: "log", Data: map[string]string{}}
			},
		},
		// CrimeStat
		{
			name: "crimestat_increased",
			re:   regexp.MustCompile(`Added notification "CrimeStat Rating Increased`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "crimestat_increased", Source: "log", Data: map[string]string{}}
			},
		},
		// Fines
		{
			name: "fined",
			re:   regexp.MustCompile(`Added notification "Fined (\d+) UEC`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "fined", Source: "log",
					Data: map[string]string{"amount": m[1], "currency": "UEC"},
				}
			},
		},
		// Money sent (multi-line: "You sent PlayerName:\nAMOUNT aUEC\n : ")
		{
			name: "money_sent",
			re:   regexp.MustCompile(`Added notification "You sent ([^:]*?):\s*$`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "money_sent_pending", Source: "log",
					Data: map[string]string{"recipient": strings.TrimSpace(m[1])},
				}
			},
		},
		// Money amount line (continuation of money_sent)
		{
			name: "money_amount",
			re:   regexp.MustCompile(`^\s*(\d+)\s+aUEC\s*$`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "money_amount", Source: "log",
					Data: map[string]string{"amount": m[1]},
				}
			},
		},
		// Injury
		{
			name: "injury",
			re:   regexp.MustCompile(`Added notification "(Minor|Major|Severe) Injury Detected - ([^-]+) - Tier (\d+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "injury", Source: "log",
					Data: map[string]string{
						"severity":  m[1],
						"body_part": strings.TrimSpace(m[2]),
						"tier":      m[3],
					},
				}
			},
		},
		// Incapacitated
		{
			name: "incapacitated",
			re:   regexp.MustCompile(`Added notification "Incapacitated:`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "incapacitated", Source: "log", Data: map[string]string{}}
			},
		},
		// Insurance claim
		{
			name: "insurance_claim",
			re:   regexp.MustCompile(`<CWallet::ProcessClaimToNextStep> New Insurance Claim Request - entitlementURN: ([^,]+), requestId\s*:\s*(\d+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "insurance_claim", Source: "log",
					Data: map[string]string{"urn": m[1], "request_id": m[2]},
				}
			},
		},
		// Insurance claim complete
		{
			name: "insurance_claim_complete",
			re:   regexp.MustCompile(`<CWallet::RmMulticastOnProcessClaimCallback> Claim Complete - entitlementURN: ([^,]+), result:\s*(\d+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "insurance_claim_complete", Source: "log",
					Data: map[string]string{"urn": m[1], "result": m[2]},
				}
			},
		},
		// Blueprint received (4.7+)
		{
			name: "blueprint_received",
			re:   regexp.MustCompile(`Added notification "Received Blueprint:\s*(.+?):`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "blueprint_received", Source: "log",
					Data: map[string]string{"name": strings.TrimSpace(m[1])},
				}
			},
		},
		// Refinery complete
		{
			name: "refinery_complete",
			re:   regexp.MustCompile(`Added notification "A Refinery Work Order has been Completed at (.+?):`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "refinery_complete", Source: "log",
					Data: map[string]string{"location": strings.TrimSpace(m[1])},
				}
			},
		},
		// Transaction complete
		{
			name: "transaction_complete",
			re:   regexp.MustCompile(`Added notification "Transaction Complete`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "transaction_complete", Source: "log", Data: map[string]string{}}
			},
		},
		// Vehicle impounded
		{
			name: "vehicle_impounded",
			re:   regexp.MustCompile(`Added notification "Vehicle Impounded:\s*(.+?):`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "vehicle_impounded", Source: "log",
					Data: map[string]string{"reason": strings.TrimSpace(m[1])},
				}
			},
		},
		// Fatal collision
		{
			name: "fatal_collision",
			re:   regexp.MustCompile(`<FatalCollision> Fatal Collision occured for vehicle (\S+).*Zone:\s*([^,\]]+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "fatal_collision", Source: "log",
					Data: map[string]string{"vehicle": m[1], "zone": strings.TrimSpace(m[2])},
				}
			},
		},
		// Server join
		{
			name: "server_joined",
			re:   regexp.MustCompile(`Join PU - address\[([^\]]+)\].*shard\[([^\]]+)\]`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "server_joined", Source: "log",
					Data: map[string]string{"address": m[1], "shard": m[2]},
				}
			},
		},
		// Rewards earned
		{
			name: "rewards_earned",
			re:   regexp.MustCompile(`Added notification "You've Earned:\s*(\d+)\s+Rewards`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "rewards_earned", Source: "log",
					Data: map[string]string{"count": m[1]},
				}
			},
		},
	}
	return p
}

// Parse attempts to extract an event from a log line.
func (p *Parser) Parse(line string) (events.Event, bool) {
	// Extract timestamp if present
	ts := extractTimestamp(line)

	for _, pat := range p.patterns {
		matches := pat.re.FindStringSubmatch(line)
		if matches != nil {
			evt := pat.extract(matches)
			evt.Timestamp = ts
			return evt, true
		}
	}
	return events.Event{}, false
}

// parseShipChannel splits "Manufacturer Ship : Owner" into ship and owner.
func parseShipChannel(channel string) (ship, owner string) {
	// Clean @vehicle_Name prefix if present
	channel = strings.TrimPrefix(channel, "@vehicle_Name")

	parts := strings.SplitN(channel, " : ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return channel, ""
}

// extractTimestamp pulls the ISO 8601 timestamp from the start of a log line.
func extractTimestamp(line string) time.Time {
	// Format: <2026-03-18T18:30:55.361Z>
	if len(line) < 28 || line[0] != '<' {
		return time.Time{}
	}
	end := strings.IndexByte(line, '>')
	if end < 2 {
		return time.Time{}
	}
	ts, err := time.Parse(time.RFC3339Nano, line[1:end])
	if err != nil {
		return time.Time{}
	}
	return ts
}
