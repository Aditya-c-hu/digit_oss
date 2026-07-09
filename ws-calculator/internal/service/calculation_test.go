package service

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/egov/ws-calculator/config"
	"github.com/egov/ws-calculator/internal/domain"
)

func testConfig() *config.Config {
	return &config.Config{
		FormFee:                100,
		ScrutinyFee:            50,
		SecurityCharge:         500,
		RoadCuttingRate:        200,
		PenaltyRate:            10,
		PenaltyApplicableDays:  30,
		InterestRate:           12,
		InterestApplicableDays: 30,
	}
}

func newTestSvc() *CalculationService {
	// nil repo/producer are safe for the pure-calculation paths exercised here.
	return NewCalculationService(nil, nil, testConfig())
}

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 0.001 }

func headAmount(c domain.Calculation, code string) (float64, bool) {
	for _, e := range c.TaxHeadEstimates {
		if e.TaxHeadCode == code {
			return e.EstimateAmount, true
		}
	}
	return 0, false
}

func TestEstimateOneTimeFees(t *testing.T) {
	s := newTestSvc()
	calc := s.estimateOne(domain.CalculationCriteria{TenantID: "pb.amritsar", ApplicationNo: "WS/1"})

	if _, ok := headAmount(calc, WSChargeHead); ok {
		t.Fatalf("estimate must not contain water charge %s", WSChargeHead)
	}
	for _, code := range []string{WSFormFee, WSScrutinyFee, WSSecurityCharge} {
		if _, ok := headAmount(calc, code); !ok {
			t.Errorf("missing one-time fee head %s", code)
		}
	}
	if !approxEqual(calc.TotalAmount, 650) { // 100 + 50 + 500
		t.Errorf("total = %v, want 650", calc.TotalAmount)
	}
}

func TestEstimateRoadCuttingCharge(t *testing.T) {
	s := newTestSvc()
	calc := s.estimateOne(domain.CalculationCriteria{
		TenantID:        "pb.amritsar",
		WaterConnection: &domain.WaterConnection{RoadCuttingArea: 2.5},
	})
	rc, ok := headAmount(calc, WSRoadCuttingCharge)
	if !ok {
		t.Fatal("missing road-cutting charge")
	}
	if !approxEqual(rc, 500) { // 2.5 * 200
		t.Errorf("road-cutting = %v, want 500", rc)
	}
	if !approxEqual(calc.TotalAmount, 1150) { // 100 + 50 + 500 + 500
		t.Errorf("total = %v, want 1150", calc.TotalAmount)
	}
}

func TestCalculateNonMeteredMinimumCharge(t *testing.T) {
	s := newTestSvc()
	calc := s.calculateOne(context.Background(), domain.CalculationCriteria{
		TenantID:     "pb.amritsar",
		ConnectionNo: "WS-1",
		WaterConnection: &domain.WaterConnection{
			ConnectionType:     "Non_Metered",
			ConnectionCategory: "RESIDENTIAL",
		},
	})
	charge, ok := headAmount(calc, WSChargeHead)
	if !ok {
		t.Fatal("missing water charge head")
	}
	if !approxEqual(charge, 100) { // seeded RESIDENTIAL/Non_Metered minimum
		t.Errorf("charge = %v, want 100", charge)
	}
}

func TestComputeRangeChargeMetered(t *testing.T) {
	// Uses the slab `charge` field (parity with Java getWaterEstimationCharge).
	// 0-10 @5, 10-20 @10, 20+ @15.
	slabs := []domain.Slab{
		{From: 0, To: 10, Charge: 5},
		{From: 10, To: 20, Charge: 10},
		{From: 20, To: 1000000, Charge: 15},
	}
	cases := []struct {
		consumption float64
		want        float64
	}{
		{0, 0},
		{5, 25},   // 5*5
		{15, 100}, // (10-0)*5 + 5*10
		{25, 200}, // (10-0)*5 + 15*10  (Java decrements by tier span then charges remainder)
	}
	for _, tc := range cases {
		got := computeRangeCharge(tc.consumption, slabs, true)
		if !approxEqual(got, tc.want) {
			t.Errorf("computeRangeCharge(%v) = %v, want %v", tc.consumption, got, tc.want)
		}
	}
}

func TestComputeRangeChargeNonMetered(t *testing.T) {
	slabs := []domain.Slab{
		{From: 0, To: 2, Charge: 100},
		{From: 2, To: 5, Charge: 150},
	}
	if got := computeRangeCharge(3, slabs, false); !approxEqual(got, 450) { // 3 * 150
		t.Errorf("non-metered = %v, want 450", got)
	}
}

func TestRoundOff(t *testing.T) {
	cases := []struct {
		total float64
		want  float64
	}{
		{100.0, 0},
		{100.4, -0.4},
		{100.5, 0.5},
		{100.7, 0.3},
	}
	for _, tc := range cases {
		if got := roundOff(tc.total); !approxEqual(got, tc.want) {
			t.Errorf("roundOff(%v) = %v, want %v", tc.total, got, tc.want)
		}
	}
}

func TestPenaltyAndInterest(t *testing.T) {
	s := newTestSvc()
	now := time.Now().UnixMilli()

	// Within grace period (due now) => no penalty/interest.
	if p, i := s.applyPenaltyAndInterest(100, now); p != 0 || i != 0 {
		t.Errorf("within grace: penalty=%v interest=%v, want 0/0", p, i)
	}

	// 60 days overdue => flat 10%% penalty + flat 12%% interest of the charge
	// (no day proration, matching Java PayService).
	due := now - 60*24*60*60*1000
	p, i := s.applyPenaltyAndInterest(100, due)
	if !approxEqual(p, 10) { // 100 * 10%
		t.Errorf("penalty = %v, want 10", p)
	}
	if !approxEqual(i, 12) { // 100 * 12%
		t.Errorf("interest = %v, want 12", i)
	}
}

func TestEstimateFromFeeSlabMDMS(t *testing.T) {
	s := newTestSvc()
	feeSlab := []json.RawMessage{json.RawMessage(`{"formFee":120,"scrutinyFee":60,"other":0,"taxpercentage":10,"meterCost":200}`)}
	roadType := []json.RawMessage{json.RawMessage(`{"code":"BT","unitCost":50}`)}
	plotSlab := []json.RawMessage{json.RawMessage(`{"from":0,"to":1000,"unitCost":300}`)}
	usageType := []json.RawMessage{json.RawMessage(`{"code":"RESIDENTIAL","unitCost":20}`)}
	s.RefreshFeeMasters(feeSlab, roadType, plotSlab, usageType)

	calc := s.estimateOne(domain.CalculationCriteria{
		TenantID: "pb.amritsar",
		WaterConnection: &domain.WaterConnection{
			ConnectionType: "Non Metered", RoadType: "BT", RoadCuttingArea: 2,
			UsageCategory: "RESIDENTIAL", LandArea: 100,
		},
	})
	// form 120 + scrutiny 60 + roadcut 50*2=100 + usage 20*2=40 + plot 300 = 620;
	// no meter (non-metered); tax 10% of 620 = 62; total 682.
	form, _ := headAmount(calc, WSFormFee)
	road, _ := headAmount(calc, WSRoadCuttingCharge)
	plot, _ := headAmount(calc, WSSecurityCharge)
	tax, _ := headAmount(calc, WSTaxAndCess)
	if !approxEqual(form, 120) || !approxEqual(road, 100) || !approxEqual(plot, 300) || !approxEqual(tax, 62) {
		t.Errorf("heads form=%v road=%v plot=%v tax=%v", form, road, plot, tax)
	}
	if _, ok := headAmount(calc, WSMeterCharge); ok {
		t.Error("non-metered must not have meter charge")
	}
	if !approxEqual(calc.TotalAmount, 682) {
		t.Errorf("total = %v, want 682", calc.TotalAmount)
	}
}

func TestRefreshTimeMastersFromMDMS(t *testing.T) {
	s := newTestSvc()
	penalty := []json.RawMessage{json.RawMessage(`{"rate":5,"applicableAfterDays":15}`)}
	interest := []json.RawMessage{json.RawMessage(`{"rate":8,"applicableAfterDays":15}`)}
	s.RefreshTimeMasters(penalty, interest)

	due := time.Now().UnixMilli() - 60*24*60*60*1000
	p, i := s.applyPenaltyAndInterest(200, due)
	if !approxEqual(p, 10) { // 200 * 5%
		t.Errorf("penalty after MDMS = %v, want 10", p)
	}
	if !approxEqual(i, 16) { // 200 * 8%
		t.Errorf("interest after MDMS = %v, want 16", i)
	}
}
