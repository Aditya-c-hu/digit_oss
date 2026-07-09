package property

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/egov/ws-services/internal/domain"
	"github.com/egov/ws-services/pkg/apperr"
)

// Client integrates with property-services to validate the propertyId a water
// connection is being created against. Disabled => validation skipped.
type Client struct {
	HTTP       *http.Client
	Host       string
	SearchPath string
	Enabled    bool
}

func New(host, searchPath string, enabled bool) *Client {
	return &Client{
		HTTP:       &http.Client{Timeout: 10 * time.Second},
		Host:       host,
		SearchPath: searchPath,
		Enabled:    enabled,
	}
}

// Property is the slice of the property-services response we care about.
type Property struct {
	ID         string `json:"id,omitempty"`
	PropertyID string `json:"propertyId,omitempty"`
	TenantID   string `json:"tenantId,omitempty"`
	Status     string `json:"status,omitempty"`
}

type searchResponse struct {
	Properties []Property `json:"Properties"`
}

// Search returns the properties matching the given propertyId in a tenant.
func (c *Client) Search(ctx context.Context, ri *domain.RequestInfo, tenantID, propertyID string) ([]Property, error) {
	q := url.Values{}
	q.Set("tenantId", tenantID)
	q.Set("propertyIds", propertyID)
	u := c.Host + c.SearchPath + "?" + q.Encode()
	body, _ := json.Marshal(map[string]any{"RequestInfo": ri})

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
		return nil, fmt.Errorf("property search: %s", resp.Status)
	}
	var out searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Properties, nil
}

// Validate confirms the propertyId resolves to a property in the tenant.
func (c *Client) Validate(ctx context.Context, ri *domain.RequestInfo, tenantID, propertyID string) error {
	props, err := c.Search(ctx, ri, tenantID, propertyID)
	if err != nil {
		return apperr.Internal("EG_WS_PROPERTY_LOOKUP_ERROR", err.Error())
	}
	for _, p := range props {
		if strings.EqualFold(p.PropertyID, propertyID) {
			return nil
		}
	}
	return apperr.BadRequest("EG_WS_PROPERTY_NOT_FOUND", fmt.Sprintf("propertyId %q not found in tenant %q", propertyID, tenantID))
}
