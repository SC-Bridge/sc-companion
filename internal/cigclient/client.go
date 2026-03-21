package cigclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/SC-Bridge/sc-companion/internal/grpcproxy/descriptors"
)

// WalletBalance represents a single ledger balance.
type WalletBalance struct {
	Name     string `json:"name"`
	Amount   uint64 `json:"amount"`
	Currency string `json:"currency"`
}

// Friend represents a friend with presence info.
type Friend struct {
	Nickname    string `json:"nickname"`
	DisplayName string `json:"displayName"`
	AccountID   uint32 `json:"accountId"`
	Status      string `json:"status"` // offline, online, away, etc.
	Activity    string `json:"activity"`
}

// ReputationScore represents a faction reputation score.
type ReputationScore struct {
	EntityID     string  `json:"entity_id"`
	Scope        string  `json:"scope"`
	Score        int32   `json:"score"`
	StandingTier string  `json:"standing_tier"`
	Drift        float64 `json:"drift"`
}

// ReputationHistoryEntry represents a single score history data point.
type ReputationHistoryEntry struct {
	EntityID       string `json:"entity_id"`
	Scope          string `json:"scope"`
	Score          uint64 `json:"score"`
	EventTimestamp uint32 `json:"event_timestamp"`
}

// Blueprint represents a blueprint collection entry.
type Blueprint struct {
	BlueprintID   string `json:"blueprint_id"`
	CategoryID    string `json:"category_id"`
	ItemClassID   string `json:"item_class_id"`
	Tier          uint32 `json:"tier"`
	RemainingUses int32  `json:"remaining_uses"`
	Source        string `json:"source"`
	ProcessType   string `json:"process_type"`
}

// Entitlement represents an in-game entitlement (ship, item).
type Entitlement struct {
	URN               string `json:"urn"`
	Name              string `json:"name"`
	EntityClassGUID   string `json:"entity_class_guid"`
	EntitlementType   string `json:"entitlement_type"`
	Status            string `json:"status"`
	ItemType          string `json:"item_type"`
	Source            string `json:"source"`
	InsuranceLifetime bool   `json:"insurance_lifetime"`
	InsuranceDuration uint64 `json:"insurance_duration"`
}

// Mission represents an active or recent mission.
type Mission struct {
	MissionID  string `json:"mission_id"`
	ContractID string `json:"contract_id"`
	Template   string `json:"template"`
	State      string `json:"state"`
	Title      string `json:"title"`
	RewardAUEC uint64 `json:"reward_auec"`
	ExpiresAt  string `json:"expires_at"`
	Objectives string `json:"objectives_json"` // JSON-encoded objectives
}

// PlayerStat represents a player statistic.
type PlayerStat struct {
	StatDefID string `json:"stat_def_id"`
	Value     uint32 `json:"value"`
	Best      uint32 `json:"best"`
	Category  string `json:"category"`
	GameMode  string `json:"game_mode"`
}

// Client is a direct gRPC client to CIG's game services.
type Client struct {
	mu       sync.Mutex
	conn     *grpc.ClientConn
	endpoint string
	token    string // JWT bearer token
	methods  map[string]methodInfo
}

type methodInfo struct {
	input  protoreflect.MessageDescriptor
	output protoreflect.MessageDescriptor
}

// NewClient creates a CIG API client from login data.
func NewClient(ld *LoginData) (*Client, error) {
	// Load method descriptors
	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(descriptors.DescriptorSet, fds); err != nil {
		return nil, fmt.Errorf("unmarshal descriptors: %w", err)
	}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		return nil, fmt.Errorf("build file registry: %w", err)
	}

	methods := make(map[string]methodInfo)
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			svc := fd.Services().Get(i)
			for j := 0; j < svc.Methods().Len(); j++ {
				m := svc.Methods().Get(j)
				path := fmt.Sprintf("/%s/%s", svc.FullName(), m.Name())
				methods[path] = methodInfo{input: m.Input(), output: m.Output()}
			}
		}
		return true
	})

	// Get the initial JWT by calling GetCurrentPlayer with the auth_token
	// StarBreaker does: identityClient.GetCurrentPlayer(req, {"Authorization": "Bearer " + authToken})
	// The response contains a JWT that's used for subsequent calls.
	c := &Client{
		endpoint: ld.StarNetwork.ServicesEndpoint,
		token:    ld.AuthToken,
		methods:  methods,
	}

	return c, nil
}

// Connect establishes the gRPC connection.
func (c *Client) Connect(ctx context.Context) error {
	// Set conn under lock, then release before calling getJWT (which also locks)
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}

	// gRPC expects host:port, not a URL — strip https:// prefix
	target := c.endpoint
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "http://")

	slog.Info("connecting to CIG endpoint", "target", target)

	// Use DialContext with explicit timeout to ensure connection actually establishes
	dialCtx, dialCancel := context.WithTimeout(ctx, 15*time.Second)
	defer dialCancel()

	//nolint:staticcheck // DialContext is deprecated but NewClient doesn't enforce timeouts
	cc, err := grpc.DialContext(dialCtx,
		target,
		grpc.WithTransportCredentials(credentials.NewTLS(nil)),
		grpc.WithBlock(),
	)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("dial CIG: %w", err)
	}
	c.conn = cc
	c.mu.Unlock()
	slog.Info("gRPC connection established")

	// Get the proper JWT via GetCurrentPlayer (with timeout)
	// NOTE: getJWT → call() takes c.mu internally, so we must NOT hold it here
	jwtCtx, jwtCancel := context.WithTimeout(ctx, 15*time.Second)
	defer jwtCancel()

	slog.Info("requesting JWT from IdentityService...")
	jwt, err := c.getJWT(jwtCtx)
	if err != nil {
		slog.Warn("failed to get JWT, using auth_token directly", "error", err)
		// Fall back to using auth_token directly
	} else {
		c.mu.Lock()
		c.token = jwt
		c.mu.Unlock()
		slog.Info("got JWT from IdentityService")
	}

	return nil
}

// getJWT calls IdentityService/GetCurrentPlayer to exchange auth_token for a JWT.
func (c *Client) getJWT(ctx context.Context) (string, error) {
	resp, err := c.call(ctx, "/sc.external.services.identity.v1.IdentityService/GetCurrentPlayer", nil)
	if err != nil {
		return "", err
	}

	// Extract JWT from response
	jwtField := resp.Descriptor().Fields().ByName("jwt")
	if jwtField == nil {
		return "", fmt.Errorf("no jwt field in GetCurrentPlayer response")
	}
	jwt := resp.Get(jwtField).String()
	if jwt == "" {
		return "", fmt.Errorf("empty jwt in response")
	}
	return jwt, nil
}

// GetWallet returns the player's wallet balances.
func (c *Client) GetWallet(ctx context.Context) ([]WalletBalance, error) {
	resp, err := c.call(ctx, "/sc.external.services.ledger.v1.LedgerService/GetFunds", nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := protojson.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	slog.Debug("GetFunds response", "json", string(jsonBytes))

	// Extract ledgers from response
	var wallets []WalletBalance
	ledgersField := resp.Descriptor().Fields().ByName("ledgers")
	if ledgersField == nil {
		return nil, fmt.Errorf("no ledgers field in response")
	}

	list := resp.Get(ledgersField).List()
	currencies := []string{"UNKNOWN", "UEC", "REC", "AUEC", "MER"}

	for i := 0; i < list.Len(); i++ {
		ledger := list.Get(i).Message()
		name := ledger.Get(ledger.Descriptor().Fields().ByName("name")).String()

		fundsField := ledger.Descriptor().Fields().ByName("funds")
		if fundsField == nil {
			continue
		}
		funds := ledger.Get(fundsField).Message()
		amount := funds.Get(funds.Descriptor().Fields().ByName("amount")).Uint()
		currencyNum := int(funds.Get(funds.Descriptor().Fields().ByName("currency")).Enum())
		currency := "UNKNOWN"
		if currencyNum >= 0 && currencyNum < len(currencies) {
			currency = currencies[currencyNum]
		}

		wallets = append(wallets, WalletBalance{
			Name:     name,
			Amount:   amount,
			Currency: currency,
		})
	}

	return wallets, nil
}

// GetFriends returns the player's friend list with presence.
func (c *Client) GetFriends(ctx context.Context) ([]Friend, error) {
	resp, err := c.call(ctx, "/sc.external.services.friends.v1.FriendService/GetFriendList", nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := protojson.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	slog.Debug("GetFriendList response", "json", string(jsonBytes))

	var friends []Friend
	friendsField := resp.Descriptor().Fields().ByName("friends")
	if friendsField == nil {
		return nil, fmt.Errorf("no friends field in response")
	}

	statuses := []string{"unspecified", "offline", "online", "away", "dnd", "activity", "invisible"}

	list := resp.Get(friendsField).List()
	for i := 0; i < list.Len(); i++ {
		friendMsg := list.Get(i).Message()

		// Get account info
		accountField := friendMsg.Descriptor().Fields().ByName("account")
		if accountField == nil {
			continue
		}
		account := friendMsg.Get(accountField).Message()
		nickname := account.Get(account.Descriptor().Fields().ByName("nickname")).String()
		displayName := account.Get(account.Descriptor().Fields().ByName("display_name")).String()
		accountID := uint32(account.Get(account.Descriptor().Fields().ByName("account_id")).Uint())

		// Get presence
		status := "unknown"
		activity := ""
		presenceField := friendMsg.Descriptor().Fields().ByName("presence")
		if presenceField != nil {
			presence := friendMsg.Get(presenceField).Message()
			statusField := presence.Descriptor().Fields().ByName("status")
			if statusField != nil {
				statusNum := int(presence.Get(statusField).Enum())
				if statusNum >= 0 && statusNum < len(statuses) {
					status = statuses[statusNum]
				}
			}
			activityField := presence.Descriptor().Fields().ByName("activity")
			if activityField != nil && presence.Has(activityField) {
				actMsg := presence.Get(activityField).Message()
				stateField := actMsg.Descriptor().Fields().ByName("state")
				if stateField != nil {
					activity = actMsg.Get(stateField).String()
				}
			}
		}

		friends = append(friends, Friend{
			Nickname:    nickname,
			DisplayName: displayName,
			AccountID:   accountID,
			Status:      status,
			Activity:    activity,
		})
	}

	return friends, nil
}

// GetReputation returns the player's reputation scores across all factions.
func (c *Client) GetReputation(ctx context.Context) ([]ReputationScore, error) {
	resp, err := c.call(ctx, "/sc.external.services.reputation.v1.ReputationService/QueryReputations", nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, _ := protojson.Marshal(resp)
	slog.Debug("QueryReputations response", "json", string(jsonBytes))

	var scores []ReputationScore
	resultsField := resp.Descriptor().Fields().ByName("results")
	if resultsField == nil {
		return nil, fmt.Errorf("no results field in QueryReputations response")
	}

	list := resp.Get(resultsField).List()
	for i := 0; i < list.Len(); i++ {
		vr := list.Get(i).Message()

		// VersionedReputation has nested reputation + standing
		repField := vr.Descriptor().Fields().ByName("reputation")
		if repField == nil {
			continue
		}
		rep := vr.Get(repField).Message()

		entityID := rep.Get(rep.Descriptor().Fields().ByName("entity")).String()
		scope := rep.Get(rep.Descriptor().Fields().ByName("scope")).String()
		score := int32(rep.Get(rep.Descriptor().Fields().ByName("score")).Int())

		// Extract standing tier name
		standingTier := ""
		standingField := vr.Descriptor().Fields().ByName("standing")
		if standingField != nil {
			standing := vr.Get(standingField).Message()
			nameField := standing.Descriptor().Fields().ByName("name")
			if nameField != nil {
				standingTier = standing.Get(nameField).String()
			}
		}

		// Extract drift from lock or standing
		var drift float64
		standingDriftField := func() *float64 {
			if standingField == nil {
				return nil
			}
			s := vr.Get(standingField).Message()
			df := s.Descriptor().Fields().ByName("drift")
			if df == nil {
				return nil
			}
			driftMsg := s.Get(df).Message()
			amountField := driftMsg.Descriptor().Fields().ByName("amount")
			if amountField == nil {
				return nil
			}
			v := float64(driftMsg.Get(amountField).Int())
			return &v
		}()
		if standingDriftField != nil {
			drift = *standingDriftField
		}

		scores = append(scores, ReputationScore{
			EntityID:     entityID,
			Scope:        scope,
			Score:        score,
			StandingTier: standingTier,
			Drift:        drift,
		})
	}

	return scores, nil
}

// GetReputationHistory returns score history for the given reputation IDs.
func (c *Client) GetReputationHistory(ctx context.Context, reputationIDs []string, days uint32) ([]ReputationHistoryEntry, error) {
	resp, err := c.call(ctx, "/sc.external.services.reputation.v1.ReputationService/GetScoreHistory", func(req *dynamicpb.Message) {
		// Build repeated ScoreHistory entries
		scoresField := req.Descriptor().Fields().ByName("reputation_scores")
		if scoresField == nil {
			return
		}
		list := req.Mutable(scoresField).List()
		for _, id := range reputationIDs {
			entry := dynamicpb.NewMessage(scoresField.Message())
			entry.Set(entry.Descriptor().Fields().ByName("reputation_id"), protoreflect.ValueOfString(id))
			entry.Set(entry.Descriptor().Fields().ByName("days"), protoreflect.ValueOfUint32(days))
			list.Append(protoreflect.ValueOfMessage(entry))
		}
	})
	if err != nil {
		return nil, err
	}

	jsonBytes, _ := protojson.Marshal(resp)
	slog.Debug("GetScoreHistory response", "json", string(jsonBytes))

	var entries []ReputationHistoryEntry
	resultsField := resp.Descriptor().Fields().ByName("reputation_scores")
	if resultsField == nil {
		return entries, nil
	}

	list := resp.Get(resultsField).List()
	for i := 0; i < list.Len(); i++ {
		histMsg := list.Get(i).Message()
		repID := histMsg.Get(histMsg.Descriptor().Fields().ByName("reputation_id")).String()

		scoresField := histMsg.Descriptor().Fields().ByName("scores")
		if scoresField == nil {
			continue
		}
		scoresList := histMsg.Get(scoresField).List()
		for j := 0; j < scoresList.Len(); j++ {
			scoreMsg := scoresList.Get(j).Message()
			entries = append(entries, ReputationHistoryEntry{
				EntityID:       repID,
				Scope:          "default",
				Score:          scoreMsg.Get(scoreMsg.Descriptor().Fields().ByName("score")).Uint(),
				EventTimestamp: uint32(scoreMsg.Get(scoreMsg.Descriptor().Fields().ByName("timestamp")).Uint()),
			})
		}
	}

	return entries, nil
}

// GetBlueprints returns the player's blueprint collection.
func (c *Client) GetBlueprints(ctx context.Context) ([]Blueprint, error) {
	resp, err := c.call(ctx, "/sc.external.services.blueprint_library.v1.BlueprintLibraryService/QueryBlueprintEntries", nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, _ := protojson.Marshal(resp)
	slog.Debug("QueryBlueprintEntries response", "json", string(jsonBytes))

	blueprintSources := []string{"UNSPECIFIED", "GAMEPLAY", "PLATFORM"}
	blueprintProcessTypes := []string{"UNSPECIFIED", "CREATE", "REFINE", "REPAIR", "UPGRADE", "DISMANTLE", "RESEARCH"}

	var blueprints []Blueprint
	resultsField := resp.Descriptor().Fields().ByName("results")
	if resultsField == nil {
		return blueprints, nil
	}

	list := resp.Get(resultsField).List()
	for i := 0; i < list.Len(); i++ {
		entry := list.Get(i).Message()

		blueprintID := entry.Get(entry.Descriptor().Fields().ByName("blueprint_id")).String()
		categoryID := entry.Get(entry.Descriptor().Fields().ByName("category_id")).String()
		itemClassID := entry.Get(entry.Descriptor().Fields().ByName("item_class_id")).String()
		tier := uint32(entry.Get(entry.Descriptor().Fields().ByName("tier")).Uint())
		remainingUses := int32(entry.Get(entry.Descriptor().Fields().ByName("remaining_uses")).Int())

		sourceNum := int(entry.Get(entry.Descriptor().Fields().ByName("source")).Enum())
		source := "UNSPECIFIED"
		if sourceNum >= 0 && sourceNum < len(blueprintSources) {
			source = blueprintSources[sourceNum]
		}

		processNum := int(entry.Get(entry.Descriptor().Fields().ByName("process_type")).Enum())
		processType := "UNSPECIFIED"
		if processNum >= 0 && processNum < len(blueprintProcessTypes) {
			processType = blueprintProcessTypes[processNum]
		}

		blueprints = append(blueprints, Blueprint{
			BlueprintID:   blueprintID,
			CategoryID:    categoryID,
			ItemClassID:   itemClassID,
			Tier:          tier,
			RemainingUses: remainingUses,
			Source:        source,
			ProcessType:   processType,
		})
	}

	return blueprints, nil
}

// GetEntitlements returns the player's entitlements (ships, items).
func (c *Client) GetEntitlements(ctx context.Context) ([]Entitlement, error) {
	resp, err := c.call(ctx, "/sc.external.services.entitlement.v1.ExternalEntitlementService/Query", nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, _ := protojson.Marshal(resp)
	slog.Debug("EntitlementService/Query response", "json", string(jsonBytes))

	entitlementTypes := []string{"UNSPECIFIED", "PERMANENT", "RENTAL"}
	entitlementStatuses := []string{"UNSPECIFIED", "PENDING", "FULFILLED", "REVOKED", "UNCLAIMED", "FAILED"}
	entitlementSources := []string{"UNSPECIFIED", "PLATFORM", "ARENA_COMMANDER", "STAR_MARINE", "PERSISTENT_UNIVERSE", "LONGTERM_PERSISTENCE"}
	entitlementItemTypes := []string{"UNSPECIFIED", "SHIP", "HANGAR", "HANGAR_DECORATION", "OTHER"}

	var entitlements []Entitlement
	resultsField := resp.Descriptor().Fields().ByName("results")
	if resultsField == nil {
		return entitlements, nil
	}

	list := resp.Get(resultsField).List()
	for i := 0; i < list.Len(); i++ {
		e := list.Get(i).Message()

		urn := e.Get(e.Descriptor().Fields().ByName("urn")).String()
		name := e.Get(e.Descriptor().Fields().ByName("name")).String()
		entityClassGUID := e.Get(e.Descriptor().Fields().ByName("entity_class_guid")).String()

		typeNum := int(e.Get(e.Descriptor().Fields().ByName("type")).Enum())
		eType := "PERMANENT"
		if typeNum >= 0 && typeNum < len(entitlementTypes) {
			eType = entitlementTypes[typeNum]
		}

		statusNum := int(e.Get(e.Descriptor().Fields().ByName("status")).Enum())
		eStatus := "UNSPECIFIED"
		if statusNum >= 0 && statusNum < len(entitlementStatuses) {
			eStatus = entitlementStatuses[statusNum]
		}

		sourceNum := int(e.Get(e.Descriptor().Fields().ByName("source")).Enum())
		eSource := "UNSPECIFIED"
		if sourceNum >= 0 && sourceNum < len(entitlementSources) {
			eSource = entitlementSources[sourceNum]
		}

		itemTypeNum := int(e.Get(e.Descriptor().Fields().ByName("item_type")).Enum())
		eItemType := "UNSPECIFIED"
		if itemTypeNum >= 0 && itemTypeNum < len(entitlementItemTypes) {
			eItemType = entitlementItemTypes[itemTypeNum]
		}

		// Extract insurance info
		isLifetime := false
		var duration uint64
		insuranceField := e.Descriptor().Fields().ByName("insurance")
		if insuranceField != nil && e.Has(insuranceField) {
			insurance := e.Get(insuranceField).Message()
			policyField := insurance.Descriptor().Fields().ByName("policy")
			if policyField != nil && insurance.Has(policyField) {
				policy := insurance.Get(policyField).Message()
				lifetimeField := policy.Descriptor().Fields().ByName("lifetime")
				if lifetimeField != nil && policy.Has(lifetimeField) {
					isLifetime = true
				}
				durationField := policy.Descriptor().Fields().ByName("duration")
				if durationField != nil && policy.Has(durationField) {
					dur := policy.Get(durationField).Message()
					expiresField := dur.Descriptor().Fields().ByName("expires_at")
					if expiresField != nil {
						duration = dur.Get(expiresField).Uint()
					}
				}
			}
		}

		entitlements = append(entitlements, Entitlement{
			URN:               urn,
			Name:              name,
			EntityClassGUID:   entityClassGUID,
			EntitlementType:   eType,
			Status:            eStatus,
			ItemType:          eItemType,
			Source:            eSource,
			InsuranceLifetime: isLifetime,
			InsuranceDuration: duration,
		})
	}

	return entitlements, nil
}

// GetActiveMissions returns active missions (2-step: get IDs, then details).
func (c *Client) GetActiveMissions(ctx context.Context) ([]Mission, error) {
	// Step 1: Get active mission IDs
	activeResp, err := c.call(ctx, "/sc.external.services.mission_service.v1.MissionService/QueryActiveMissions", nil)
	if err != nil {
		return nil, err
	}

	idsField := activeResp.Descriptor().Fields().ByName("mission_ids")
	if idsField == nil {
		return nil, nil
	}

	idsList := activeResp.Get(idsField).List()
	if idsList.Len() == 0 {
		return nil, nil
	}

	// Collect mission IDs
	var missionIDs []string
	for i := 0; i < idsList.Len(); i++ {
		body := idsList.Get(i).Message()
		mid := body.Get(body.Descriptor().Fields().ByName("mission_id")).String()
		if mid != "" {
			missionIDs = append(missionIDs, mid)
		}
	}

	if len(missionIDs) == 0 {
		return nil, nil
	}

	// Step 2: Get mission details
	detailResp, err := c.call(ctx, "/sc.external.services.mission_service.v1.MissionService/QueryMissions", func(req *dynamicpb.Message) {
		queriesField := req.Descriptor().Fields().ByName("queries")
		if queriesField == nil {
			return
		}
		list := req.Mutable(queriesField).List()
		for _, mid := range missionIDs {
			body := dynamicpb.NewMessage(queriesField.Message())
			body.Set(body.Descriptor().Fields().ByName("mission_id"), protoreflect.ValueOfString(mid))
			list.Append(protoreflect.ValueOfMessage(body))
		}
	})
	if err != nil {
		return nil, err
	}

	jsonBytes, _ := protojson.Marshal(detailResp)
	slog.Debug("QueryMissions response", "json", string(jsonBytes))

	missionStates := []string{"UNSPECIFIED", "PENDING", "ACTIVE", "SUSPENDED", "COMPLETED", "FAILED", "EXPIRED", "ENDED", "WITHDRAWN"}

	var missions []Mission
	missionsField := detailResp.Descriptor().Fields().ByName("missions")
	if missionsField == nil {
		return missions, nil
	}

	missionsList := detailResp.Get(missionsField).List()
	for i := 0; i < missionsList.Len(); i++ {
		m := missionsList.Get(i).Message()

		missionID := m.Get(m.Descriptor().Fields().ByName("mission_id")).String()
		contractID := m.Get(m.Descriptor().Fields().ByName("contract_id")).String()

		// Extract template contract_definition_id
		template := ""
		templateField := m.Descriptor().Fields().ByName("mission_template")
		if templateField != nil && m.Has(templateField) {
			tmpl := m.Get(templateField).Message()
			defField := tmpl.Descriptor().Fields().ByName("contract_definition_id")
			if defField != nil {
				template = tmpl.Get(defField).String()
			}
		}

		stateNum := int(m.Get(m.Descriptor().Fields().ByName("mission_state")).Enum())
		state := "UNSPECIFIED"
		if stateNum >= 0 && stateNum < len(missionStates) {
			state = missionStates[stateNum]
		}

		// Extract reward aUEC — MissionReward structure is complex, best-effort
		var rewardAUEC uint64
		rewardField := m.Descriptor().Fields().ByName("reward")
		if rewardField != nil && m.Has(rewardField) {
			reward := m.Get(rewardField).Message()
			// Try to find a currency_amount or similar field
			for k := 0; k < reward.Descriptor().Fields().Len(); k++ {
				f := reward.Descriptor().Fields().Get(k)
				slog.Debug("mission reward field", "name", f.Name(), "kind", f.Kind())
			}
		}

		// Extract expiry timestamp
		expiresAt := ""
		expiryField := m.Descriptor().Fields().ByName("expiry")
		if expiryField != nil && m.Has(expiryField) {
			expiry := m.Get(expiryField).Message()
			secondsField := expiry.Descriptor().Fields().ByName("seconds")
			if secondsField != nil {
				secs := expiry.Get(secondsField).Int()
				if secs > 0 {
					expiresAt = fmt.Sprintf("%d", secs)
				}
			}
		}

		// Serialize objectives as JSON
		objectives := ""
		objField := m.Descriptor().Fields().ByName("mission_objectives")
		if objField != nil {
			objList := m.Get(objField).List()
			if objList.Len() > 0 {
				// Marshal the whole objectives list to JSON
				type objSummary struct {
					ID       string `json:"id"`
					State    int    `json:"state"`
					Progress int64  `json:"progress"`
					Max      int64  `json:"max"`
				}
				var objs []objSummary
				for j := 0; j < objList.Len(); j++ {
					obj := objList.Get(j).Message()
					objID := obj.Get(obj.Descriptor().Fields().ByName("objective_id")).String()
					objState := int(obj.Get(obj.Descriptor().Fields().ByName("state")).Enum())
					progress := obj.Get(obj.Descriptor().Fields().ByName("progress_counter_current")).Int()
					max := obj.Get(obj.Descriptor().Fields().ByName("progress_counter_max")).Int()
					objs = append(objs, objSummary{ID: objID, State: objState, Progress: progress, Max: max})
				}
				if objJSON, err := json.Marshal(objs); err == nil {
					objectives = string(objJSON)
				}
			}
		}

		missions = append(missions, Mission{
			MissionID:  missionID,
			ContractID: contractID,
			Template:   template,
			State:      state,
			RewardAUEC: rewardAUEC,
			ExpiresAt:  expiresAt,
			Objectives: objectives,
		})
	}

	return missions, nil
}

// GetStats returns the player's stats.
func (c *Client) GetStats(ctx context.Context) ([]PlayerStat, error) {
	resp, err := c.call(ctx, "/sc.external.services.stats.v1.StatsService/FindStats", nil)
	if err != nil {
		return nil, err
	}

	jsonBytes, _ := protojson.Marshal(resp)
	slog.Debug("FindStats response", "json", string(jsonBytes))

	var stats []PlayerStat
	resultsField := resp.Descriptor().Fields().ByName("results")
	if resultsField == nil {
		return stats, nil
	}

	list := resp.Get(resultsField).List()
	for i := 0; i < list.Len(); i++ {
		s := list.Get(i).Message()

		statDefID := s.Get(s.Descriptor().Fields().ByName("stat_def_id")).String()
		value := uint32(s.Get(s.Descriptor().Fields().ByName("value")).Uint())
		best := uint32(s.Get(s.Descriptor().Fields().ByName("best")).Uint())
		category := s.Get(s.Descriptor().Fields().ByName("category")).String()
		gameMode := s.Get(s.Descriptor().Fields().ByName("game_mode")).String()

		stats = append(stats, PlayerStat{
			StatDefID: statDefID,
			Value:     value,
			Best:      best,
			Category:  category,
			GameMode:  gameMode,
		})
	}

	return stats, nil
}

// call makes a unary gRPC call using dynamicpb.
// Uses the standard proto codec — passes *dynamicpb.Message directly to Invoke
// so gRPC sends Content-Type: application/grpc+proto (which CIG servers expect).
func (c *Client) call(ctx context.Context, method string, setFields func(*dynamicpb.Message)) (*dynamicpb.Message, error) {
	c.mu.Lock()
	conn := c.conn
	token := c.token
	c.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	mi, ok := c.methods[method]
	if !ok {
		return nil, fmt.Errorf("unknown method: %s", method)
	}

	req := dynamicpb.NewMessage(mi.input)
	if setFields != nil {
		setFields(req)
	}

	// Ensure a timeout exists — default 15s if none set
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		deadline, _ = ctx.Deadline()
	}

	slog.Info("gRPC call starting",
		"method", method,
		"token_len", len(token),
		"deadline", deadline.Format(time.RFC3339),
		"req_type", string(req.ProtoReflect().Descriptor().FullName()),
	)

	// Create outgoing context with auth
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})
	callCtx := metadata.NewOutgoingContext(ctx, md)

	// Pass dynamicpb.Message directly — gRPC's default proto codec handles
	// serialization via the proto.Message interface that dynamicpb implements.
	resp := dynamicpb.NewMessage(mi.output)

	// Run Invoke in a goroutine so we can log if it hangs past the deadline
	done := make(chan error, 1)
	go func() {
		done <- conn.Invoke(callCtx, method, req, resp)
	}()

	select {
	case err := <-done:
		if err != nil {
			slog.Error("gRPC call failed", "method", method, "error", err)
			return nil, fmt.Errorf("invoke %s: %w", method, err)
		}
		slog.Info("gRPC call succeeded", "method", method)
		return resp, nil
	case <-ctx.Done():
		slog.Error("gRPC call timed out — Invoke did not respect context deadline",
			"method", method,
			"error", ctx.Err(),
		)
		return nil, fmt.Errorf("invoke %s: %w", method, ctx.Err())
	}
}

// Close shuts down the connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}
