package user

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/egov/ws-services/internal/domain"
)

// Client integrates with egov-user for resolving / creating connection-holder
// users. Disabled => holder enrichment skipped.
type Client struct {
	HTTP       *http.Client
	Host       string
	SearchPath string
	CreatePath string
	Enabled    bool
}

func New(host, searchPath, createPath string, enabled bool) *Client {
	return &Client{
		HTTP:       &http.Client{Timeout: 10 * time.Second},
		Host:       host,
		SearchPath: searchPath,
		CreatePath: createPath,
		Enabled:    enabled,
	}
}

// User is the slice of the egov-user payload the connection holder needs.
type User struct {
	ID           int64  `json:"id,omitempty"`
	UUID         string `json:"uuid,omitempty"`
	UserName     string `json:"userName,omitempty"`
	Name         string `json:"name,omitempty"`
	MobileNumber string `json:"mobileNumber,omitempty"`
	EmailID      string `json:"emailId,omitempty"`
	Type         string `json:"type,omitempty"`
	TenantID     string `json:"tenantId,omitempty"`
}

type userResponse struct {
	User []User `json:"user"`
}

// SearchByMobile returns users matching a mobile number in a tenant.
func (c *Client) SearchByMobile(ctx context.Context, ri *domain.RequestInfo, tenantID, mobile string) ([]User, error) {
	body, _ := json.Marshal(map[string]any{
		"RequestInfo":  ri,
		"tenantId":     tenantID,
		"mobileNumber": mobile,
	})
	return c.call(ctx, c.Host+c.SearchPath, body)
}

// Create registers a new citizen user and returns it (with uuid populated).
func (c *Client) Create(ctx context.Context, ri *domain.RequestInfo, u User) (*User, error) {
	if u.Type == "" {
		u.Type = "CITIZEN"
	}
	body, _ := json.Marshal(map[string]any{"RequestInfo": ri, "user": u})
	users, err := c.call(ctx, c.Host+c.CreatePath, body)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user create: empty response")
	}
	return &users[0], nil
}

// Resolve returns the uuid for a holder's mobile number, creating the user when
// none exists. Used to populate connection-holder userid on create.
func (c *Client) Resolve(ctx context.Context, ri *domain.RequestInfo, tenantID, name, mobile string) (string, error) {
	if mobile != "" {
		existing, err := c.SearchByMobile(ctx, ri, tenantID, mobile)
		if err != nil {
			return "", err
		}
		if len(existing) > 0 {
			return existing[0].UUID, nil
		}
	}
	created, err := c.Create(ctx, ri, User{Name: name, MobileNumber: mobile, UserName: mobile, TenantID: tenantID})
	if err != nil {
		return "", err
	}
	return created.UUID, nil
}

func (c *Client) call(ctx context.Context, u string, body []byte) ([]User, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("egov-user %s: %s", u, resp.Status)
	}
	var out userResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.User, nil
}
