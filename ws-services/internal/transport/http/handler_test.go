package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/egov/ws-services/internal/domain"
	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func TestParseSearchCriteria(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost,
		"/ws-services/wc/_search?tenantId=pb.amritsar&ids=a,b&applicationNumber=A1&applicationNumber=A2"+
			"&limit=10&offset=5&propertyId=P-1&isPropertyDetailsRequired=true&fromDate=100&toDate=200", nil)
	c := parseSearchCriteria(r)

	if c.TenantID != "pb.amritsar" {
		t.Errorf("tenantId = %q", c.TenantID)
	}
	if len(c.IDs) != 2 || c.IDs[0] != "a" || c.IDs[1] != "b" {
		t.Errorf("ids = %v (want [a b] from comma form)", c.IDs)
	}
	if len(c.ApplicationNumber) != 2 {
		t.Errorf("applicationNumber = %v (want 2 from repeated form)", c.ApplicationNumber)
	}
	if c.Limit != 10 || c.Offset != 5 {
		t.Errorf("limit/offset = %d/%d", c.Limit, c.Offset)
	}
	if c.PropertyID != "P-1" {
		t.Errorf("propertyId = %q", c.PropertyID)
	}
	if !c.IsPropertyDetailsRequired {
		t.Error("isPropertyDetailsRequired = false, want true")
	}
	if c.FromDate != 100 || c.ToDate != 200 {
		t.Errorf("fromDate/toDate = %d/%d", c.FromDate, c.ToDate)
	}
}

func TestEncryptOldDataReturns400Envelope(t *testing.T) {
	eng := gin.New()
	(&Handler{}).Register(eng, "/ws-services")
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/ws-services/wc/_encryptOldData", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var res domain.ErrorRes
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("body not eGov ErrorRes: %v", err)
	}
	if len(res.Errors) != 1 || res.Errors[0].Code != "EG_WS_ENC_OLD_DATA_ERROR" {
		t.Errorf("errors = %+v", res.Errors)
	}
	if res.ResponseInfo == nil || res.ResponseInfo.Status != "failed" {
		t.Errorf("responseInfo = %+v, want status failed", res.ResponseInfo)
	}
}

func TestResponseInfoFromRequest(t *testing.T) {
	ri := &domain.RequestInfo{APIID: "ws", Ver: "1.0", MsgID: "m1"}
	out := responseInfoFromRequest(ri, true)
	if out.Status != "successful" || out.APIID != "ws" || out.Ver != "1.0" || out.MsgID != "m1" {
		t.Errorf("got %+v", out)
	}
	if responseInfoFromRequest(nil, false).Status != "failed" {
		t.Error("nil request should still yield failed status")
	}
}
