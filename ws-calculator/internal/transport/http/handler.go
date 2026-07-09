// Package httptransport is the REST transport layer (the Java @RestController
// equivalent) for ws-calculator. It uses the Gin framework for routing and
// request binding, maps requests to domain DTOs, calls the service layer, and
// writes the eGov response envelope. No business logic, no SQL. Named
// httptransport (not http) so net/http can be imported for status constants;
// its directory is transport/http.
package httptransport

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/egov/ws-calculator/internal/domain"
	"github.com/egov/ws-calculator/internal/service"
	"github.com/egov/ws-calculator/pkg/apperr"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Calc   *service.CalculationService
	Demand *service.DemandService
	Meter  *service.MeterService
}

func New(calc *service.CalculationService, demand *service.DemandService, meter *service.MeterService) *Handler {
	return &Handler{Calc: calc, Demand: demand, Meter: meter}
}

// Register matches Java's CalculatorController + MeterReadingController routes.
func (h *Handler) Register(r gin.IRouter, prefix string) {
	r.POST(prefix+"/waterCalculator/_estimate", h.Estimate)
	r.POST(prefix+"/waterCalculator/_calculate", h.Calculate)
	r.POST(prefix+"/waterCalculator/_updateDemand", h.UpdateDemand)
	r.POST(prefix+"/waterCalculator/_jobscheduler", h.JobScheduler)
	r.POST(prefix+"/waterCalculator/_applyAdhocTax", h.ApplyAdhocTax)
	r.POST(prefix+"/meterConnection/_create", h.CreateMeterReading)
	r.POST(prefix+"/meterConnection/_search", h.SearchMeterReadings)
}

func (h *Handler) Estimate(c *gin.Context) {
	var req domain.CalculationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		failJSON(c, req.RequestInfo, http.StatusBadRequest, "EG_WS_INVALID_REQUEST", err.Error())
		return
	}
	c.JSON(http.StatusOK, domain.CalculationRes{
		ResponseInfo: respInfo(req.RequestInfo),
		Calculation:  h.Calc.Estimate(c.Request.Context(), &req),
	})
}

func (h *Handler) Calculate(c *gin.Context) {
	var req domain.CalculationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		failJSON(c, req.RequestInfo, http.StatusBadRequest, "EG_WS_INVALID_REQUEST", err.Error())
		return
	}
	c.JSON(http.StatusOK, domain.CalculationRes{
		ResponseInfo: respInfo(req.RequestInfo),
		Calculation:  h.Calc.Calculate(c.Request.Context(), &req),
	})
}

func (h *Handler) UpdateDemand(c *gin.Context) {
	var wrapper domain.RequestInfoWrapper
	_ = c.ShouldBindJSON(&wrapper)
	crit := parseGetBillCriteria(c.Request)
	demands := h.Demand.UpdateDemands(c.Request.Context(), &wrapper, &crit)
	c.JSON(http.StatusOK, domain.DemandResponse{
		ResponseInfo: respInfo(wrapper.RequestInfo),
		Demands:      demands,
	})
}

func (h *Handler) JobScheduler(c *gin.Context) {
	var req domain.BulkBillReq
	if err := c.ShouldBindJSON(&req); err != nil {
		failJSON(c, req.RequestInfo, http.StatusBadRequest, "EG_WS_INVALID_REQUEST", err.Error())
		return
	}
	if err := h.Calc.GenerateDemandsForCycle(c.Request.Context(), req.RequestInfo, req.BulkBillCriteria); err != nil {
		fail(c, req.RequestInfo, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "scheduled"})
}

func (h *Handler) ApplyAdhocTax(c *gin.Context) {
	var req domain.AdhocTaxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		failJSON(c, req.RequestInfo, http.StatusBadRequest, "EG_WS_INVALID_REQUEST", err.Error())
		return
	}
	c.JSON(http.StatusOK, domain.CalculationRes{
		ResponseInfo: respInfo(req.RequestInfo),
		Calculation:  h.Calc.ApplyAdhocTax(c.Request.Context(), &req),
	})
}

func (h *Handler) CreateMeterReading(c *gin.Context) {
	var req domain.MeterConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failJSON(c, req.RequestInfo, http.StatusBadRequest, "EG_WS_INVALID_REQUEST", err.Error())
		return
	}
	m, err := h.Meter.Create(c.Request.Context(), &req)
	if err != nil {
		fail(c, req.RequestInfo, err)
		return
	}
	c.JSON(http.StatusOK, domain.MeterReadingResponse{
		ResponseInfo:  respInfo(req.RequestInfo),
		MeterReadings: []domain.MeterReading{*m},
	})
}

func (h *Handler) SearchMeterReadings(c *gin.Context) {
	var wrapper domain.RequestInfoWrapper
	_ = c.ShouldBindJSON(&wrapper)
	crit := parseMeterSearchCriteria(c.Request)
	rows, err := h.Meter.Search(c.Request.Context(), &crit)
	if err != nil {
		fail(c, wrapper.RequestInfo, err)
		return
	}
	c.JSON(http.StatusOK, domain.MeterReadingResponse{
		ResponseInfo:  respInfo(wrapper.RequestInfo),
		MeterReadings: rows,
	})
}

// queryList accepts repeated (?k=a&k=b) and comma (?k=a,b) list forms.
func queryList(r *http.Request, key string) []string {
	vals := r.URL.Query()[key]
	if len(vals) == 1 && strings.Contains(vals[0], ",") {
		return strings.Split(vals[0], ",")
	}
	return vals
}

func parseGetBillCriteria(r *http.Request) domain.GetBillCriteria {
	return domain.GetBillCriteria{
		TenantID:     r.URL.Query().Get("tenantId"),
		ConsumerCode: queryList(r, "consumerCodes"),
	}
}

func parseMeterSearchCriteria(r *http.Request) domain.MeterReadingSearchCriteria {
	q := r.URL.Query()
	atoi := func(s string) int { n, _ := strconv.Atoi(s); return n }
	return domain.MeterReadingSearchCriteria{
		TenantID:      q.Get("tenantId"),
		ConnectionNos: queryList(r, "connectionNos"),
		IDs:           queryList(r, "ids"),
		Limit:         atoi(q.Get("limit")),
		Offset:        atoi(q.Get("offset")),
	}
}

func respInfo(req *domain.RequestInfo) *domain.ResponseInfo {
	out := &domain.ResponseInfo{TS: time.Now().UnixMilli(), Status: "successful"}
	if req == nil {
		return out
	}
	out.APIID = req.APIID
	out.Ver = req.Ver
	out.MsgID = req.MsgID
	return out
}

func failJSON(c *gin.Context, req *domain.RequestInfo, status int, code, msg string) {
	ri := respInfo(req)
	ri.Status = "failed"
	c.JSON(status, domain.ErrorRes{
		ResponseInfo: ri,
		Errors:       []domain.ErrorItem{{Code: code, Message: msg}},
	})
}

func fail(c *gin.Context, req *domain.RequestInfo, err error) {
	status, code, msg := apperr.Resolve(err)
	if status >= http.StatusInternalServerError {
		// Do not leak internal error detail to the client; log server-side.
		slog.Error("request failed", "code", code, "detail", msg)
		msg = "internal server error"
	}
	failJSON(c, req, status, code, msg)
}
