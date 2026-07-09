package mdms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/egov/ws-calculator/internal/domain"
)

// Client integrates with egov-mdms-service to load billing slabs and other
// calculator master data. When disabled, the calculator keeps its seeded slabs.
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

type masterDetail struct {
	Name string `json:"name"`
}

type moduleDetail struct {
	ModuleName    string         `json:"moduleName"`
	MasterDetails []masterDetail `json:"masterDetails"`
}

type criteria struct {
	TenantID      string         `json:"tenantId"`
	ModuleDetails []moduleDetail `json:"moduleDetails"`
}

type searchRequest struct {
	RequestInfo  *domain.RequestInfo `json:"RequestInfo"`
	MdmsCriteria criteria            `json:"MdmsCriteria"`
}

type searchResponse struct {
	MdmsRes map[string]map[string][]json.RawMessage `json:"MdmsRes"`
}

// LoadMaster fetches the raw rows of a single MDMS master. Empty result returns
// (nil, nil) so callers can keep their existing values.
func (c *Client) LoadMaster(ctx context.Context, ri *domain.RequestInfo, tenantID, module, master string) ([]json.RawMessage, error) {
	body, err := json.Marshal(searchRequest{
		RequestInfo: ri,
		MdmsCriteria: criteria{
			TenantID:      tenantID,
			ModuleDetails: []moduleDetail{{ModuleName: module, MasterDetails: []masterDetail{{Name: master}}}},
		},
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
		return nil, fmt.Errorf("mdms search %s.%s: %s", module, master, resp.Status)
	}
	var out searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	rows := out.MdmsRes[module][master]
	if len(rows) == 0 {
		return nil, nil
	}
	return rows, nil
}

// LoadBillingSlabs fetches the WCBillingSlab master and unmarshals it into the
// calculator's BillingSlab model.
func (c *Client) LoadBillingSlabs(ctx context.Context, ri *domain.RequestInfo, tenantID, module, master string) ([]domain.BillingSlab, error) {
	rows, err := c.LoadMaster(ctx, ri, tenantID, module, master)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	slabs := make([]domain.BillingSlab, 0, len(rows))
	for _, raw := range rows {
		var s domain.BillingSlab
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		slabs = append(slabs, s)
	}
	return slabs, nil
}
