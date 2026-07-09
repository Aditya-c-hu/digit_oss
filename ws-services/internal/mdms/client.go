package mdms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/egov/ws-services/internal/domain"
)

// Client integrates with egov-mdms-service for master-data lookups used by the
// connection validator. When disabled, the validator skips master checks.
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

type MasterDetail struct {
	Name   string `json:"name"`
	Filter string `json:"filter,omitempty"`
}

type ModuleDetail struct {
	ModuleName    string         `json:"moduleName"`
	MasterDetails []MasterDetail `json:"masterDetails"`
}

type criteria struct {
	TenantID      string         `json:"tenantId"`
	ModuleDetails []ModuleDetail `json:"moduleDetails"`
}

type searchRequest struct {
	RequestInfo  *domain.RequestInfo `json:"RequestInfo"`
	MdmsCriteria criteria            `json:"MdmsCriteria"`
}

type searchResponse struct {
	MdmsRes map[string]map[string][]json.RawMessage `json:"MdmsRes"`
}

// Search returns the MdmsRes map keyed by module then master name. Mirrors the
// Java MDMS _search call used during connection validation.
func (c *Client) Search(ctx context.Context, ri *domain.RequestInfo, tenantID string, modules []ModuleDetail) (map[string]map[string][]json.RawMessage, error) {
	body, err := json.Marshal(searchRequest{
		RequestInfo:  ri,
		MdmsCriteria: criteria{TenantID: tenantID, ModuleDetails: modules},
	})
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Host+c.Path, bytes.NewReader(body))
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
		return nil, fmt.Errorf("mdms search: %s", resp.Status)
	}
	var out searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.MdmsRes, nil
}
