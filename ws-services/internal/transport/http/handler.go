// Package httptransport is the REST transport layer (the Java @RestController
// equivalent). It uses the Gin framework for routing and request binding,
// maps requests to domain DTOs, calls the service layer, and writes the eGov
// response envelope. It contains no business logic and no SQL. The package is
// named httptransport (not http) so it can import the standard net/http package
// for status constants without a name clash; its directory is transport/http.
package httptransport

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/egov/ws-services/internal/domain"
	"github.com/egov/ws-services/internal/service"
	"github.com/egov/ws-services/pkg/apperr"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{Svc: svc}
}

// Register attaches all REST routes under the given prefix to the Gin router.
// The path layout matches the Spring controller: /wc/_create, /wc/_search,
// /wc/_update, /wc/_plainsearch, /wc/_encryptOldData.
func (h *Handler) Register(r gin.IRouter, prefix string) {
	r.POST(prefix+"/wc/_create", h.Create)
	r.POST(prefix+"/wc/_update", h.Update)
	r.POST(prefix+"/wc/_search", h.Search)
	r.POST(prefix+"/wc/_plainsearch", h.Search)
	r.POST(prefix+"/wc/_encryptOldData", h.EncryptOldData)
}

// Create creates a new water connection application (mirrors Java POST /wc/_create).
func (h *Handler) Create(c *gin.Context) {
	var req domain.WaterConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failJSON(c, req.RequestInfo, http.StatusBadRequest, "EG_WS_INVALID_REQUEST", err.Error())
		return
	}
	req.CreateCall = true
	wc, err := h.Svc.Create(c.Request.Context(), &req)
	if err != nil {
		fail(c, req.RequestInfo, err)
		return
	}
	c.JSON(http.StatusOK, domain.WaterConnectionResponse{
		ResponseInfo:    responseInfoFromRequest(req.RequestInfo, true),
		WaterConnection: []domain.WaterConnection{*wc},
	})
}

// Update advances/updates a water connection (POST /wc/_update).
func (h *Handler) Update(c *gin.Context) {
	var req domain.WaterConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failJSON(c, req.RequestInfo, http.StatusBadRequest, "EG_WS_INVALID_REQUEST", err.Error())
		return
	}
	wc, err := h.Svc.Update(c.Request.Context(), &req)
	if err != nil {
		fail(c, req.RequestInfo, err)
		return
	}
	c.JSON(http.StatusOK, domain.WaterConnectionResponse{
		ResponseInfo:    responseInfoFromRequest(req.RequestInfo, true),
		WaterConnection: []domain.WaterConnection{*wc},
	})
}

// Search returns connections matching the query criteria (POST /wc/_search).
// Criteria travel as query params; the body carries RequestInfo.
func (h *Handler) Search(c *gin.Context) {
	var wrapper domain.RequestInfoWrapper
	_ = c.ShouldBindJSON(&wrapper) // body is optional; criteria come from query params
	crit := parseSearchCriteria(c.Request)

	rows, total, err := h.Svc.Search(c.Request.Context(), &crit)
	if err != nil {
		fail(c, wrapper.RequestInfo, err)
		return
	}
	c.JSON(http.StatusOK, domain.WaterConnectionResponse{
		ResponseInfo:    responseInfoFromRequest(wrapper.RequestInfo, true),
		WaterConnection: rows,
		TotalCount:      total,
	})
}

// EncryptOldData mirrors the Java endpoint which hard-fails unless privacy
// migration is enabled.
func (h *Handler) EncryptOldData(c *gin.Context) {
	failJSON(c, nil, http.StatusBadRequest, "EG_WS_ENC_OLD_DATA_ERROR", "Privacy disabled: The encryption of old data is disabled")
}

// parseSearchCriteria maps query params into the search criteria, accepting both
// repeated (?ids=a&ids=b) and comma-separated (?ids=a,b) list forms.
func parseSearchCriteria(r *http.Request) domain.SearchCriteria {
	q := r.URL.Query()
	atoiInt := func(s string) int { n, _ := strconv.Atoi(s); return n }
	atoiI64 := func(s string) int64 { n, _ := strconv.ParseInt(s, 10, 64); return n }
	list := func(key string) []string {
		vals := q[key]
		if len(vals) == 1 && strings.Contains(vals[0], ",") {
			return strings.Split(vals[0], ",")
		}
		return vals
	}
	c := domain.SearchCriteria{
		TenantID:            q.Get("tenantId"),
		IDs:                 list("ids"),
		ApplicationNumber:   list("applicationNumber"),
		ApplicationStatus:   list("applicationStatus"),
		ConnectionNumber:    list("connectionNumber"),
		OldConnectionNumber: q.Get("oldConnectionNumber"),
		MobileNumber:        q.Get("mobileNumber"),
		PropertyID:          q.Get("propertyId"),
		Status:              q.Get("status"),
		FromDate:            atoiI64(q.Get("fromDate")),
		ToDate:              atoiI64(q.Get("toDate")),
		Offset:              atoiInt(q.Get("offset")),
		Limit:               atoiInt(q.Get("limit")),
		ApplicationType:     q.Get("applicationType"),
		Locality:            q.Get("locality"),
		SearchType:          q.Get("searchType"),
		DoorNo:              q.Get("doorNo"),
		OwnerName:           q.Get("ownerName"),
		Assignee:            q.Get("assignee"),
	}
	if v := q.Get("isPropertyDetailsRequired"); v != "" {
		c.IsPropertyDetailsRequired, _ = strconv.ParseBool(v)
	}
	return c
}

func responseInfoFromRequest(req *domain.RequestInfo, ok bool) *domain.ResponseInfo {
	out := &domain.ResponseInfo{TS: time.Now().UnixMilli()}
	status := "successful"
	if !ok {
		status = "failed"
	}
	out.Status = status
	if req == nil {
		return out
	}
	out.APIID = req.APIID
	out.Ver = req.Ver
	out.MsgID = req.MsgID
	return out
}

// failJSON writes the eGov error envelope (ResponseInfo + Errors) at an explicit
// status, for client-detected errors like malformed bodies/queries.
func failJSON(c *gin.Context, req *domain.RequestInfo, status int, code, msg string) {
	c.JSON(status, domain.ErrorRes{
		ResponseInfo: responseInfoFromRequest(req, false),
		Errors:       []domain.ErrorItem{{Code: code, Message: msg}},
	})
}

// fail maps a service error to its eGov code + HTTP status (400 for
// validation/business, 500 for infra) and writes the envelope.
func fail(c *gin.Context, req *domain.RequestInfo, err error) {
	status, code, msg := apperr.Resolve(err)
	if status >= http.StatusInternalServerError {
		// Do not leak internal error detail (SQL text, downstream URLs) to the
		// client; log it server-side and return a generic message.
		slog.Error("request failed", "code", code, "detail", msg)
		msg = "internal server error"
	}
	failJSON(c, req, status, code, msg)
}
