// Package rowmapper maps database rows to domain structs for the calculator
// repository. It is the explicit row-mapping layer used by the postgres package
// so SQL execution and struct assembly stay separate.
package rowmapper

import (
	"github.com/egov/ws-calculator/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ScanMeterReading scans one row of query.MeterReadingColumns into a
// domain.MeterReading (with its AuditDetails).
func ScanMeterReading(rows pgx.Rows) (domain.MeterReading, error) {
	var m domain.MeterReading
	var aud domain.AuditDetails
	err := rows.Scan(&m.ID, &m.ConnectionNo, &m.BillingPeriod, &m.MeterStatus,
		&m.LastReading, &m.LastReadingDate, &m.CurrentReading, &m.CurrentReadingDate, &m.Consumption,
		&m.TenantID, &aud.CreatedBy, &aud.LastModifiedBy, &aud.CreatedTime, &aud.LastModifiedTime)
	if err != nil {
		return m, err
	}
	m.AuditDetails = &aud
	return m, nil
}
