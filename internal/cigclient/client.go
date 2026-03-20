package cigclient

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}

	cc, err := grpc.NewClient(
		c.endpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(nil)),
	)
	if err != nil {
		return fmt.Errorf("dial CIG: %w", err)
	}
	c.conn = cc

	// Get the proper JWT via GetCurrentPlayer
	jwt, err := c.getJWT(ctx)
	if err != nil {
		slog.Warn("failed to get JWT, using auth_token directly", "error", err)
		// Fall back to using auth_token directly
	} else {
		c.token = jwt
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

// call makes a unary gRPC call using dynamicpb.
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

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create outgoing context with auth
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})
	callCtx := metadata.NewOutgoingContext(ctx, md)

	// Make the call
	respBytes := make([]byte, 0)
	err = conn.Invoke(callCtx, method, reqBytes, &respBytes)
	if err != nil {
		return nil, fmt.Errorf("invoke %s: %w", method, err)
	}

	resp := dynamicpb.NewMessage(mi.output)
	if err := proto.Unmarshal(respBytes, resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return resp, nil
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
