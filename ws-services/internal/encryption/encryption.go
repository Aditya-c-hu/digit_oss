// Package encryption integrates with egov-enc-service to protect connection
// holder / plumber PII at rest, mirroring the Java EnrichmentService +
// EncryptionDecryptionUtil behaviour (encrypt on create, decrypt on search).
//
// IMPORTANT (parity caveat): in Java the exact set of encrypted attributes and
// the ABAC decryption visibility come from the MDMS DataSecurity SecurityPolicy
// for the "WnSConnection" model. That policy is not part of this repository, so
// this port encrypts the standard WS PII fields (connection-holder and plumber
// name / mobile / correspondence address) directly via egov-enc-service. Before
// enabling in production (IS_ENCRYPTION_ENABLED=true), confirm the attribute set
// and key/type match the tenant's SecurityPolicy so ciphertext round-trips with
// the Java service.
package encryption

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/egov/ws-services/internal/domain"
)

type Client struct {
	HTTP        *http.Client
	Host        string
	EncryptPath string
	DecryptPath string
	StateTenant string
	Enabled     bool
}

func New(host, encryptPath, decryptPath, stateTenant string, enabled bool) *Client {
	return &Client{
		HTTP:        &http.Client{Timeout: 10 * time.Second},
		Host:        host,
		EncryptPath: encryptPath,
		DecryptPath: decryptPath,
		StateTenant: stateTenant,
		Enabled:     enabled,
	}
}

type encryptionRequest struct {
	TenantID string `json:"tenantId"`
	Type     string `json:"type"`
	Value    string `json:"value"`
}

// encryptValues sends a batch of plaintext values to egov-enc-service and
// returns ciphertext aligned by index.
func (c *Client) encryptValues(ctx context.Context, values []string) ([]string, error) {
	reqs := make([]encryptionRequest, 0, len(values))
	for _, v := range values {
		reqs = append(reqs, encryptionRequest{TenantID: c.StateTenant, Type: "Normal", Value: v})
	}
	body, _ := json.Marshal(map[string]any{"encryptionRequests": reqs})
	var out []string
	if err := c.post(ctx, c.Host+c.EncryptPath, body, &out); err != nil {
		return nil, err
	}
	if len(out) != len(values) {
		return nil, fmt.Errorf("enc-service returned %d values for %d inputs", len(out), len(values))
	}
	return out, nil
}

// decryptValues reverses encryptValues. RequestInfo is forwarded for ABAC.
func (c *Client) decryptValues(ctx context.Context, ri *domain.RequestInfo, values []string) ([]string, error) {
	body, _ := json.Marshal(map[string]any{"RequestInfo": ri, "decryptionRequests": values})
	var out []string
	if err := c.post(ctx, c.Host+c.DecryptPath, body, &out); err != nil {
		return nil, err
	}
	if len(out) != len(values) {
		return nil, fmt.Errorf("enc-service returned %d values for %d inputs", len(out), len(values))
	}
	return out, nil
}

func (c *Client) post(ctx context.Context, url string, body []byte, out any) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("enc-service %s: %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// piiRefs returns pointers to the PII string fields of a connection in a stable
// order, so encrypt/decrypt operate on the same slots.
func piiRefs(wc *domain.WaterConnection) []*string {
	var refs []*string
	for i := range wc.ConnectionHolders {
		h := &wc.ConnectionHolders[i]
		refs = append(refs, &h.Name, &h.MobileNumber)
	}
	for i := range wc.PlumberInfo {
		p := &wc.PlumberInfo[i]
		refs = append(refs, &p.Name, &p.MobileNumber, &p.CorrespondenceAddress)
	}
	return refs
}

// EncryptConnection encrypts holder/plumber PII in place before persistence.
func (c *Client) EncryptConnection(ctx context.Context, wc *domain.WaterConnection) error {
	if !c.Enabled || wc == nil {
		return nil
	}
	return c.transform(ctx, wc, c.encryptInPlace)
}

// DecryptConnection decrypts holder/plumber PII in place after a search load.
func (c *Client) DecryptConnection(ctx context.Context, ri *domain.RequestInfo, wc *domain.WaterConnection) error {
	if !c.Enabled || wc == nil {
		return nil
	}
	return c.transform(ctx, wc, func(ctx context.Context, vals []string) ([]string, error) {
		return c.decryptValues(ctx, ri, vals)
	})
}

func (c *Client) encryptInPlace(ctx context.Context, vals []string) ([]string, error) {
	return c.encryptValues(ctx, vals)
}

// transform collects non-empty PII values, runs op, and writes results back.
func (c *Client) transform(ctx context.Context, wc *domain.WaterConnection, op func(context.Context, []string) ([]string, error)) error {
	refs := piiRefs(wc)
	idx := make([]int, 0, len(refs))
	vals := make([]string, 0, len(refs))
	for i, r := range refs {
		if *r != "" {
			idx = append(idx, i)
			vals = append(vals, *r)
		}
	}
	if len(vals) == 0 {
		return nil
	}
	res, err := op(ctx, vals)
	if err != nil {
		return err
	}
	for j, i := range idx {
		*refs[i] = res[j]
	}
	return nil
}
