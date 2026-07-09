package service

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/egov/ws-calculator/config"
	"github.com/egov/ws-calculator/internal/domain"
	"github.com/egov/ws-calculator/internal/repository/postgres"
	"github.com/egov/ws-calculator/internal/transport/kafka"
	"github.com/egov/ws-calculator/pkg/apperr"
	"github.com/google/uuid"
)

const (
	WSChargeHead         = "WS_CHARGE"
	WSTimePenalty        = "WS_TIME_PENALTY"
	WSTimeInterest       = "WS_TIME_INTEREST"
	WSRoundOff           = "WS_ROUND_OFF"
	WSAdhocPenalty       = "WS_ADHOC_PENALTY"
	WSAdhocRebate        = "WS_ADHOC_REBATE"
	WSConnectionFee      = "WS_ONE_TIME_FEE"
	WSConnectionRoundOff = "WS_ONE_TIME_FEE_ROUND_OFF"

	// One-time fee tax heads emitted by _estimate.
	WSFormFee           = "WS_FORM_FEE"
	WSScrutinyFee       = "WS_SCRUTINY_FEE"
	WSSecurityCharge    = "WS_SECURITY_CHARGE"
	WSRoadCuttingCharge = "WS_ROAD_CUTTING_CHARGE"
	WSMeterCharge       = "WS_METER_CHARGE"
	WSOtherCharge       = "WS_OTHER_CHARGE"
	WSUsageTypeFee      = "WS_ONE_TIME_FEE"
	WSTaxAndCess        = "WS_TAX_AND_CESS"
)

// CalculationService is the brain of ws-calculator. The Java estimator pulls
// MDMS billing slabs over HTTP per request; here we maintain a registered
// in-process slab table that the master-data loader can refresh on a goroutine.
type CalculationService struct {
	Repo     *postgres.CalculatorRepository
	Producer *kafka.Producer
	Cfg      *config.Config

	slabMu           sync.RWMutex
	slabs            []domain.BillingSlab
	defaultMinCharge float64

	// One-time fee amounts (seeded from config; MDMS fee masters can override).
	feeForm     float64
	feeScrutiny float64
	feeSecurity float64
	roadCutRate float64

	// Penalty / interest (seeded from config; MDMS Penalty/Interest masters
	// override via RefreshTimeMasters). Rate is a percentage of the water charge.
	penaltyRate  float64
	penaltyFlat  float64
	penaltyDays  float64
	interestRate float64
	interestFlat float64
	interestDays float64

	// MDMS FeeSlab estimate config (EstimationService parity). When loaded, the
	// estimate uses these instead of the flat config fees.
	fee feeConfig
}

// feeConfig holds the parsed MDMS fee masters used by _estimate.
type feeConfig struct {
	loaded      bool
	formFee     float64
	scrutinyFee float64
	otherCharge float64
	meterCost   float64
	taxPct      float64
	roadUnit    map[string]float64 // RoadType code -> unitCost
	usageUnit   map[string]float64 // PropertyUsageType code -> unitCost
	plotSlabs   []plotSlab         // PlotSizeSlab from/to/unitCost
}

type plotSlab struct {
	From     float64 `json:"from"`
	To       float64 `json:"to"`
	UnitCost float64 `json:"unitCost"`
}

func NewCalculationService(r *postgres.CalculatorRepository, p *kafka.Producer, cfg *config.Config) *CalculationService {
	s := &CalculationService{
		Repo: r, Producer: p, Cfg: cfg, defaultMinCharge: 100.0,
		feeForm:      cfg.FormFee,
		feeScrutiny:  cfg.ScrutinyFee,
		feeSecurity:  cfg.SecurityCharge,
		roadCutRate:  cfg.RoadCuttingRate,
		penaltyRate:  cfg.PenaltyRate,
		penaltyFlat:  cfg.PenaltyFlat,
		penaltyDays:  cfg.PenaltyApplicableDays,
		interestRate: cfg.InterestRate,
		interestFlat: cfg.InterestFlat,
		interestDays: cfg.InterestApplicableDays,
	}
	s.seedDefaultSlabs()
	return s
}

// seedDefaultSlabs preloads representative MDMS slabs (Domestic / NonMetered,
// Domestic / Metered, NonDomestic / Metered) so /_estimate works before the
// MDMS loader ever runs. In production this is overwritten by RefreshSlabs.
func (s *CalculationService) seedDefaultSlabs() {
	s.slabs = []domain.BillingSlab{
		{
			ID: "default-domestic-non", BuildingType: "RESIDENTIAL", ConnectionType: "Non_Metered",
			CalculationAttribute: "Flat", MinimumCharge: 100,
			Slabs: []domain.Slab{{From: 0, To: 99999, Charge: 100, MeterCharge: 0}},
		},
		{
			ID: "default-domestic-met", BuildingType: "RESIDENTIAL", ConnectionType: "Metered",
			CalculationAttribute: "Volumetric", MinimumCharge: 50,
			Slabs: []domain.Slab{
				{From: 0, To: 10, Charge: 5},
				{From: 10, To: 20, Charge: 10},
				{From: 20, To: 1000000, Charge: 15},
			},
		},
		{
			ID: "default-non-domestic", BuildingType: "NON_RESIDENTIAL", ConnectionType: "Metered",
			CalculationAttribute: "Volumetric", MinimumCharge: 200,
			Slabs: []domain.Slab{
				{From: 0, To: 50, Charge: 25},
				{From: 50, To: 1000000, Charge: 40},
			},
		},
	}
}

// RefreshSlabs is the hook the master-data loader goroutine calls when it
// finishes pulling fresh MDMS data. Concurrent reads from /_estimate continue
// uninterrupted thanks to the RWMutex.
func (s *CalculationService) RefreshSlabs(slabs []domain.BillingSlab) {
	s.slabMu.Lock()
	defer s.slabMu.Unlock()
	s.slabs = slabs
}

// Estimate computes the one-time application/connection fees for a brand-new
// application (POST /waterCalculator/_estimate). This is the form/scrutiny/
// security/road-cutting fee preview — NOT the periodic water charge, which is
// produced by Calculate. Mirrors the Java EstimationService fee path.
func (s *CalculationService) Estimate(ctx context.Context, req *domain.CalculationReq) []domain.Calculation {
	out := make([]domain.Calculation, 0, len(req.CalculationCriteria))
	for _, c := range req.CalculationCriteria {
		out = append(out, s.estimateOne(c))
	}
	return out
}

// estimateOne builds the one-time fee tax heads for a draft application. When
// MDMS FeeSlab data is loaded it follows EstimationService; otherwise it uses
// the flat config fees (local/offline).
func (s *CalculationService) estimateOne(c domain.CalculationCriteria) domain.Calculation {
	s.slabMu.RLock()
	fee := s.fee
	s.slabMu.RUnlock()
	if fee.loaded {
		return s.estimateFromFeeSlab(c, fee)
	}
	return s.estimateFromConfig(c)
}

// estimateFromFeeSlab mirrors EstimationService.getTaxHeadForFeeEstimation.
func (s *CalculationService) estimateFromFeeSlab(c domain.CalculationCriteria, fee feeConfig) domain.Calculation {
	calc := domain.Calculation{ApplicationNO: c.ApplicationNo, ConnectionNo: c.ConnectionNo, TenantID: c.TenantID}
	wc := c.WaterConnection
	if wc == nil {
		wc = &domain.WaterConnection{}
	}

	meterCost := 0.0
	if isMetered(wc.ConnectionType) {
		meterCost = fee.meterCost
	}
	roadCut, usage := 0.0, 0.0
	if wc.RoadCuttingArea > 0 {
		if u, ok := fee.roadUnit[wc.RoadType]; ok {
			roadCut = u * wc.RoadCuttingArea
		}
		if u, ok := fee.usageUnit[wc.UsageCategory]; ok {
			usage = u * wc.RoadCuttingArea
		}
	}
	plot := 0.0
	if wc.LandArea > 0 {
		plot = plotFee(fee.plotSlabs, wc.LandArea)
	}
	total := fee.formFee + fee.scrutinyFee + fee.otherCharge + meterCost + roadCut + plot + usage
	tax := total * fee.taxPct / 100.0

	estimates := []domain.TaxHeadEstimate{}
	add := func(code string, amt float64) {
		if amt != 0 {
			estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: code, EstimateAmount: round2(amt), Category: "CHARGES"})
		}
	}
	add(WSFormFee, fee.formFee)
	add(WSScrutinyFee, fee.scrutinyFee)
	add(WSMeterCharge, meterCost)
	add(WSOtherCharge, fee.otherCharge)
	add(WSRoadCuttingCharge, roadCut)
	add(WSUsageTypeFee, usage)
	add(WSSecurityCharge, plot)
	add(WSTaxAndCess, tax)

	calc.TaxHeadEstimates = estimates
	for _, e := range estimates {
		calc.TotalAmount += e.EstimateAmount
		calc.TaxAmount += e.EstimateAmount
	}
	calc.TotalAmount = round2(calc.TotalAmount)
	return calc
}

// estimateFromConfig is the flat-fee fallback used when MDMS is unavailable.
func (s *CalculationService) estimateFromConfig(c domain.CalculationCriteria) domain.Calculation {
	calc := domain.Calculation{
		ApplicationNO: c.ApplicationNo,
		ConnectionNo:  c.ConnectionNo,
		TenantID:      c.TenantID,
	}
	estimates := []domain.TaxHeadEstimate{}
	if s.feeForm > 0 {
		estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSFormFee, EstimateAmount: round2(s.feeForm), Category: "CHARGES"})
	}
	if s.feeScrutiny > 0 {
		estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSScrutinyFee, EstimateAmount: round2(s.feeScrutiny), Category: "CHARGES"})
	}
	if s.feeSecurity > 0 {
		estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSSecurityCharge, EstimateAmount: round2(s.feeSecurity), Category: "CHARGES"})
	}
	if c.WaterConnection != nil && c.WaterConnection.RoadCuttingArea > 0 && s.roadCutRate > 0 {
		rc := c.WaterConnection.RoadCuttingArea * s.roadCutRate
		estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSRoadCuttingCharge, EstimateAmount: round2(rc), Category: "CHARGES"})
	}

	total := 0.0
	for _, e := range estimates {
		total += e.EstimateAmount
	}
	if rOff := roundOff(total); rOff != 0 {
		estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSConnectionRoundOff, EstimateAmount: rOff})
	}

	calc.TaxHeadEstimates = estimates
	for _, e := range estimates {
		calc.TotalAmount += e.EstimateAmount
		calc.TaxAmount += e.EstimateAmount
	}
	calc.TotalAmount = round2(calc.TotalAmount)
	return calc
}

// plotFee returns the PlotSizeSlab unitCost whose [from,to) contains plotSize.
func plotFee(slabs []plotSlab, plotSize float64) float64 {
	for _, p := range slabs {
		if plotSize >= p.From && plotSize < p.To {
			return p.UnitCost
		}
	}
	return 0
}

// RefreshFeeMasters parses the MDMS FeeSlab / RoadType / PlotSizeSlab /
// PropertyUsageType masters into the estimate fee config. Mirrors the masters
// EstimationService reads. FeeSlab uses the first entry, like Java.
func (s *CalculationService) RefreshFeeMasters(feeSlab, roadType, plotSlabRows, usageType []json.RawMessage) {
	type feeSlabRow struct {
		FormFee     *float64 `json:"formFee"`
		ScrutinyFee *float64 `json:"scrutinyFee"`
		Other       *float64 `json:"other"`
		TaxPct      *float64 `json:"taxpercentage"`
		MeterCost   *float64 `json:"meterCost"`
	}
	type codeUnit struct {
		Code     string  `json:"code"`
		UnitCost float64 `json:"unitCost"`
	}

	f := feeConfig{roadUnit: map[string]float64{}, usageUnit: map[string]float64{}}
	if len(feeSlab) > 0 {
		var row feeSlabRow
		if json.Unmarshal(feeSlab[0], &row) == nil {
			f.formFee = derefF(row.FormFee)
			f.scrutinyFee = derefF(row.ScrutinyFee)
			f.otherCharge = derefF(row.Other)
			f.taxPct = derefF(row.TaxPct)
			f.meterCost = derefF(row.MeterCost)
		}
	}
	for _, raw := range roadType {
		var cu codeUnit
		if json.Unmarshal(raw, &cu) == nil && cu.Code != "" {
			f.roadUnit[cu.Code] = cu.UnitCost
		}
	}
	for _, raw := range usageType {
		var cu codeUnit
		if json.Unmarshal(raw, &cu) == nil && cu.Code != "" {
			f.usageUnit[cu.Code] = cu.UnitCost
		}
	}
	for _, raw := range plotSlabRows {
		var p plotSlab
		if json.Unmarshal(raw, &p) == nil {
			f.plotSlabs = append(f.plotSlabs, p)
		}
	}
	f.loaded = true

	s.slabMu.Lock()
	s.fee = f
	s.slabMu.Unlock()
}

func derefF(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// Calculate runs full per-billing-cycle calculation, persists nothing in this
// path (demand persistence is the caller's job) but does publish the result on
// the demand-save topic for the billing service.
func (s *CalculationService) Calculate(ctx context.Context, req *domain.CalculationReq) []domain.Calculation {
	out := make([]domain.Calculation, 0, len(req.CalculationCriteria))
	for _, c := range req.CalculationCriteria {
		calc := s.calculateOne(ctx, c)
		out = append(out, calc)
	}
	if s.Producer != nil {
		_ = s.Producer.Push(ctx, s.Cfg.DemandSaveTopic, map[string]any{
			"RequestInfo": req.RequestInfo,
			"Calculation": out,
		})
	}
	return out
}

func (s *CalculationService) calculateOne(ctx context.Context, c domain.CalculationCriteria) domain.Calculation {
	calc := domain.Calculation{
		ApplicationNO: c.ApplicationNo,
		ConnectionNo:  c.ConnectionNo,
		TenantID:      c.TenantID,
	}
	wc := c.WaterConnection
	if wc == nil {
		wc = &domain.WaterConnection{ConnectionType: "Non_Metered"}
	}

	slab := s.findSlab(wc.ConnectionCategory, wc.ConnectionType)
	billingSlabIDs := []string{}
	if slab != nil {
		billingSlabIDs = append(billingSlabIDs, slab.ID)
	}

	// Compute base water charge. Mirrors EstimationService.getWaterEstimationCharge:
	// Flat calculationAttribute -> minimum charge; otherwise a range calculation
	// over the slab's `charge` field (metered = cumulative tiers on consumption,
	// non-metered = single matching tier on the unit of measurement), floored at
	// the slab's minimum charge.
	var charge float64
	switch {
	case slab == nil:
		charge = s.defaultMinCharge
	case isFlatAttribute(slab.CalculationAttribute):
		charge = slab.MinimumCharge
	default:
		metered := isMetered(wc.ConnectionType)
		uom := s.unitOfMeasurement(ctx, c, wc, metered)
		charge = computeRangeCharge(uom, slab.Slabs, metered)
		if charge < slab.MinimumCharge {
			charge = slab.MinimumCharge
		}
	}

	estimates := []domain.TaxHeadEstimate{
		{TaxHeadCode: WSChargeHead, EstimateAmount: round2(charge), Category: "CHARGES"},
	}

	// Penalty / interest only after billing expiry, mirroring PayService logic.
	if c.To > 0 {
		penalty, interest := s.applyPenaltyAndInterest(charge, c.To)
		if penalty > 0 {
			estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSTimePenalty, EstimateAmount: round2(penalty), Category: "PENALTY"})
		}
		if interest > 0 {
			estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSTimeInterest, EstimateAmount: round2(interest), Category: "INTEREST"})
		}
	}

	// Round-off applied to the total: mirrors PayService.roundOfDecimals.
	total := 0.0
	for _, e := range estimates {
		total += e.EstimateAmount
	}
	if rOff := roundOff(total); rOff != 0 {
		estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSRoundOff, EstimateAmount: rOff})
	}

	calc.TaxHeadEstimates = estimates
	calc.BillingSlabIDs = billingSlabIDs
	calc.Charge = round2(charge)

	for _, e := range estimates {
		calc.TotalAmount += e.EstimateAmount
		switch e.Category {
		case "PENALTY":
			calc.Penalty += e.EstimateAmount
		case "REBATE":
			calc.Rebate += e.EstimateAmount
		default:
			calc.TaxAmount += e.EstimateAmount
		}
	}
	calc.TotalAmount = round2(calc.TotalAmount)
	return calc
}

func (s *CalculationService) findSlab(building, connType string) *domain.BillingSlab {
	s.slabMu.RLock()
	defer s.slabMu.RUnlock()
	for i := range s.slabs {
		if s.slabs[i].BuildingType == building && s.slabs[i].ConnectionType == connType {
			return &s.slabs[i]
		}
	}
	for i := range s.slabs {
		if s.slabs[i].ConnectionType == connType {
			return &s.slabs[i]
		}
	}
	if len(s.slabs) > 0 {
		return &s.slabs[0]
	}
	return nil
}

func (s *CalculationService) consumptionForConnection(ctx context.Context, tenant, connNo string) float64 {
	if connNo == "" || tenant == "" {
		return 0
	}
	mr, err := s.Repo.LatestReading(ctx, tenant, connNo)
	if err != nil || mr == nil {
		return 0
	}
	if mr.Consumption > 0 {
		return mr.Consumption
	}
	return mr.CurrentReading - mr.LastReading
}

// applyPenaltyAndInterest mirrors PayService.applyPenaltyRebateAndInterest:
// once the connection is overdue beyond applicableAfterDays, a flat rate% of the
// water charge (or a flat amount) is applied — no day proration. Rates come from
// the MDMS Penalty/Interest masters when enabled, else from config.
func (s *CalculationService) applyPenaltyAndInterest(waterCharge float64, dueDateMillis int64) (penalty, interest float64) {
	if waterCharge <= 0 {
		return 0, 0
	}
	s.slabMu.RLock()
	pRate, pFlat, pDays := s.penaltyRate, s.penaltyFlat, s.penaltyDays
	iRate, iFlat, iDays := s.interestRate, s.interestFlat, s.interestDays
	s.slabMu.RUnlock()

	now := time.Now().UnixMilli()
	noOfDays := math.Floor(math.Abs(float64(dueDateMillis-now)) / (1000 * 60 * 60 * 24))
	if noOfDays >= 1 { // matches Java: noOfDays = noOfDays + 1 once >= 1
		noOfDays++
	}
	penalty = round2(applicableCharge(waterCharge, noOfDays, pRate, pFlat, pDays))
	interest = round2(applicableCharge(waterCharge, noOfDays, iRate, iFlat, iDays))
	return
}

// applicableCharge mirrors PayService.getApplicablePenalty/getApplicableInterest:
// zero until noOfDays exceeds applicableDays, then rate% of the charge; if no
// rate is set, a flat amount (ignored when it exceeds the charge).
func applicableCharge(waterCharge, noOfDays, rate, flat, applicableDays float64) float64 {
	if noOfDays-applicableDays < 1 {
		return 0
	}
	if rate > 0 {
		return waterCharge * rate / 100.0
	}
	if flat > waterCharge {
		return 0
	}
	return flat
}

// timeMasterEntry is the relevant slice of an MDMS Penalty/Interest master row.
type timeMasterEntry struct {
	Rate                *float64 `json:"rate"`
	FlatAmount          *float64 `json:"flatAmount"`
	ApplicableAfterDays *float64 `json:"applicableAfterDays"`
}

// RefreshTimeMasters overrides penalty/interest config from the MDMS Penalty and
// Interest masters. Java selects the entry applicable to the assessment year;
// here we take the first entry (masters are typically single-entry per tenant).
func (s *CalculationService) RefreshTimeMasters(penalty, interest []json.RawMessage) {
	s.slabMu.Lock()
	defer s.slabMu.Unlock()
	if e := firstTimeMaster(penalty); e != nil {
		s.penaltyRate, s.penaltyFlat, s.penaltyDays = applyTimeMaster(e, s.penaltyRate, s.penaltyFlat, s.penaltyDays)
	}
	if e := firstTimeMaster(interest); e != nil {
		s.interestRate, s.interestFlat, s.interestDays = applyTimeMaster(e, s.interestRate, s.interestFlat, s.interestDays)
	}
}

func firstTimeMaster(rows []json.RawMessage) *timeMasterEntry {
	for _, raw := range rows {
		var e timeMasterEntry
		if err := json.Unmarshal(raw, &e); err == nil {
			return &e
		}
	}
	return nil
}

func applyTimeMaster(e *timeMasterEntry, rate, flat, days float64) (float64, float64, float64) {
	if e.Rate != nil {
		rate = *e.Rate
	} else {
		rate = 0 // flat-amount mode
	}
	if e.FlatAmount != nil {
		flat = *e.FlatAmount
	}
	if e.ApplicableAfterDays != nil {
		days = *e.ApplicableAfterDays
	}
	return rate, flat, days
}

// ApplyAdhocTax constructs penalty / rebate tax-head estimates on top of an
// existing demand. Real impl would persist back to billing-service; we return
// the calc so the controller can surface it.
func (s *CalculationService) ApplyAdhocTax(ctx context.Context, req *domain.AdhocTaxReq) []domain.Calculation {
	if req == nil || req.TenantID == "" {
		return nil
	}
	estimates := []domain.TaxHeadEstimate{}
	if req.AdhocPenalty > 0 {
		estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSAdhocPenalty, EstimateAmount: round2(req.AdhocPenalty), Category: "PENALTY"})
	}
	if req.AdhocRebate > 0 {
		estimates = append(estimates, domain.TaxHeadEstimate{TaxHeadCode: WSAdhocRebate, EstimateAmount: round2(-req.AdhocRebate), Category: "REBATE"})
	}
	calc := domain.Calculation{
		ApplicationNO:    req.ConsumerCode,
		ConnectionNo:     req.ConsumerCode,
		TenantID:         req.TenantID,
		TaxHeadEstimates: estimates,
	}
	for _, e := range estimates {
		calc.TotalAmount += e.EstimateAmount
		switch e.Category {
		case "PENALTY":
			calc.Penalty += e.EstimateAmount
		case "REBATE":
			calc.Rebate += e.EstimateAmount
		default:
			calc.TaxAmount += e.EstimateAmount
		}
	}
	calc.TotalAmount = round2(calc.TotalAmount)
	return []domain.Calculation{calc}
}

// GenerateDemandsForCycle is the workflow target of POST /waterCalculator/_jobscheduler.
// It fans calculations across goroutines, bounded by a worker pool, then publishes
// each demand on the bill-gen topic. This is the Go analogue of the Java
// BulkDemandAndBillGenService that uses a ThreadPoolExecutor.
func (s *CalculationService) GenerateDemandsForCycle(ctx context.Context, req *domain.RequestInfo, criteria *domain.BulkBillCriteria) error {
	if criteria == nil || criteria.TenantID == "" {
		return apperr.BadRequest("EG_WS_TENANTID_REQUIRED", "tenantId is required")
	}
	jobs := make(chan string)
	results := make(chan domain.Calculation)
	const workers = 8
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for cn := range jobs {
				calc := s.calculateOne(ctx, domain.CalculationCriteria{
					ConnectionNo:    cn,
					TenantID:        criteria.TenantID,
					From:            time.Now().AddDate(0, -1, 0).UnixMilli(),
					To:              time.Now().UnixMilli(),
					WaterConnection: &domain.WaterConnection{ConnectionType: "Metered", ConnectionCategory: "RESIDENTIAL"},
				})
				results <- calc
			}
		}()
	}
	go func() {
		for _, cn := range criteria.ConsumerCodes {
			jobs <- cn
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	for r := range results {
		if s.Producer != nil {
			_ = s.Producer.Push(ctx, s.Cfg.BillGenTopic, map[string]any{
				"RequestInfo": req,
				"Calculation": r,
				"id":          uuid.NewString(),
			})
		}
	}
	return nil
}

// computeRangeCharge mirrors EstimationService.getWaterEstimationCharge range
// calculation, using the slab `charge` field (NOT meterCharge):
//   - metered: cumulative tiers across the unit-of-measurement;
//   - non-metered: the single tier whose [from,to) contains the UOM, charged as
//     uom * charge.
func computeRangeCharge(uom float64, slabs []domain.Slab, metered bool) float64 {
	if uom <= 0 || len(slabs) == 0 {
		return 0
	}
	if metered {
		total := 0.0
		remaining := uom
		for _, s := range slabs {
			if remaining > s.To {
				total += (s.To - s.From) * s.Charge
				remaining -= (s.To - s.From)
			} else if remaining < s.To {
				total += remaining * s.Charge
				break
			}
		}
		return total
	}
	for _, s := range slabs {
		if uom >= s.From && uom < s.To {
			return uom * s.Charge
		}
	}
	return 0
}

// unitOfMeasurement returns the UOM the range calculation runs over: metered
// connections use the latest meter consumption; non-metered use the tap count
// (the common DIGIT "No. of taps" attribute).
func (s *CalculationService) unitOfMeasurement(ctx context.Context, c domain.CalculationCriteria, wc *domain.WaterConnection, metered bool) float64 {
	if metered {
		return s.consumptionForConnection(ctx, c.TenantID, c.ConnectionNo)
	}
	return float64(wc.NoOfTaps)
}

func isMetered(t string) bool {
	return t == "Metered"
}

func isFlatAttribute(attr string) bool {
	return strings.EqualFold(attr, "Flat") || attr == ""
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// roundOff returns the adjustment amount to nudge a total to its nearest rupee.
// Mirrors PayService.roundOfDecimals.
func roundOff(total float64) float64 {
	rem := total - math.Floor(total)
	if rem >= 0.5 {
		return round2(1 - rem)
	}
	if rem > 0 {
		return round2(-rem)
	}
	return 0
}
