package idgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/egov/ws-services/internal/domain"
)

// Client integrates with egov-idgen to mint application and connection numbers.
// When disabled (local / offline), callers fall back to local synthesis.
type Client struct {
	HTTP    *http.Client
	Host    string
	Path    string
	Enabled bool
}

func New(host, path string, enabled bool) *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 10 * time.Second},
		Host:    host,
		Path:    path,
		Enabled: enabled,
	}
}

type idRequest struct {
	IDName   string `json:"idName"`
	TenantID string `json:"tenantId"`
	Format   string `json:"format,omitempty"`
	Count    int    `json:"count,omitempty"`
}

type generateRequest struct {
	RequestInfo *domain.RequestInfo `json:"RequestInfo"`
	IDRequests  []idRequest         `json:"idRequests"`
}

type idResponse struct {
	ID string `json:"id"`
}

type generateResponse struct {
	IDResponses []idResponse `json:"idResponses"`
}

// Generate returns a single id for the given idName/format. Mirrors the Java
// IdGenRepository.getId single-id path.
func (c *Client) Generate(ctx context.Context, ri *domain.RequestInfo, idName, tenantID, format string) (string, error) {
	body, err := json.Marshal(generateRequest{
		RequestInfo: ri,
		IDRequests:  []idRequest{{IDName: idName, TenantID: tenantID, Format: format, Count: 1}},
	})
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Host+c.Path, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("idgen %s: %s", idName, resp.Status)
	}
	var out generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.IDResponses) == 0 || out.IDResponses[0].ID == "" {
		return "", fmt.Errorf("idgen %s: empty response", idName)
	}
	return out.IDResponses[0].ID, nil
}
