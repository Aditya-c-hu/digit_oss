package service

import (
	"context"
	"time"

	"github.com/egov/ws-calculator/internal/billing"
	"github.com/egov/ws-calculator/internal/domain"
)

// DemandService turns calculations into billing-service demands. When the
// billing client is enabled it searches/creates/updates real demands in
// billing-service; when disabled (local/offline) it synthesizes demand records
// so callers still get a usable response.
type DemandService struct {
	Calc    *CalculationService
	Billing *billing.Client
}

func NewDemandService(c *CalculationService, b *billing.Client) *DemandService {
	return &DemandService{Calc: c, Billing: b}
}

func (d *DemandService) UpdateDemands(ctx context.Context, req *domain.RequestInfoWrapper, getBill *domain.GetBillCriteria) []domain.Demand {
	out := []domain.Demand{}
	if getBill == nil {
		return out
	}
	for _, code := range getBill.ConsumerCode {
		calcs := d.Calc.Calculate(ctx, &domain.CalculationReq{
			RequestInfo: req.RequestInfo,
			CalculationCriteria: []domain.CalculationCriteria{{
				TenantID:     getBill.TenantID,
				ConnectionNo: code,
				To:           time.Now().UnixMilli(),
				WaterConnection: &domain.WaterConnection{
					ConnectionType:     "Metered",
					ConnectionCategory: "RESIDENTIAL",
				},
			}},
		})
		for _, calc := range calcs {
			demand := buildDemand(getBill.TenantID, code, calc)
			persisted, err := d.persist(ctx, req.RequestInfo, demand)
			if err != nil || len(persisted) == 0 {
				// Fall back to the locally-built demand so the endpoint still
				// responds; billing errors are not fatal to the preview.
				out = append(out, demand)
				continue
			}
			out = append(out, persisted...)
		}
	}
	return out
}

// persist routes a demand to billing-service (search→update or create) when the
// billing client is enabled, otherwise returns the local demand unchanged.
func (d *DemandService) persist(ctx context.Context, ri *domain.RequestInfo, demand domain.Demand) ([]domain.Demand, error) {
	if d.Billing == nil || !d.Billing.Enabled {
		return []domain.Demand{demand}, nil
	}
	existing, err := d.Billing.Search(ctx, ri, demand.TenantID, demand.BusinessService, demand.ConsumerCode)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		demand.ID = existing[0].ID
		return d.Billing.Update(ctx, ri, []domain.Demand{demand})
	}
	return d.Billing.Create(ctx, ri, []domain.Demand{demand})
}

func buildDemand(tenantID, consumerCode string, calc domain.Calculation) domain.Demand {
	demand := domain.Demand{
		TenantID:        tenantID,
		ConsumerCode:    consumerCode,
		BusinessService: "WS",
		TaxPeriodFrom:   time.Now().AddDate(0, -1, 0).UnixMilli(),
		TaxPeriodTo:     time.Now().UnixMilli(),
		Status:          "ACTIVE",
	}
	for _, est := range calc.TaxHeadEstimates {
		demand.DemandDetails = append(demand.DemandDetails, domain.DemandDetail{
			TaxHeadMasterCode: est.TaxHeadCode,
			TaxAmount:         est.EstimateAmount,
		})
	}
	return demand
}
