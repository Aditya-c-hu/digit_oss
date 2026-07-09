package service

import (
	"context"
	"time"

	"github.com/egov/ws-calculator/internal/domain"
	"github.com/egov/ws-calculator/internal/repository/postgres"
	"github.com/egov/ws-calculator/pkg/apperr"
	"github.com/google/uuid"
)

// MeterService is the equivalent of MeterServicesImpl in Java. It handles
// create + search of meter readings, deriving consumption when not provided.
type MeterService struct {
	Repo *postgres.CalculatorRepository
}

func NewMeterService(r *postgres.CalculatorRepository) *MeterService {
	return &MeterService{Repo: r}
}

func (s *MeterService) Create(ctx context.Context, req *domain.MeterConnectionRequest) (*domain.MeterReading, error) {
	if req == nil || req.MeterReading == nil {
		return nil, apperr.BadRequest("EG_WS_MR_PAYLOAD_REQUIRED", "MeterReading payload is required")
	}
	m := req.MeterReading
	if m.TenantID == "" || m.ConnectionNo == "" {
		return nil, apperr.BadRequest("EG_WS_MR_FIELDS_REQUIRED", "tenantId and connectionNo are required")
	}
	m.ID = uuid.NewString()
	if m.Consumption == 0 {
		m.Consumption = m.CurrentReading - m.LastReading
	}
	if req.RequestInfo != nil && req.RequestInfo.UserInfo != nil {
		now := time.Now().UnixMilli()
		m.AuditDetails = &domain.AuditDetails{
			CreatedBy:        req.RequestInfo.UserInfo.UUID,
			LastModifiedBy:   req.RequestInfo.UserInfo.UUID,
			CreatedTime:      now,
			LastModifiedTime: now,
		}
	}
	if err := s.Repo.SaveMeterReading(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MeterService) Search(ctx context.Context, c *domain.MeterReadingSearchCriteria) ([]domain.MeterReading, error) {
	if c.Limit == 0 {
		c.Limit = 50
	}
	return s.Repo.SearchMeterReadings(ctx, c)
}
