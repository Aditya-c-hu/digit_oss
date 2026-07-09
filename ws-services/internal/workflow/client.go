package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/egov/ws-services/internal/domain"
)

// Client integrates with the egov-workflow-v2 service for application transitions.
// When external workflow is disabled (offline / dev), Transition is a no-op that
// just stamps the action onto the connection state.
type Client struct {
	HTTP               *http.Client
	Host               string
	TransitionPath     string
	BusinessSearchPath string
	Enabled            bool
}

func New(host, transitionPath, businessSearchPath string, enabled bool) *Client {
	return &Client{
		HTTP:               &http.Client{Timeout: 10 * time.Second},
		Host:               host,
		TransitionPath:     transitionPath,
		BusinessSearchPath: businessSearchPath,
		Enabled:            enabled,
	}
}

type TransitionRequest struct {
	RequestInfo      *domain.RequestInfo      `json:"RequestInfo"`
	ProcessInstances []domain.ProcessInstance `json:"ProcessInstances"`
}

type TransitionResponse struct {
	ResponseInfo     *domain.ResponseInfo     `json:"ResponseInfo"`
	ProcessInstances []domain.ProcessInstance `json:"ProcessInstances"`
}

// Transition kicks off / advances the workflow for a connection. The Java code
// does the same call and reads back the resolved applicationStatus from the
// returned ProcessInstance to stamp on the entity.
func (c *Client) Transition(ctx context.Context, req TransitionRequest) (*TransitionResponse, error) {
	if !c.Enabled {
		// Local mode: synthesize a state matching the action so the caller
		// sees a deterministic applicationStatus without an external dep.
		out := &TransitionResponse{}
		for _, pi := range req.ProcessInstances {
			pi.State = &domain.State{
				ApplicationStatus: actionToStatus(pi.Action),
				State:             actionToStatus(pi.Action),
			}
			out.ProcessInstances = append(out.ProcessInstances, pi)
		}
		return out, nil
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Host+c.TransitionPath, bytes.NewReader(body))
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
		return nil, errors.New("workflow transition failed: " + resp.Status)
	}
	var out TransitionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// actionToStatus is a fallback mapping when running without external workflow.
// Mirrors the common WS state machine in DIGIT.
func actionToStatus(action string) string {
	switch action {
	case "INITIATE":
		return "INITIATED"
	case "SUBMIT_APPLICATION":
		return "PENDING_APPROVAL_FOR_CONNECTION"
	case "APPROVE_FOR_CONNECTION":
		return "PENDING_APPROVAL_FOR_CONNECTION"
	case "PAY":
		return "PENDING_PAYMENT"
	case "ACTIVATE_CONNECTION":
		return "CONNECTION_ACTIVATED"
	case "REJECT":
		return "REJECTED"
	default:
		return action
	}
}
