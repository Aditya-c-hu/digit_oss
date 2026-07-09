// Package query holds the SQL query builders for the water-connection
// repository. It is deliberately free of any database driver dependency: each
// builder returns a parameterized SQL string plus its ordered argument slice,
// which the postgres package executes. This mirrors the query-construction
// concern that lived in the Java repository's prepared-statement builders.
package query

import (
	"fmt"
	"strings"

	"github.com/egov/ws-services/internal/domain"
)

// columns selected by the search query, joined with documents + plumbers.
const searchSelect = `SELECT
	conn.id, conn.tenantid, conn.property_id, conn.applicationno, conn.applicationstatus, conn.status,
	conn.connectionno, conn.oldconnectionno, COALESCE(conn.action,''), COALESCE(conn.roadtype,''),
	COALESCE(wc.connectioncategory,''), COALESCE(wc.connectiontype,''), COALESCE(wc.watersource,''),
	COALESCE(wc.meterid,''),
	conn.roadcuttingarea, wc.pipesize, wc.proposedpipesize,
	wc.nooftaps, wc.proposedtaps,
	wc.meterinstallationdate, wc.connectionexecutiondate,
	COALESCE(conn.createdby,''), COALESCE(conn.lastmodifiedby,''),
	COALESCE(conn.createdtime,0), COALESCE(conn.lastmodifiedtime,0),
	document.id, document.documenttype, document.filestoreid,
	plumber.id, plumber.name, plumber.licenseno,
	conn.locality, conn.channel, conn.applicationType,
	conn.dateEffectiveFrom, conn.disconnectionreason, conn.isDisconnectionTemporary,
	wc.disconnectionExecutionDate, conn.isoldapplication`

const searchFrom = `
	FROM eg_ws_connection conn
	INNER JOIN eg_ws_service wc ON wc.connection_id = conn.id
	LEFT OUTER JOIN eg_ws_applicationdocument document ON document.wsid = conn.id
	LEFT OUTER JOIN eg_ws_plumberinfo plumber ON plumber.wsid = conn.id`

// BuildSearch returns the paged search SQL and its args for the given criteria.
func BuildSearch(c *domain.SearchCriteria) (string, []any) {
	b := newBuilder()
	b.addSearchFilters(c)
	where := b.whereClause()
	offsetArg := b.addArg(searchOffset(c))
	limitArg := b.addArg(searchLimit(c))
	q := searchSelect + searchFrom + where +
		" ORDER BY wc.appCreatedDate DESC " +
		fmt.Sprintf("OFFSET %s LIMIT %s", offsetArg, limitArg)
	return q, b.args
}

// BuildCount returns the COUNT(DISTINCT) SQL and its args for the criteria.
func BuildCount(c *domain.SearchCriteria) (string, []any) {
	b := newBuilder()
	b.addSearchFilters(c)
	return "SELECT COUNT(DISTINCT conn.id) " + searchFrom + b.whereClause(), b.args
}

type builder struct {
	args  []any
	where []string
}

func newBuilder() *builder { return &builder{args: []any{}, where: []string{}} }

func (b *builder) addArg(v any) string {
	b.args = append(b.args, v)
	return fmt.Sprintf("$%d", len(b.args))
}

func (b *builder) addStringFilter(column, value string) {
	if value != "" {
		b.where = append(b.where, column+" = "+b.addArg(value))
	}
}

func (b *builder) addArrayFilter(column string, values []string) {
	if len(values) > 0 {
		b.where = append(b.where, column+" = ANY("+b.addArg(values)+")")
	}
}

func (b *builder) addInt64LowerBound(column string, value int64) {
	if value > 0 {
		b.where = append(b.where, column+" >= "+b.addArg(value))
	}
}

func (b *builder) addInt64UpperBound(column string, value int64) {
	if value > 0 {
		b.where = append(b.where, column+" <= "+b.addArg(value))
	}
}

func (b *builder) addSearchFilters(c *domain.SearchCriteria) {
	b.addStringFilter("conn.tenantid", c.TenantID)
	b.addArrayFilter("conn.id", c.IDs)
	b.addArrayFilter("conn.applicationno", c.ApplicationNumber)
	b.addArrayFilter("conn.applicationstatus", c.ApplicationStatus)
	b.addArrayFilter("conn.connectionno", c.ConnectionNumber)
	b.addStringFilter("conn.oldconnectionno", c.OldConnectionNumber)
	b.addStringFilter("conn.property_id", c.PropertyID)
	b.addStringFilter("conn.status", c.Status)
	b.addInt64LowerBound("wc.appCreatedDate", c.FromDate)
	b.addInt64UpperBound("wc.appCreatedDate", c.ToDate)
	b.addStringFilter("conn.applicationType", c.ApplicationType)
	b.addStringFilter("conn.locality", c.Locality)
}

func (b *builder) whereClause() string {
	if len(b.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(b.where, " AND ")
}

func searchLimit(c *domain.SearchCriteria) int {
	if c.Limit <= 0 {
		return 50
	}
	return c.Limit
}

func searchOffset(c *domain.SearchCriteria) int {
	if c.Offset < 0 {
		return 0
	}
	return c.Offset
}
