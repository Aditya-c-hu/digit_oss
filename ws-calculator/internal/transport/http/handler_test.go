package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/egov/ws-calculator/config"
	"github.com/egov/ws-calculator/internal/billing"
	"github.com/egov/ws-calculator/internal/domain"
	"github.com/egov/ws-calculator/internal/service"
	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func newTestEngine() *gin.Engine {
	cfg := &config.Config{FormFee: 100, ScrutinyFee: 50, SecurityCharge: 500, RoadCuttingRate: 200}
	calc := service.NewCalculationService(nil, nil, cfg)
	demand := service.NewDemandService(calc, billing.New("", "", "", "", false))
	meter := service.NewMeterService(nil)
	eng := gin.New()
	New(calc, demand, meter).Register(eng, "/ws-calculator")
	return eng
}

func TestEstimateHandlerReturnsFees(t *testing.T) {
	eng := newTestEngine()
	body := `{"RequestInfo":{"apiId":"ws"},"CalculationCriteria":[{"tenantId":"pb.amritsar","applicationNo":"WS/1"}]}`
	r := httptest.NewRequest(http.MethodPost, "/ws-calculator/waterCalculator/_estimate", strings.NewReader(body))
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var res domain.CalculationRes
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(res.Calculation) != 1 {
		t.Fatalf("calculations = %d, want 1", len(res.Calculation))
	}
	if got := res.Calculation[0].TotalAmount; got != 650 {
		t.Errorf("total = %v, want 650 (form+scrutiny+security)", got)
	}
}

func TestEstimateHandlerBadBody(t *testing.T) {
	eng := newTestEngine()
	r := httptest.NewRequest(http.MethodPost, "/ws-calculator/waterCalculator/_estimate", strings.NewReader("{not-json"))
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var res domain.ErrorRes
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("body not ErrorRes: %v", err)
	}
	if len(res.Errors) == 0 || res.Errors[0].Code != "EG_WS_INVALID_REQUEST" {
		t.Errorf("errors = %+v", res.Errors)
	}
}

func TestParseGetBillCriteria(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/x?tenantId=pb.amritsar&consumerCodes=C1,C2", nil)
	c := parseGetBillCriteria(r)
	if c.TenantID != "pb.amritsar" || len(c.ConsumerCode) != 2 {
		t.Errorf("got %+v", c)
	}
}

func TestParseMeterSearchCriteria(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/x?tenantId=pb&connectionNos=A&connectionNos=B&limit=20&offset=3", nil)
	c := parseMeterSearchCriteria(r)
	if c.TenantID != "pb" || len(c.ConnectionNos) != 2 || c.Limit != 20 || c.Offset != 3 {
		t.Errorf("got %+v", c)
	}
}
