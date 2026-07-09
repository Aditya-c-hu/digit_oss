// Package rowmapper maps database rows to domain structs for the water
// connection repository. It is the explicit row-mapping layer (the Java
// BeanPropertyRowMapper / ResultSetExtractor equivalent): the postgres package
// scans rows through here so SQL execution and struct assembly stay separate.
package rowmapper

import (
	"github.com/egov/ws-services/internal/domain"
	"github.com/jackc/pgx/v5"
)

// SearchRow is the flat projection of one joined search row (connection +
// service + one document + one plumber). A connection can yield several rows
// (one per joined child), which ToWaterConnection / AppendJoinedChildren fold
// back into a single graph.
type SearchRow struct {
	id, tenantID, propID, appNo, appStatus, status string
	connNo, oldConnNo, action, roadType            string
	category, connType, waterSource, meterID       string
	roadCuttingArea, pipeSize, proposedPipeSize    *float64
	noOfTaps, proposedTaps                         *int
	meterInstallDate, connExecDate                 *int64
	createdBy, lastModBy                           string
	createdTime, lastModTime                       int64
	docID, docType, fileStoreID                    *string
	plumberID, plumberName, plumberLicense         *string
	locality, channel, applicationType             *string
	dateEffective                                  *int64
	disconnectReason                               *string
	isDiscTemp                                     *bool
	discExecDate                                   *int64
	oldApp                                         *bool
}

// ID returns the connection id for this row (used to group joined rows).
func (row *SearchRow) ID() string { return row.id }

// ScanSearchRow scans the current result row into a SearchRow. The column order
// must match query.searchSelect.
func ScanSearchRow(rows pgx.Rows) (*SearchRow, error) {
	row := &SearchRow{}
	err := rows.Scan(
		&row.id, &row.tenantID, &row.propID, &row.appNo, &row.appStatus, &row.status,
		&row.connNo, &row.oldConnNo, &row.action, &row.roadType,
		&row.category, &row.connType, &row.waterSource, &row.meterID,
		&row.roadCuttingArea, &row.pipeSize, &row.proposedPipeSize,
		&row.noOfTaps, &row.proposedTaps,
		&row.meterInstallDate, &row.connExecDate,
		&row.createdBy, &row.lastModBy, &row.createdTime, &row.lastModTime,
		&row.docID, &row.docType, &row.fileStoreID,
		&row.plumberID, &row.plumberName, &row.plumberLicense,
		&row.locality, &row.channel, &row.applicationType,
		&row.dateEffective, &row.disconnectReason, &row.isDiscTemp, &row.discExecDate, &row.oldApp,
	)
	return row, err
}

// ToWaterConnection builds the parent connection struct from the row.
func (row *SearchRow) ToWaterConnection() *domain.WaterConnection {
	conn := &domain.WaterConnection{
		Connection: domain.Connection{
			ID:                 row.id,
			TenantID:           row.tenantID,
			PropertyID:         row.propID,
			ApplicationNo:      row.appNo,
			ApplicationStatus:  row.appStatus,
			Status:             row.status,
			ConnectionNo:       row.connNo,
			OldConnectionNo:    row.oldConnNo,
			Action:             row.action,
			RoadType:           row.roadType,
			ConnectionCategory: row.category,
			ConnectionType:     row.connType,
			AuditDetails: &domain.AuditDetails{
				CreatedBy:        row.createdBy,
				LastModifiedBy:   row.lastModBy,
				CreatedTime:      row.createdTime,
				LastModifiedTime: row.lastModTime,
			},
		},
		WaterSource: row.waterSource,
		MeterID:     row.meterID,
	}
	row.applyOptionalFields(conn)
	return conn
}

func (row *SearchRow) applyOptionalFields(conn *domain.WaterConnection) {
	assignFloat(row.roadCuttingArea, &conn.RoadCuttingArea)
	assignFloat(row.pipeSize, &conn.PipeSize)
	assignFloat(row.proposedPipeSize, &conn.ProposedPipeSize)
	assignInt(row.noOfTaps, &conn.NoOfTaps)
	assignInt(row.proposedTaps, &conn.ProposedTaps)
	assignInt64(row.meterInstallDate, &conn.MeterInstallationDate)
	assignInt64(row.connExecDate, &conn.ConnectionExecutionDate)
	assignString(row.locality, &conn.Locality)
	assignString(row.channel, &conn.Channel)
	assignString(row.applicationType, &conn.ApplicationType)
	assignInt64(row.dateEffective, &conn.DateEffectiveFrom)
	assignString(row.disconnectReason, &conn.DisconnectionReason)
	assignBool(row.isDiscTemp, &conn.IsDisconnectionTemporary)
	assignInt64(row.discExecDate, &conn.DisconnectionExecutionDate)
	assignBool(row.oldApp, &conn.OldApplication)
}

// AppendJoinedChildren appends the document / plumber carried on this joined
// row to the already-built connection, de-duplicating empty join rows.
func (row *SearchRow) AppendJoinedChildren(conn *domain.WaterConnection) {
	if row.docID != nil && *row.docID != "" {
		conn.Documents = append(conn.Documents, domain.Document{
			ID: *row.docID, DocumentType: derefStr(row.docType), FileStoreID: derefStr(row.fileStoreID),
		})
	}
	if row.plumberID != nil && *row.plumberID != "" {
		conn.PlumberInfo = append(conn.PlumberInfo, domain.PlumberInfo{
			ID: *row.plumberID, Name: derefStr(row.plumberName), LicenseNo: derefStr(row.plumberLicense),
		})
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func assignString(src, dest *string) {
	if src != nil {
		*dest = *src
	}
}

func assignFloat(src, dest *float64) {
	if src != nil {
		*dest = *src
	}
}

func assignInt(src, dest *int) {
	if src != nil {
		*dest = *src
	}
}

func assignInt64(src, dest *int64) {
	if src != nil {
		*dest = *src
	}
}

func assignBool(src, dest *bool) {
	if src != nil {
		*dest = *src
	}
}
