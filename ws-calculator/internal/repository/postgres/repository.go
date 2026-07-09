// Package postgres is the PostgreSQL persistence layer for the calculator
// (meter readings). It owns statement execution, delegating SQL construction to
// the query package and row mapping to the rowmapper package. No HTTP logic.
package postgres

import (
	"context"
	"time"

	"github.com/egov/ws-calculator/internal/domain"
	"github.com/egov/ws-calculator/internal/repository/query"
	"github.com/egov/ws-calculator/internal/repository/rowmapper"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CalculatorRepository struct {
	Pool *pgxpool.Pool
}

func New(p *pgxpool.Pool) *CalculatorRepository { return &CalculatorRepository{Pool: p} }

// SaveMeterReading inserts a meter reading row. The Java service publishes the
// payload to a Kafka topic and a consumer thread does the insert; we keep the
// insert synchronous and let the caller decide whether to publish.
func (r *CalculatorRepository) SaveMeterReading(ctx context.Context, m *domain.MeterReading) error {
	if m.AuditDetails == nil {
		now := time.Now().UnixMilli()
		m.AuditDetails = &domain.AuditDetails{CreatedTime: now, LastModifiedTime: now}
	}
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO eg_ws_meterreading
		(id, connectionno, billingperiod, meterstatus, lastreading, lastreadingdate,
		 currentreading, currentreadingdate, consumption, createdby, lastmodifiedby,
		 createdtime, lastmodifiedtime, tenantid)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		m.ID, m.ConnectionNo, m.BillingPeriod, m.MeterStatus,
		m.LastReading, m.LastReadingDate, m.CurrentReading, m.CurrentReadingDate, m.Consumption,
		m.AuditDetails.CreatedBy, m.AuditDetails.LastModifiedBy, m.AuditDetails.CreatedTime, m.AuditDetails.LastModifiedTime,
		m.TenantID,
	)
	return err
}

// SearchMeterReadings returns meter reading rows matching the criteria.
func (r *CalculatorRepository) SearchMeterReadings(ctx context.Context, c *domain.MeterReadingSearchCriteria) ([]domain.MeterReading, error) {
	q, args := query.BuildMeterSearch(c)
	rows, err := r.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.MeterReading
	for rows.Next() {
		m, err := rowmapper.ScanMeterReading(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// LatestReading returns the most recent reading for a given connection. Used by
// the estimator to derive consumption when consumption is not pre-supplied.
func (r *CalculatorRepository) LatestReading(ctx context.Context, tenantID, connectionNo string) (*domain.MeterReading, error) {
	row := r.Pool.QueryRow(ctx, `
		SELECT id, connectionno, billingperiod, meterstatus, lastreading, lastreadingdate,
		       currentreading, currentreadingdate, consumption, tenantid
		FROM eg_ws_meterreading
		WHERE tenantid = $1 AND connectionno = $2
		ORDER BY currentreadingdate DESC LIMIT 1`,
		tenantID, connectionNo)
	var m domain.MeterReading
	if err := row.Scan(&m.ID, &m.ConnectionNo, &m.BillingPeriod, &m.MeterStatus,
		&m.LastReading, &m.LastReadingDate, &m.CurrentReading, &m.CurrentReadingDate, &m.Consumption,
		&m.TenantID); err != nil {
		return nil, err
	}
	return &m, nil
}
