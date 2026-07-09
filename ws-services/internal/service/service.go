package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egov/ws-services/config"
	"github.com/egov/ws-services/internal/domain"
	"github.com/egov/ws-services/internal/encryption"
	"github.com/egov/ws-services/internal/idgen"
	"github.com/egov/ws-services/internal/property"
	"github.com/egov/ws-services/internal/user"
	"github.com/egov/ws-services/internal/validator"
	"github.com/egov/ws-services/internal/workflow"
	"github.com/egov/ws-services/pkg/apperr"
	"github.com/google/uuid"
)

// Repository is the narrow persistence surface the service needs. Satisfied by
// *repository.WaterRepository; narrowed to an interface so the service can be
// unit-tested with a fake (no DB).
type Repository interface {
	Save(ctx context.Context, wc *domain.WaterConnection) error
	Update(ctx context.Context, wc *domain.WaterConnection) error
	Search(ctx context.Context, c *domain.SearchCriteria) ([]domain.WaterConnection, error)
	Count(ctx context.Context, c *domain.SearchCriteria) (int, error)
}

// Publisher is the narrow Kafka surface the service needs. Satisfied by
// *kafka.Producer.
type Publisher interface {
	Push(ctx context.Context, topic string, payload any) error
}

type Service struct {
	Repo      Repository
	Producer  Publisher
	Workflow  *workflow.Client
	IDGen     *idgen.Client
	Validator *validator.Validator
	Property  *property.Client
	User      *user.Client
	Encryptor *encryption.Client
	Cfg       *config.Config
}

type Dependencies struct {
	Repo      Repository
	Producer  Publisher
	Workflow  *workflow.Client
	IDGen     *idgen.Client
	Validator *validator.Validator
	Property  *property.Client
	User      *user.Client
	Encryptor *encryption.Client
	Cfg       *config.Config
}

func New(deps Dependencies) *Service {
	return &Service{
		Repo:      deps.Repo,
		Producer:  deps.Producer,
		Workflow:  deps.Workflow,
		IDGen:     deps.IDGen,
		Validator: deps.Validator,
		Property:  deps.Property,
		User:      deps.User,
		Encryptor: deps.Encryptor,
		Cfg:       deps.Cfg,
	}
}

// Create implements the equivalent of WaterServiceImpl.createWaterConnection:
//  1. Validate (light)
//  2. Enrich (ids, audit, applicationNo)
//  3. Run workflow transition (or stub)
//  4. Persist
//  5. Publish save-ws-connection topic for downstream consumers
func (s *Service) Create(ctx context.Context, req *domain.WaterConnectionRequest) (*domain.WaterConnection, error) {
	if err := s.Validator.ValidateCreate(ctx, req); err != nil {
		return nil, err
	}
	wc := req.WaterConnection

	if err := s.validateProperty(ctx, req, wc); err != nil {
		return nil, err
	}
	if err := s.enrichNewConnection(ctx, req, wc); err != nil {
		return nil, err
	}
	if err := s.resolveConnectionHolders(ctx, req, wc); err != nil {
		return nil, err
	}
	if err := s.transitionCreateWorkflow(ctx, req, wc); err != nil {
		return nil, err
	}
	if err := s.encryptConnection(ctx, wc); err != nil {
		return nil, err
	}
	if err := s.persistCreatedConnection(ctx, wc); err != nil {
		return nil, err
	}

	_ = s.Producer.Push(ctx, s.Cfg.OnWaterSavedTopic, req)
	s.notify(ctx, req, wc)

	return wc, nil
}

func (s *Service) validateProperty(ctx context.Context, req *domain.WaterConnectionRequest, wc *domain.WaterConnection) error {
	if s.Property == nil || !s.Property.Enabled {
		return nil
	}
	return s.Property.Validate(ctx, req.RequestInfo, wc.TenantID, wc.PropertyID)
}

func (s *Service) enrichNewConnection(ctx context.Context, req *domain.WaterConnectionRequest, wc *domain.WaterConnection) error {
	now := time.Now().UnixMilli()
	wc.ID = uuid.NewString()
	appNo, err := s.applicationNumber(ctx, req)
	if err != nil {
		return err
	}
	wc.ApplicationNo = appNo
	applyCreateDefaults(wc)
	wc.AuditDetails = auditDetails(req, now)
	assignChildIDs(wc)
	return nil
}

func applyCreateDefaults(wc *domain.WaterConnection) {
	if wc.Status == "" {
		wc.Status = "Active"
	}
	if wc.ApplicationStatus == "" {
		wc.ApplicationStatus = "INITIATED"
	}
}

func auditDetails(req *domain.WaterConnectionRequest, now int64) *domain.AuditDetails {
	auditUser := ""
	if req.RequestInfo != nil && req.RequestInfo.UserInfo != nil {
		auditUser = req.RequestInfo.UserInfo.UUID
	}
	return &domain.AuditDetails{
		CreatedBy:        auditUser,
		LastModifiedBy:   auditUser,
		CreatedTime:      now,
		LastModifiedTime: now,
	}
}

func assignChildIDs(wc *domain.WaterConnection) {
	for i := range wc.Documents {
		if wc.Documents[i].ID == "" {
			wc.Documents[i].ID = uuid.NewString()
		}
	}
	for i := range wc.PlumberInfo {
		if wc.PlumberInfo[i].ID == "" {
			wc.PlumberInfo[i].ID = uuid.NewString()
		}
	}
	for i := range wc.RoadCuttingInfo {
		if wc.RoadCuttingInfo[i].ID == "" {
			wc.RoadCuttingInfo[i].ID = uuid.NewString()
		}
		wc.RoadCuttingInfo[i].Active = true
	}
}

func (s *Service) resolveConnectionHolders(ctx context.Context, req *domain.WaterConnectionRequest, wc *domain.WaterConnection) error {
	if s.User == nil || !s.User.Enabled {
		return nil
	}
	for i := range wc.ConnectionHolders {
		if err := s.resolveConnectionHolder(ctx, req, wc, i); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) resolveConnectionHolder(ctx context.Context, req *domain.WaterConnectionRequest, wc *domain.WaterConnection, i int) error {
	h := &wc.ConnectionHolders[i]
	if h.UUID != "" {
		return nil
	}
	resolved, err := s.User.Resolve(ctx, req.RequestInfo, wc.TenantID, h.Name, h.MobileNumber)
	if err != nil {
		return apperr.Internal("EG_WS_USER_ERROR", "connection-holder user resolve failed: "+err.Error())
	}
	h.UUID = resolved
	return nil
}

func (s *Service) transitionCreateWorkflow(ctx context.Context, req *domain.WaterConnectionRequest, wc *domain.WaterConnection) error {
	if wc.ProcessInstance == nil || wc.ProcessInstance.Action == "" {
		return nil
	}
	pi := *wc.ProcessInstance
	pi.BusinessID = wc.ApplicationNo
	pi.TenantID = wc.TenantID
	pi.ModuleName = "ws-services"
	pi.BusinessService = s.Cfg.BusinessServiceValue
	return s.transitionWorkflow(ctx, req, wc, pi)
}

func (s *Service) transitionWorkflow(ctx context.Context, req *domain.WaterConnectionRequest, wc *domain.WaterConnection, pi domain.ProcessInstance) error {
	out, err := s.Workflow.Transition(ctx, workflow.TransitionRequest{
		RequestInfo:      req.RequestInfo,
		ProcessInstances: []domain.ProcessInstance{pi},
	})
	if err != nil {
		return apperr.Internal("EG_WS_WORKFLOW_ERROR", "workflow transition failed: "+err.Error())
	}
	if len(out.ProcessInstances) > 0 && out.ProcessInstances[0].State != nil {
		wc.ApplicationStatus = out.ProcessInstances[0].State.ApplicationStatus
	}
	return nil
}

func (s *Service) encryptConnection(ctx context.Context, wc *domain.WaterConnection) error {
	if s.Encryptor == nil || !s.Encryptor.Enabled {
		return nil
	}
	if err := s.Encryptor.EncryptConnection(ctx, wc); err != nil {
		return apperr.Internal("EG_WS_ENCRYPTION_ERROR", err.Error())
	}
	return nil
}

func (s *Service) persistCreatedConnection(ctx context.Context, wc *domain.WaterConnection) error {
	if s.Cfg.IsPersisterEnabled {
		return nil
	}
	return s.Repo.Save(ctx, wc)
}

// Update mirrors WaterServiceImpl.updateWaterConnection.
func (s *Service) Update(ctx context.Context, req *domain.WaterConnectionRequest) (*domain.WaterConnection, error) {
	if req == nil || req.WaterConnection == nil {
		return nil, apperr.BadRequest("EG_WS_PAYLOAD_REQUIRED", "WaterConnection payload is required")
	}
	wc := req.WaterConnection
	if wc.ID == "" {
		return nil, apperr.BadRequest("EG_WS_ID_REQUIRED", "id is required for update")
	}

	if wc.ProcessInstance != nil && wc.ProcessInstance.Action != "" {
		pi := *wc.ProcessInstance
		pi.BusinessID = wc.ApplicationNo
		pi.TenantID = wc.TenantID
		pi.ModuleName = "ws-services"
		if pi.BusinessService == "" {
			pi.BusinessService = s.Cfg.BusinessServiceValue
		}
		if err := s.transitionWorkflow(ctx, req, wc, pi); err != nil {
			return nil, err
		}
	}

	// See Create: persister owns the write when enabled.
	if !s.Cfg.IsPersisterEnabled {
		if err := s.Repo.Update(ctx, wc); err != nil {
			return nil, err
		}
	}

	_ = s.Producer.Push(ctx, s.Cfg.OnWaterUpdatedTopic, req)
	s.notify(ctx, req, wc)
	return wc, nil
}

// notify publishes SMS / email notification events for the connection's current
// status. Gated by config flags; topics are consumed by egov-notification-*.
// Fire-and-forget — notification failure never blocks the connection write.
func (s *Service) notify(ctx context.Context, req *domain.WaterConnectionRequest, wc *domain.WaterConnection) {
	mobile := ""
	if req.RequestInfo != nil && req.RequestInfo.UserInfo != nil {
		mobile = req.RequestInfo.UserInfo.MobileNumber
	}
	msg := fmt.Sprintf("Your water connection application %s is now %s.", wc.ApplicationNo, wc.ApplicationStatus)

	if s.Cfg.IsSMSEnabled && s.Cfg.SmsNotifTopic != "" && mobile != "" {
		_ = s.Producer.Push(ctx, s.Cfg.SmsNotifTopic, map[string]any{
			"tenantId":     wc.TenantID,
			"mobileNumber": mobile,
			"message":      msg,
		})
	}
	if s.Cfg.IsEmailEnabled && s.Cfg.EmailNotifTopic != "" {
		_ = s.Producer.Push(ctx, s.Cfg.EmailNotifTopic, map[string]any{
			"tenantId": wc.TenantID,
			"subject":  "Water Connection Update",
			"body":     msg,
		})
	}
}

func (s *Service) Search(ctx context.Context, c *domain.SearchCriteria) ([]domain.WaterConnection, int, error) {
	if c.Limit > s.Cfg.MaxLimit {
		c.Limit = s.Cfg.MaxLimit
	}
	if c.Limit == 0 {
		c.Limit = s.Cfg.DefaultLimit
	}
	rows, err := s.Repo.Search(ctx, c)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.Repo.Count(ctx, c)
	if err != nil {
		return nil, 0, err
	}

	// Decrypt holder / plumber PII after load (egov-enc-service), when enabled.
	// Matches Java enrichmentService.decryptConnectionDetails on search.
	if s.Encryptor != nil && s.Encryptor.Enabled {
		for i := range rows {
			if err := s.Encryptor.DecryptConnection(ctx, nil, &rows[i]); err != nil {
				return nil, 0, apperr.Internal("EG_WS_DECRYPTION_ERROR", err.Error())
			}
		}
	}
	return rows, count, nil
}

// applicationNumber returns an application number from egov-idgen when enabled,
// falling back to local synthesis for offline / local deployments.
func (s *Service) applicationNumber(ctx context.Context, req *domain.WaterConnectionRequest) (string, error) {
	wc := req.WaterConnection
	if s.IDGen != nil && s.IDGen.Enabled {
		return s.IDGen.Generate(ctx, req.RequestInfo, s.Cfg.WCAPIDName, wc.TenantID, s.Cfg.WCAPIDFormat)
	}
	return generateApplicationNo(wc.TenantID, time.Now().UnixMilli()), nil
}

// generateApplicationNo synthesizes an application number when running without
// the eGov idgen service (local / offline mode only).
func generateApplicationNo(tenantID string, now int64) string {
	parts := strings.Split(tenantID, ".")
	city := parts[len(parts)-1]
	return fmt.Sprintf("WS_AP/%s/%d/%s", strings.ToUpper(city), now, uuid.New().String()[:8])
}
