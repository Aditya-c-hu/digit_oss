// Package query holds the SQL query builders for the calculator repository.
// Builders return a parameterized SQL string plus its ordered argument slice;
// the postgres package executes them. No database driver dependency lives here.
package query

import (
	"fmt"
	"strings"

	"github.com/egov/ws-calculator/internal/domain"
)

// MeterReadingColumns is the SELECT projection scanned by rowmapper.ScanMeterReading.
const MeterReadingColumns = `id, connectionno, billingperiod, meterstatus, lastreading, lastreadingdate,
	currentreading, currentreadingdate, consumption, tenantid,
	COALESCE(createdby,''), COALESCE(lastmodifiedby,''), COALESCE(createdtime,0), COALESCE(lastmodifiedtime,0)`

// BuildMeterSearch returns the meter-reading search SQL and its args.
func BuildMeterSearch(c *domain.MeterReadingSearchCriteria) (string, []any) {
	args := []any{}
	idx := 0
	addArg := func(v any) string {
		idx++
		args = append(args, v)
		return fmt.Sprintf("$%d", idx)
	}
	where := []string{}
	if c.TenantID != "" {
		where = append(where, "tenantid = "+addArg(c.TenantID))
	}
	if len(c.ConnectionNos) > 0 {
		where = append(where, "connectionno = ANY("+addArg(c.ConnectionNos)+")")
	}
	if len(c.IDs) > 0 {
		where = append(where, "id = ANY("+addArg(c.IDs)+")")
	}
	q := "SELECT " + MeterReadingColumns + " FROM eg_ws_meterreading"
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY currentreadingdate DESC"
	if c.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %s", addArg(c.Limit))
	}
	if c.Offset > 0 {
		q += fmt.Sprintf(" OFFSET %s", addArg(c.Offset))
	}
	return q, args
}
