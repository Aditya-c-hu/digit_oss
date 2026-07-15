// Package postgres is the PostgreSQL persistence layer for water connections.
// It owns transactions and statement execution, delegating SQL construction to
// the query package and row-to-struct assembly to the rowmapper package. It
// contains no HTTP or controller logic.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/egov/ws-services/internal/domain"
	"github.com/egov/ws-services/internal/repository/query"
	"github.com/egov/ws-services/internal/repository/rowmapper"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WaterRepository struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *WaterRepository {
	return &WaterRepository{Pool: pool}
}

// Save persists a brand-new water connection inside a single transaction
// covering the parent connection row, the water-specific row, and child
// document/plumber/holder records.
func (r *WaterRepository) Save(ctx context.Context, wc *domain.WaterConnection) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UnixMilli()
	if wc.AuditDetails == nil {
		wc.AuditDetails = &domain.AuditDetails{CreatedTime: now, LastModifiedTime: now}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO eg_ws_connection
		(id, tenantid, property_id, applicationno, applicationstatus, status, connectionno,
		 oldconnectionno, roadcuttingarea, action, roadtype,
		 createdby, lastmodifiedby, createdtime, lastmodifiedtime,
		 applicationType, channel, dateEffectiveFrom, isoldapplication, locality,
		 disconnectionreason, isDisconnectionTemporary)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
		ON CONFLICT (id) DO NOTHING`,
		wc.ID, wc.TenantID, wc.PropertyID, wc.ApplicationNo, wc.ApplicationStatus, statusOrDefault(wc.Status),
		wc.ConnectionNo, wc.OldConnectionNo, wc.RoadCuttingArea, wc.Action, wc.RoadType,
		wc.AuditDetails.CreatedBy, wc.AuditDetails.LastModifiedBy, wc.AuditDetails.CreatedTime, wc.AuditDetails.LastModifiedTime,
		wc.ApplicationType, wc.Channel, wc.DateEffectiveFrom, wc.OldApplication, wc.Locality,
		wc.DisconnectionReason, wc.IsDisconnectionTemporary,
	)
	if err != nil {
		return fmt.Errorf("insert eg_ws_connection: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO eg_ws_service
		(connection_id, connectioncategory, connectiontype, watersource, meterid, meterinstallationdate,
		 pipesize, nooftaps, connectionexecutiondate, proposedpipesize, proposedtaps, appCreatedDate,
		 disconnectionExecutionDate)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		wc.ID, wc.ConnectionCategory, wc.ConnectionType, wc.WaterSource, wc.MeterID, wc.MeterInstallationDate,
		wc.PipeSize, wc.NoOfTaps, wc.ConnectionExecutionDate, wc.ProposedPipeSize, wc.ProposedTaps, now,
		wc.DisconnectionExecutionDate,
	)
	if err != nil {
		return fmt.Errorf("insert eg_ws_service: %w", err)
	}

	if err := r.saveChildren(ctx, tx, wc); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Update writes audited changes to a connection. It does not handle workflow transitions; that
// happens upstream in the service layer before this is called.
func (r *WaterRepository) Update(ctx context.Context, wc *domain.WaterConnection) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UnixMilli()
	if wc.AuditDetails == nil {
		wc.AuditDetails = &domain.AuditDetails{LastModifiedTime: now}
	} else {
		wc.AuditDetails.LastModifiedTime = now
	}

	_, err = tx.Exec(ctx, `
		UPDATE eg_ws_connection SET
			applicationstatus=$1, status=$2, connectionno=$3, oldconnectionno=$4,
			roadcuttingarea=$5, action=$6, roadtype=$7, lastmodifiedby=$8, lastmodifiedtime=$9,
			applicationType=$10, channel=$11, dateEffectiveFrom=$12,
			disconnectionreason=$13, isDisconnectionTemporary=$14, locality=$15
		WHERE id=$16`,
		wc.ApplicationStatus, statusOrDefault(wc.Status), wc.ConnectionNo, wc.OldConnectionNo,
		wc.RoadCuttingArea, wc.Action, wc.RoadType, wc.AuditDetails.LastModifiedBy, wc.AuditDetails.LastModifiedTime,
		wc.ApplicationType, wc.Channel, wc.DateEffectiveFrom, wc.DisconnectionReason, wc.IsDisconnectionTemporary,
		wc.Locality, wc.ID,
	)
	if err != nil {
		return fmt.Errorf("update eg_ws_connection: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE eg_ws_service SET
			connectioncategory=$1, connectiontype=$2, watersource=$3, meterid=$4, meterinstallationdate=$5,
			pipesize=$6, nooftaps=$7, connectionexecutiondate=$8, proposedpipesize=$9, proposedtaps=$10,
			disconnectionExecutionDate=$11
		WHERE connection_id=$12`,
		wc.ConnectionCategory, wc.ConnectionType, wc.WaterSource, wc.MeterID, wc.MeterInstallationDate,
		wc.PipeSize, wc.NoOfTaps, wc.ConnectionExecutionDate, wc.ProposedPipeSize, wc.ProposedTaps,
		wc.DisconnectionExecutionDate, wc.ID,
	)
	if err != nil {
		return fmt.Errorf("update eg_ws_service: %w", err)
	}

	// Replace child rows so the update reflects the submitted graph. Mirrors the
	// Java repository, which deletes and re-inserts the connection's documents,
	// plumbers, holders and road-cutting entries on every update.
	if wc.AuditDetails.CreatedTime == 0 {
		wc.AuditDetails.CreatedTime = wc.AuditDetails.LastModifiedTime
		wc.AuditDetails.CreatedBy = wc.AuditDetails.LastModifiedBy
	}
	for _, child := range []struct{ table, col string }{
		{"eg_ws_applicationdocument", "wsid"},
		{"eg_ws_plumberinfo", "wsid"},
		{"eg_ws_connectionholder", "connectionid"},
		{"eg_ws_roadcuttinginfo", "wsid"},
	} {
		if _, err := tx.Exec(ctx, "DELETE FROM "+child.table+" WHERE "+child.col+"=$1", wc.ID); err != nil {
			return fmt.Errorf("delete %s: %w", child.table, err)
		}
	}
	if err := r.saveChildren(ctx, tx, wc); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *WaterRepository) saveChildren(ctx context.Context, tx pgx.Tx, wc *domain.WaterConnection) error {
	if err := r.saveDocuments(ctx, tx, wc); err != nil {
		return err
	}
	if err := r.savePlumbers(ctx, tx, wc); err != nil {
		return err
	}
	if err := r.saveConnectionHolders(ctx, tx, wc); err != nil {
		return err
	}
	return r.saveRoadCuttingInfo(ctx, tx, wc)
}

// Search returns the connections matching the given criteria, joined with their child rows.
func (r *WaterRepository) Search(ctx context.Context, c *domain.SearchCriteria) ([]domain.WaterConnection, error) {
	q, args := query.BuildSearch(c)
	rows, err := r.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conns := map[string]*domain.WaterConnection{}
	order := []string{}

	for rows.Next() {
		row, err := rowmapper.ScanSearchRow(rows)
		if err != nil {
			return nil, err
		}
		conn, ok := conns[row.ID()]
		if !ok {
			conn = row.ToWaterConnection()
			conns[row.ID()] = conn
			order = append(order, row.ID())
		}
		row.AppendJoinedChildren(conn)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// Load the remaining child collections (holders + road-cutting) in batched
	// follow-up queries and attach them, so search returns the full graph like
	// the Java service (documents + plumbers come from the joined query above).
	holders, err := r.loadConnectionHolders(ctx, order)
	if err != nil {
		return nil, err
	}
	roadCutting, err := r.loadRoadCuttingInfo(ctx, order)
	if err != nil {
		return nil, err
	}
	for _, id := range order {
		conns[id].ConnectionHolders = holders[id]
		conns[id].RoadCuttingInfo = roadCutting[id]
	}

	out := make([]domain.WaterConnection, 0, len(order))
	for _, id := range order {
		out = append(out, *conns[id])
	}
	return out, nil
}

// loadConnectionHolders returns holders keyed by connection id for the given ids.
func (r *WaterRepository) loadConnectionHolders(ctx context.Context, ids []string) (map[string][]domain.OwnerInfo, error) {
	out := map[string][]domain.OwnerInfo{}
	if len(ids) == 0 {
		return out, nil
	}
	// holdershippercentage is character varying(128) in the schema (Java reads it
	// with rs.getDouble, which coerces the text to a double). pgx is stricter, so
	// the column is COALESCEd to '' and scanned as text, then parsed to float64 —
	// COALESCE(...,0) here would be a varchar/integer type error in Postgres.
	rows, err := r.Pool.Query(ctx, `
		SELECT connectionid, COALESCE(userid,''), COALESCE(status,''), COALESCE(isprimaryholder,false),
		       COALESCE(connectionholdertype,''), COALESCE(holdershippercentage,''), COALESCE(relationship,'')
		FROM eg_ws_connectionholder WHERE connectionid = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var connID, holderPct string
		var o domain.OwnerInfo
		if err := rows.Scan(&connID, &o.UUID, &o.Status, &o.IsPrimaryHolder,
			&o.ConnectionHolderType, &holderPct, &o.Relationship); err != nil {
			return nil, err
		}
		o.HolderShipPercentage = parseHolderPercent(holderPct)
		out[connID] = append(out[connID], o)
	}
	return out, rows.Err()
}

// loadRoadCuttingInfo returns road-cutting rows keyed by connection id (wsid).
func (r *WaterRepository) loadRoadCuttingInfo(ctx context.Context, ids []string) (map[string][]domain.RoadCuttingInfo, error) {
	out := map[string][]domain.RoadCuttingInfo{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.Pool.Query(ctx, `
		SELECT wsid, COALESCE(id,''), COALESCE(roadtype,''), COALESCE(roadcuttingarea,0), COALESCE(active,'')
		FROM eg_ws_roadcuttinginfo WHERE wsid = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var wsid, active string
		var rc domain.RoadCuttingInfo
		if err := rows.Scan(&wsid, &rc.ID, &rc.RoadType, &rc.RoadCuttingArea, &active); err != nil {
			return nil, err
		}
		rc.Active = active == "ACTIVE"
		out[wsid] = append(out[wsid], rc)
	}
	return out, rows.Err()
}

// Count returns total connections matching the criteria. Used by the search endpoint
// to populate `totalCount` for pagination UIs.
func (r *WaterRepository) Count(ctx context.Context, c *domain.SearchCriteria) (int, error) {
	q, args := query.BuildCount(c)
	var n int
	if err := r.Pool.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *WaterRepository) GetByApplicationNo(ctx context.Context, tenantID, appNo string) (*domain.WaterConnection, error) {
	c := &domain.SearchCriteria{TenantID: tenantID, ApplicationNumber: []string{appNo}, Limit: 1}
	res, err := r.Search(ctx, c)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errors.New("not found")
	}
	return &res[0], nil
}

func (r *WaterRepository) saveDocuments(ctx context.Context, tx pgx.Tx, wc *domain.WaterConnection) error {
	for _, d := range wc.Documents {
		id := d.ID
		if id == "" {
			id = uuid.NewString()
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO eg_ws_applicationdocument (id, tenantid, documenttype, filestoreid, wsid, active, documentUid,
				createdby, lastmodifiedby, createdtime, lastmodifiedtime)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			id, wc.TenantID, d.DocumentType, d.FileStoreID, wc.ID, "ACTIVE", d.DocumentUID,
			wc.AuditDetails.CreatedBy, wc.AuditDetails.LastModifiedBy, wc.AuditDetails.CreatedTime, wc.AuditDetails.LastModifiedTime,
		)
		if err != nil {
			return fmt.Errorf("insert document: %w", err)
		}
	}
	return nil
}

func (r *WaterRepository) savePlumbers(ctx context.Context, tx pgx.Tx, wc *domain.WaterConnection) error {
	for _, p := range wc.PlumberInfo {
		id := p.ID
		if id == "" {
			id = uuid.NewString()
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO eg_ws_plumberinfo (id, name, licenseno, mobilenumber, gender, fatherorhusbandname,
				correspondenceaddress, relationship, wsid)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			id, p.Name, p.LicenseNo, p.MobileNumber, p.Gender, p.FatherOrHusbandName,
			p.CorrespondenceAddress, p.Relationship, wc.ID,
		)
		if err != nil {
			return fmt.Errorf("insert plumber: %w", err)
		}
	}
	return nil
}

func (r *WaterRepository) saveConnectionHolders(ctx context.Context, tx pgx.Tx, wc *domain.WaterConnection) error {
	for _, h := range wc.ConnectionHolders {
		_, err := tx.Exec(ctx, `
			INSERT INTO eg_ws_connectionholder (tenantid, connectionid, userid, status, isprimaryholder,
				connectionholdertype, holdershippercentage, relationship, createdby, createdtime,
				lastmodifiedby, lastmodifiedtime)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT DO NOTHING`,
			wc.TenantID, wc.ID, h.UUID, h.Status, h.IsPrimaryHolder,
			h.ConnectionHolderType, holderPercentText(h.HolderShipPercentage), h.Relationship,
			wc.AuditDetails.CreatedBy, wc.AuditDetails.CreatedTime,
			wc.AuditDetails.LastModifiedBy, wc.AuditDetails.LastModifiedTime,
		)
		if err != nil {
			return fmt.Errorf("insert holder: %w", err)
		}
	}
	return nil
}

func (r *WaterRepository) saveRoadCuttingInfo(ctx context.Context, tx pgx.Tx, wc *domain.WaterConnection) error {
	for _, rc := range wc.RoadCuttingInfo {
		id := rc.ID
		if id == "" {
			id = uuid.NewString()
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO eg_ws_roadcuttinginfo (id, wsid, roadtype, roadcuttingarea, active)
			VALUES ($1,$2,$3,$4,$5)`,
			id, wc.ID, rc.RoadType, rc.RoadCuttingArea, ifTrue(rc.Active, "ACTIVE", "INACTIVE"),
		)
		if err != nil {
			return fmt.Errorf("insert roadcuttinginfo: %w", err)
		}
	}
	return nil
}

// holderPercentText renders the holder-ship percentage for the varchar(128)
// column. Zero is stored as empty (the column is nullable text in the schema),
// otherwise the shortest exact decimal form.
func holderPercentText(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// parseHolderPercent parses the text holder-ship percentage back to a float,
// mirroring the Java rowmapper's rs.getDouble (blank/non-numeric -> 0).
func parseHolderPercent(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func statusOrDefault(s string) string {
	if s == "" {
		return "Active"
	}
	return s
}

func ifTrue(b bool, a, c string) string {
	if b {
		return a
	}
	return c
}
