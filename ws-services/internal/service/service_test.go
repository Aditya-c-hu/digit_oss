package service

import (
	"context"
	"errors"
	"testing"

	"github.com/egov/ws-services/config"
	"github.com/egov/ws-services/internal/domain"
	"github.com/egov/ws-services/internal/encryption"
	"github.com/egov/ws-services/internal/idgen"
	"github.com/egov/ws-services/internal/property"
	"github.com/egov/ws-services/internal/user"
	"github.com/egov/ws-services/internal/validator"
	"github.com/egov/ws-services/internal/workflow"
	"github.com/egov/ws-services/pkg/apperr"
)

// --- fakes (narrow interfaces, no DB / no Kafka) ---

type fakeRepo struct {
	saved   []*domain.WaterConnection
	updated []*domain.WaterConnection
	rows    []domain.WaterConnection
	count   int
}

func (f *fakeRepo) Save(_ context.Context, wc *domain.WaterConnection) error {
	f.saved = append(f.saved, wc)
	return nil
}
func (f *fakeRepo) Update(_ context.Context, wc *domain.WaterConnection) error {
	f.updated = append(f.updated, wc)
	return nil
}
func (f *fakeRepo) Search(_ context.Context, _ *domain.SearchCriteria) ([]domain.WaterConnection, error) {
	return f.rows, nil
}
func (f *fakeRepo) Count(_ context.Context, _ *domain.SearchCriteria) (int, error) {
	return f.count, nil
}

type fakePublisher struct{ topics []string }

func (f *fakePublisher) Push(_ context.Context, topic string, _ any) error {
	f.topics = append(f.topics, topic)
	return nil
}

// newSvc wires the service with all external integrations disabled (local mode).
func newSvc(repo Repository, pub Publisher) *Service {
	cfg := &config.Config{
		DefaultLimit: 50, MaxLimit: 500,
		OnWaterSavedTopic:   "save-ws-connection",
		OnWaterUpdatedTopic: "update-ws-connection",
	}
	return New(Dependencies{
		Repo:      repo,
		Producer:  pub,
		Workflow:  workflow.New("", "", "", false),
		IDGen:     idgen.New("", "", false),
		Validator: validator.New(nil),
		Property:  property.New("", "", false),
		User:      user.New("", "", "", false),
		Encryptor: encryption.New("", "", "", "", false),
		Cfg:       cfg,
	})
}

func createReq(tenant, prop string) *domain.WaterConnectionRequest {
	return &domain.WaterConnectionRequest{
		WaterConnection: &domain.WaterConnection{
			Connection: domain.Connection{TenantID: tenant, PropertyID: prop},
		},
	}
}

func TestCreateEnrichesPersistsPublishes(t *testing.T) {
	repo := &fakeRepo{}
	pub := &fakePublisher{}
	svc := newSvc(repo, pub)

	wc, err := svc.Create(context.Background(), createReq("pb.amritsar", "P-1"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if wc.ID == "" {
		t.Error("id not enriched")
	}
	if wc.ApplicationNo == "" {
		t.Error("applicationNo not generated (local synth expected when idgen disabled)")
	}
	if wc.Status != "Active" || wc.ApplicationStatus != "INITIATED" {
		t.Errorf("defaults: status=%q applicationStatus=%q", wc.Status, wc.ApplicationStatus)
	}
	if wc.AuditDetails == nil || wc.AuditDetails.CreatedTime == 0 {
		t.Error("audit details not set")
	}
	if len(repo.saved) != 1 {
		t.Errorf("repo.Save calls = %d, want 1", len(repo.saved))
	}
	if len(pub.topics) != 1 || pub.topics[0] != "save-ws-connection" {
		t.Errorf("published topics = %v", pub.topics)
	}
}

func TestCreateValidationErrorDoesNotPersist(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakePublisher{})

	_, err := svc.Create(context.Background(), createReq("", "P-1")) // missing tenant
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != "EG_WS_TENANTID_REQUIRED" || ae.Status != 400 {
		t.Fatalf("want 400 EG_WS_TENANTID_REQUIRED, got %v", err)
	}
	if len(repo.saved) != 0 {
		t.Error("must not persist on validation failure")
	}
}

func TestUpdateRequiresID(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakePublisher{})

	req := &domain.WaterConnectionRequest{WaterConnection: &domain.WaterConnection{}}
	_, err := svc.Update(context.Background(), req)
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != "EG_WS_ID_REQUIRED" {
		t.Fatalf("want EG_WS_ID_REQUIRED, got %v", err)
	}
	if len(repo.updated) != 0 {
		t.Error("must not update without id")
	}
}

func TestUpdatePersistsAndPublishes(t *testing.T) {
	repo := &fakeRepo{}
	pub := &fakePublisher{}
	svc := newSvc(repo, pub)

	req := &domain.WaterConnectionRequest{
		WaterConnection: &domain.WaterConnection{Connection: domain.Connection{ID: "abc", TenantID: "pb.amritsar"}},
	}
	if _, err := svc.Update(context.Background(), req); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(repo.updated) != 1 {
		t.Errorf("repo.Update calls = %d, want 1", len(repo.updated))
	}
	if len(pub.topics) != 1 || pub.topics[0] != "update-ws-connection" {
		t.Errorf("published topics = %v", pub.topics)
	}
}

func TestSearchClampsLimitAndReturnsCount(t *testing.T) {
	repo := &fakeRepo{rows: []domain.WaterConnection{{}}, count: 7}
	svc := newSvc(repo, &fakePublisher{})

	crit := &domain.SearchCriteria{TenantID: "pb.amritsar", Limit: 9999}
	rows, total, err := svc.Search(context.Background(), crit)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if crit.Limit != 500 {
		t.Errorf("limit not clamped to MaxLimit: %d", crit.Limit)
	}
	if total != 7 || len(rows) != 1 {
		t.Errorf("rows=%d total=%d", len(rows), total)
	}
}
