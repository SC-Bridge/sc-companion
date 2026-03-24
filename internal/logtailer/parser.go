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
			re:   regexp.MustCompile(`Added notification "You have joined channel '(.+?)'"`),
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
			re:   regexp.MustCompile(`Added notification "You have left the channel '(.+?)'"`),
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
			re:   regexp.MustCompile(`Added notification "(Minor|Moderate|Major|Severe) Injury Detected - ([^-]+) - Tier (\d+)`),
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
		// Insurance claim — strip account URN, keep only request ID
		{
			name: "insurance_claim",
			re:   regexp.MustCompile(`<CWallet::ProcessClaimToNextStep> New Insurance Claim Request.*requestId\s*:\s*(\d+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "insurance_claim", Source: "log",
					Data: map[string]string{"request_id": m[1]},
				}
			},
		},
		// Insurance claim complete — strip account URN, keep result
		{
			name: "insurance_claim_complete",
			re:   regexp.MustCompile(`<CWallet::RmMulticastOnProcessClaimCallback> Claim Complete.*result:\s*(\d+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "insurance_claim_complete", Source: "log",
					Data: map[string]string{"result": m[1]},
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
		// Server join — capture shard only, not IP address
		{
			name: "server_joined",
			re:   regexp.MustCompile(`Join PU.*shard\[([^\]]+)\]`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "server_joined", Source: "log",
					Data: map[string]string{"shard": m[1]},
				}
			},
		},
		// Rewards earned
		{
			name: "rewards_earned",
			re:   regexp.MustCompile(`Added notification "You've earned:\s*(\d+)\s+rewards`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "rewards_earned", Source: "log",
					Data: map[string]string{"count": m[1]},
				}
			},
		},

		// --- New patterns: mission lifecycle ---

		// Mission ended (push message from server)
		{
			name: "mission_ended",
			re:   regexp.MustCompile(`<MissionEnded> Received MissionEnded push message for: mission_id (\S+) - mission_state (\S+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "mission_ended", Source: "log",
					Data: map[string]string{"mission_id": m[1], "state": m[2]},
				}
			},
		},
		// End mission (local processing)
		{
			name: "end_mission",
			re:   regexp.MustCompile(`<EndMission> Ending mission.*MissionId\[([^\]]+)\].*Player\[([^\]]+)\].*CompletionType\[([^\]]+)\].*Reason\[([^\]]+)\]`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "end_mission", Source: "log",
					Data: map[string]string{
						"mission_id":      m[1],
						"player":          m[2],
						"completion_type": m[3],
						"reason":          m[4],
					},
				}
			},
		},
		// New mission objective
		{
			name: "new_objective",
			re:   regexp.MustCompile(`Added notification "New Objective:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "new_objective", Source: "log",
					Data: map[string]string{"name": strings.TrimSpace(m[1])},
				}
			},
		},
		// Contract available (offered to player)
		{
			name: "contract_available",
			re:   regexp.MustCompile(`Added notification "Contract Available:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "contract_available", Source: "log",
					Data: map[string]string{"name": strings.TrimSpace(m[1])},
				}
			},
		},

		// --- New patterns: ship/vehicle ---

		// Hangar request completed
		{
			name: "hangar_ready",
			re:   regexp.MustCompile(`Added notification "Hangar Request Completed`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "hangar_ready", Source: "log", Data: map[string]string{}}
			},
		},
		// Ship list fetched from ASOP — count of insured entitlements
		{
			name: "ship_list_fetched",
			re:   regexp.MustCompile(`<CEntityComponentShipListProvider::FetchShipData::<lambda_1>.*Received (\d+) player insured entitlements.*player (\d+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "ship_list_fetched", Source: "log",
					Data: map[string]string{"count": m[1]},
				}
			},
		},
		// Ship spawned — vehicle list complete
		{
			name: "ships_loaded",
			re:   regexp.MustCompile(`<CEntityComponentShipListProvider::FetchOwnedShipsData.*Fetching vehicle list.*completed\. Retrieved (\d+) vehicles`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "ships_loaded", Source: "log",
					Data: map[string]string{"count": m[1]},
				}
			},
		},

		// --- New patterns: quantum travel ---

		// QT fuel requested
		{
			name: "qt_fuel_requested",
			re:   regexp.MustCompile(`<Player Requested Fuel to Quantum Target - Local>.*destination (\S+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "qt_fuel_requested", Source: "log",
					Data: map[string]string{"destination": m[1]},
				}
			},
		},

		// --- New patterns: economy/services ---

		// Emergency services
		{
			name: "emergency_services",
			re:   regexp.MustCompile(`Added notification "Standby, Local Emergency Services`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "emergency_services", Source: "log", Data: map[string]string{}}
			},
		},

		// --- New patterns: account reconciliation ---

		// Entitlement reconciliation (login) — capture count only, not account URN
		{
			name: "entitlement_reconciliation",
			re:   regexp.MustCompile(`<ReconcileAccountUpdateNotification>.*details ([^-]+).*status (\S+) - phase (\S+)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "entitlement_reconciliation", Source: "log",
					Data: map[string]string{
						"details": strings.TrimSpace(m[1]),
						"status":  m[2],
						"phase":   m[3],
					},
				}
			},
		},

		// --- Mission / contract lifecycle ---

		// Contract shared by party member
		{
			name: "contract_shared",
			re:   regexp.MustCompile(`Added notification "Contract Shared:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "contract_shared", Source: "log",
					Data: map[string]string{"name": strings.TrimSpace(m[1])},
				}
			},
		},
		// Objective completed
		{
			name: "objective_complete",
			re:   regexp.MustCompile(`Added notification "Objective Complete:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "objective_complete", Source: "log",
					Data: map[string]string{"description": strings.TrimSpace(m[1])},
				}
			},
		},
		// Objective withdrawn
		{
			name: "objective_withdrawn",
			re:   regexp.MustCompile(`Added notification "Objective Withdrawn:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "objective_withdrawn", Source: "log",
					Data: map[string]string{"description": strings.TrimSpace(m[1])},
				}
			},
		},

		// --- Zone / area transitions ---

		// Armistice zone — exiting (distinct wording from "Leaving")
		{
			name: "armistice_exiting",
			re:   regexp.MustCompile(`Added notification "Exiting Armistice Zone`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "armistice_exiting", Source: "log", Data: map[string]string{}}
			},
		},
		// Private property
		{
			name: "private_property_entered",
			re:   regexp.MustCompile(`Added notification "Entering Private Property`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "private_property_entered", Source: "log", Data: map[string]string{}}
			},
		},
		{
			name: "private_property_exited",
			re:   regexp.MustCompile(`Added notification "Leaving Private Property`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "private_property_exited", Source: "log", Data: map[string]string{}}
			},
		},
		// Restricted area
		{
			name: "restricted_area_warning",
			re:   regexp.MustCompile(`Added notification "Restricted Area - Vehicles Will Be Impounded`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "restricted_area_warning", Source: "log", Data: map[string]string{}}
			},
		},
		{
			name: "restricted_area_exited",
			re:   regexp.MustCompile(`Added notification "Leaving Restricted Area`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "restricted_area_exited", Source: "log", Data: map[string]string{}}
			},
		},
		// Monitored space infrastructure
		{
			name: "monitored_space_down",
			re:   regexp.MustCompile(`Added notification "Monitored Space Down`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "monitored_space_down", Source: "log", Data: map[string]string{}}
			},
		},
		{
			name: "monitored_space_restored",
			re:   regexp.MustCompile(`Added notification "Monitored Space Restored`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "monitored_space_restored", Source: "log", Data: map[string]string{}}
			},
		},

		// --- Crime ---

		{
			name: "crime_committed",
			re:   regexp.MustCompile(`Added notification "Crime Committed:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "crime_committed", Source: "log",
					Data: map[string]string{"crime": strings.TrimSpace(m[1])},
				}
			},
		},

		// --- Party ---

		{
			name: "party_member_joined",
			re:   regexp.MustCompile(`Added notification "New Member Joined`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "party_member_joined", Source: "log", Data: map[string]string{}}
			},
		},
		{
			name: "party_member_left",
			re:   regexp.MustCompile(`Added notification "Member Left`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "party_member_left", Source: "log", Data: map[string]string{}}
			},
		},
		{
			name: "party_disbanded",
			re:   regexp.MustCompile(`Added notification "Party Disbanded`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "party_disbanded", Source: "log", Data: map[string]string{}}
			},
		},

		// --- Misc notifications ---

		{
			name: "low_fuel",
			re:   regexp.MustCompile(`Added notification "Low Fuel`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "low_fuel", Source: "log", Data: map[string]string{}}
			},
		},
		{
			name: "journal_entry_added",
			re:   regexp.MustCompile(`Added notification "Journal Entry Added:\s*(.+?):\s*"`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "journal_entry_added", Source: "log",
					Data: map[string]string{"entry": strings.TrimSpace(m[1])},
				}
			},
		},

		// --- Player state (non-notification) ---

		// Player spawned / respawned into the world
		{
			name: "player_spawned",
			re:   regexp.MustCompile(`\[CSessionManager::OnClientSpawned\] Spawned!`),
			extract: func(m []string) events.Event {
				return events.Event{Type: "player_spawned", Source: "log", Data: map[string]string{}}
			},
		},
		// Actor death — fires when the local player dies inside a destroyed vehicle zone
		{
			name: "actor_death",
			re:   regexp.MustCompile(`<\[ActorState\] Dead>.*Actor '([^']+)'.*ejected from zone '([^']+)'`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "actor_death", Source: "log",
					Data: map[string]string{"actor": m[1], "zone": m[2]},
				}
			},
		},
		// Medical bed treatment
		{
			name: "med_bed_heal",
			re:   regexp.MustCompile(`<MED BED HEAL> Actor: (\S+).*med bed name: ([^,]+), vehicle name: ([^,]+), head: (true|false) torso: (true|false) leftArm: (true|false) rightArm: (true|false) leftLeg: (true|false) rightLeg: (true|false)`),
			extract: func(m []string) events.Event {
				return events.Event{
					Type: "med_bed_heal", Source: "log",
					Data: map[string]string{
						"actor":      m[1],
						"bed_name":   strings.TrimSpace(m[2]),
						"vehicle":    strings.TrimSpace(m[3]),
						"head":       m[4],
						"torso":      m[5],
						"left_arm":   m[6],
						"right_arm":  m[7],
						"left_leg":   m[8],
						"right_leg":  m[9],
					},
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
