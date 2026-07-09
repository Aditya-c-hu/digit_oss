package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/egov/ws-services/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestSaveSearchUpdateRoundtrip is an integration test. It runs only when
// TEST_DATABASE_URL points at a Postgres with the ws-services schema applied
// (see migrations/ddl/V001__ws_schema.sql); otherwise it is skipped.
//
// Example:
//
//	TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/rainmaker?sslmode=disable" go test ./internal/repository/...
func TestSaveSearchUpdateRoundtrip(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run repository integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("database unreachable: %v", err)
	}
	repo := New(pool)

	tenant := "pb.amritsar"
	appNo := "WS-TEST-" + uuid.NewString()[:8]
	now := time.Now().UnixMilli()
	wc := &domain.WaterConnection{
		Connection: domain.Connection{
			ID: uuid.NewString(), TenantID: tenant, PropertyID: "P-" + uuid.NewString()[:8],
			ApplicationNo: appNo, ApplicationStatus: "INITIATED", Status: "Active",
			ConnectionType:  "Non Metered",
			AuditDetails:    &domain.AuditDetails{CreatedBy: "t", LastModifiedBy: "t", CreatedTime: now, LastModifiedTime: now},
			RoadCuttingInfo: []domain.RoadCuttingInfo{{ID: uuid.NewString(), RoadType: "BT", RoadCuttingArea: 2, Active: true}},
		},
		WaterSource: "BOREWELL", NoOfTaps: 2,
	}

	if err := repo.Save(ctx, wc); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Search(ctx, &domain.SearchCriteria{TenantID: tenant, ApplicationNumber: []string{appNo}})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("search returned %d, want 1", len(got))
	}
	if len(got[0].RoadCuttingInfo) != 1 {
		t.Errorf("road-cutting not loaded in search: %+v", got[0].RoadCuttingInfo)
	}

	wc.ApplicationStatus = "PENDING_APPROVAL_FOR_CONNECTION"
	if err := repo.Update(ctx, wc); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = repo.Search(ctx, &domain.SearchCriteria{TenantID: tenant, ApplicationNumber: []string{appNo}})
	if len(got) == 1 && got[0].ApplicationStatus != "PENDING_APPROVAL_FOR_CONNECTION" {
		t.Errorf("status after update = %q", got[0].ApplicationStatus)
	}
}
