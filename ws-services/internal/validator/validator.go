package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/egov/ws-services/internal/domain"
	"github.com/egov/ws-services/internal/mdms"
	"github.com/egov/ws-services/pkg/apperr"
)

// Validator enforces create/update preconditions. Field-level checks always
// run; master-data checks run only when the MDMS client is enabled, so local
// deployments without MDMS still function.
type Validator struct {
	MDMS *mdms.Client
}

func New(m *mdms.Client) *Validator {
	return &Validator{MDMS: m}
}

const wsMastersModule = "ws-services-masters"

// masterEntry is the common {code, active} shape of MDMS master rows.
type masterEntry struct {
	Code   string `json:"code"`
	Active *bool  `json:"active"`
}

// ValidateCreate checks required fields and, when MDMS is enabled, that the
// connection's coded attributes exist in the tenant master data.
func (v *Validator) ValidateCreate(ctx context.Context, req *domain.WaterConnectionRequest) error {
	if req == nil || req.WaterConnection == nil {
		return apperr.BadRequest("EG_WS_PAYLOAD_REQUIRED", "WaterConnection payload is required")
	}
	wc := req.WaterConnection
	if wc.TenantID == "" {
		return apperr.BadRequest("EG_WS_TENANTID_REQUIRED", "tenantId is required")
	}
	if wc.PropertyID == "" {
		return apperr.BadRequest("EG_WS_PROPERTYID_REQUIRED", "propertyId is required")
	}

	if v.MDMS == nil || !v.MDMS.Enabled {
		return nil
	}
	return v.validateAgainstMasters(ctx, req)
}

func (v *Validator) validateAgainstMasters(ctx context.Context, req *domain.WaterConnectionRequest) error {
	wc := req.WaterConnection
	res, err := v.MDMS.Search(ctx, req.RequestInfo, wc.TenantID, []mdms.ModuleDetail{{
		ModuleName: wsMastersModule,
		MasterDetails: []mdms.MasterDetail{
			{Name: "connectionType"},
			{Name: "waterSource"},
			{Name: "roadType"},
		},
	}})
	if err != nil {
		return apperr.Internal("EG_WS_MDMS_ERROR", "mdms validation failed: "+err.Error())
	}
	module := res[wsMastersModule]

	if err := checkCode(module, "connectionType", wc.ConnectionType); err != nil {
		return err
	}
	if err := checkCode(module, "waterSource", wc.WaterSource); err != nil {
		return err
	}
	if err := checkCode(module, "roadType", wc.RoadType); err != nil {
		return err
	}
	return nil
}

// checkCode validates value against the active codes of an MDMS master. It is a
// no-op when the value is empty or the master is absent/empty (lenient, matches
// the Java behaviour of only rejecting values present-but-invalid).
func checkCode(module map[string][]json.RawMessage, master, value string) error {
	if value == "" {
		return nil
	}
	rows, ok := module[master]
	if !ok || len(rows) == 0 {
		return nil
	}
	allowed := make([]string, 0, len(rows))
	for _, raw := range rows {
		var e masterEntry
		if err := json.Unmarshal(raw, &e); err != nil || e.Code == "" {
			continue
		}
		if e.Active != nil && !*e.Active {
			continue
		}
		if e.Code == value {
			return nil
		}
		allowed = append(allowed, e.Code)
	}
	return apperr.BadRequest(
		"EG_WS_INVALID_"+strings.ToUpper(master),
		fmt.Sprintf("invalid %s %q: allowed [%s]", master, value, strings.Join(allowed, ", ")),
	)
}
