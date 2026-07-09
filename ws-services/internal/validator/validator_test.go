package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/egov/ws-services/internal/domain"
	"github.com/egov/ws-services/pkg/apperr"
)

// newValidator with a nil MDMS client => master checks are skipped, only field
// validation runs (the local/offline path).
func newValidator() *Validator { return New(nil) }

func req(wc *domain.WaterConnection) *domain.WaterConnectionRequest {
	return &domain.WaterConnectionRequest{WaterConnection: wc}
}

func TestValidateCreateMissingPayload(t *testing.T) {
	if err := newValidator().ValidateCreate(context.Background(), &domain.WaterConnectionRequest{}); err == nil {
		t.Fatal("expected error for missing WaterConnection")
	}
}

func TestValidateCreateMissingTenant(t *testing.T) {
	err := newValidator().ValidateCreate(context.Background(), req(&domain.WaterConnection{
		Connection: domain.Connection{PropertyID: "P-1"},
	}))
	assertBadRequest(t, err, "EG_WS_TENANTID_REQUIRED")
}

func TestValidateCreateMissingProperty(t *testing.T) {
	err := newValidator().ValidateCreate(context.Background(), req(&domain.WaterConnection{
		Connection: domain.Connection{TenantID: "pb.amritsar"},
	}))
	assertBadRequest(t, err, "EG_WS_PROPERTYID_REQUIRED")
}

func TestValidateCreateOK(t *testing.T) {
	err := newValidator().ValidateCreate(context.Background(), req(&domain.WaterConnection{
		Connection: domain.Connection{TenantID: "pb.amritsar", PropertyID: "P-1"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertBadRequest(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s, got nil", code)
	}
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != code || ae.Status != 400 {
		t.Fatalf("expected 400 %s, got %v", code, err)
	}
}
