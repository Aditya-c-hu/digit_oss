package domain

// AuditDetails captures who/when created/modified a record.
type AuditDetails struct {
	CreatedBy        string `json:"createdBy,omitempty"`
	LastModifiedBy   string `json:"lastModifiedBy,omitempty"`
	CreatedTime      int64  `json:"createdTime,omitempty"`
	LastModifiedTime int64  `json:"lastModifiedTime,omitempty"`
}

type Role struct {
	Name string `json:"name,omitempty"`
	Code string `json:"code,omitempty"`
}

type User struct {
	UUID         string `json:"uuid,omitempty"`
	UserName     string `json:"userName,omitempty"`
	Type         string `json:"type,omitempty"`
	TenantID     string `json:"tenantId,omitempty"`
	MobileNumber string `json:"mobileNumber,omitempty"`
	Roles        []Role `json:"roles,omitempty"`
}

type RequestInfo struct {
	APIID     string `json:"apiId,omitempty"`
	Ver       string `json:"ver,omitempty"`
	TS        int64  `json:"ts,omitempty"`
	Action    string `json:"action,omitempty"`
	MsgID     string `json:"msgId,omitempty"`
	AuthToken string `json:"authToken,omitempty"`
	UserInfo  *User  `json:"userInfo,omitempty"`
}

type ResponseInfo struct {
	APIID  string `json:"apiId,omitempty"`
	Ver    string `json:"ver,omitempty"`
	TS     int64  `json:"ts,omitempty"`
	MsgID  string `json:"msgId,omitempty"`
	Status string `json:"status,omitempty"`
}

type RequestInfoWrapper struct {
	RequestInfo *RequestInfo `json:"RequestInfo,omitempty"`
}

// ErrorItem is one entry in the eGov error envelope.
type ErrorItem struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
}

// ErrorRes is the platform error envelope returned on failure.
type ErrorRes struct {
	ResponseInfo *ResponseInfo `json:"ResponseInfo,omitempty"`
	Errors       []ErrorItem   `json:"Errors"`
}

// Slab is a single tier of a billing slab.
type Slab struct {
	From        float64 `json:"from"`
	To          float64 `json:"to"`
	Charge      float64 `json:"charge"`
	MeterCharge float64 `json:"meterCharge"`
}

// BillingSlab is the master record loaded from MDMS describing the rate
// structure of a connection type / property type combo.
type BillingSlab struct {
	ID                   string  `json:"id,omitempty"`
	BuildingType         string  `json:"buildingType,omitempty"`
	ConnectionType       string  `json:"connectionType,omitempty"`
	CalculationAttribute string  `json:"calculationAttribute,omitempty"`
	MinimumCharge        float64 `json:"minimumCharge,omitempty"`
	Slabs                []Slab  `json:"slabs,omitempty"`
}

// TaxHeadEstimate is one charge / fee / rebate / penalty line in a calculation.
type TaxHeadEstimate struct {
	TaxHeadCode    string  `json:"taxHeadCode,omitempty"`
	EstimateAmount float64 `json:"estimateAmount,omitempty"`
	Category       string  `json:"category,omitempty"`
}

// CalculationCriteria is the input used to compute a single connection's bill.
type CalculationCriteria struct {
	ApplicationNo   string           `json:"applicationNo,omitempty"`
	ConnectionNo    string           `json:"connectionNo,omitempty"`
	TenantID        string           `json:"tenantId,omitempty"`
	From            int64            `json:"from,omitempty"`
	To              int64            `json:"to,omitempty"`
	WaterConnection *WaterConnection `json:"waterConnection,omitempty"`
}

// CalculationReq is the body of /_estimate and /_calculate.
type CalculationReq struct {
	RequestInfo             *RequestInfo          `json:"RequestInfo,omitempty"`
	CalculationCriteria     []CalculationCriteria `json:"CalculationCriteria,omitempty"`
	IsConnectionCalculation bool                  `json:"isconnectionCalculation,omitempty"`
}

// Calculation is the per-connection result we return.
type Calculation struct {
	ApplicationNO    string            `json:"applicationNo,omitempty"`
	ConnectionNo     string            `json:"connectionNo,omitempty"`
	TenantID         string            `json:"tenantId,omitempty"`
	TotalAmount      float64           `json:"totalAmount"`
	TaxAmount        float64           `json:"taxAmount"`
	Penalty          float64           `json:"penalty"`
	Exemption        float64           `json:"exemption"`
	Rebate           float64           `json:"rebate"`
	Charge           float64           `json:"charge"`
	Fee              float64           `json:"fee"`
	TaxHeadEstimates []TaxHeadEstimate `json:"taxHeadEstimates,omitempty"`
	BillingSlabIDs   []string          `json:"billingSlabIds,omitempty"`
	WaterConnection  *WaterConnection  `json:"waterConnection,omitempty"`
}

// CalculationRes is the response wrapper.
type CalculationRes struct {
	ResponseInfo *ResponseInfo `json:"ResponseInfo,omitempty"`
	Calculation  []Calculation `json:"Calculation"`
}

// WaterConnection is a thin form of the ws-services entity used for tax
// computation. We only carry what calculator needs.
type WaterConnection struct {
	ID                 string  `json:"id,omitempty"`
	TenantID           string  `json:"tenantId,omitempty"`
	ApplicationNo      string  `json:"applicationNo,omitempty"`
	ConnectionNo       string  `json:"connectionNo,omitempty"`
	ApplicationStatus  string  `json:"applicationStatus,omitempty"`
	Status             string  `json:"status,omitempty"`
	ConnectionType     string  `json:"connectionType,omitempty"`
	ConnectionCategory string  `json:"connectionCategory,omitempty"`
	NoOfTaps           int     `json:"noOfTaps,omitempty"`
	PipeSize           float64 `json:"pipeSize,omitempty"`
	WaterSource        string  `json:"waterSource,omitempty"`
	RoadType           string  `json:"roadType,omitempty"`
	RoadCuttingArea    float64 `json:"roadCuttingArea,omitempty"`
	// UsageCategory / LandArea come from the property in Java; callers may pass
	// them so the FeeSlab estimate can apply usage-type and plot-size charges.
	UsageCategory string  `json:"usageCategory,omitempty"`
	LandArea      float64 `json:"landArea,omitempty"`
}

// MeterReading is what's stored in eg_ws_meterreading.
type MeterReading struct {
	ID                 string        `json:"id,omitempty"`
	ConnectionNo       string        `json:"connectionNo,omitempty"`
	BillingPeriod      string        `json:"billingPeriod,omitempty"`
	MeterStatus        string        `json:"meterStatus,omitempty"`
	LastReading        float64       `json:"lastReading,omitempty"`
	LastReadingDate    int64         `json:"lastReadingDate,omitempty"`
	CurrentReading     float64       `json:"currentReading,omitempty"`
	CurrentReadingDate int64         `json:"currentReadingDate,omitempty"`
	Consumption        float64       `json:"consumption,omitempty"`
	TenantID           string        `json:"tenantId,omitempty"`
	AuditDetails       *AuditDetails `json:"auditDetails,omitempty"`
}

type MeterConnectionRequest struct {
	RequestInfo  *RequestInfo  `json:"RequestInfo,omitempty"`
	MeterReading *MeterReading `json:"MeterReading,omitempty"`
}

type MeterReadingResponse struct {
	ResponseInfo  *ResponseInfo  `json:"ResponseInfo,omitempty"`
	MeterReadings []MeterReading `json:"MeterReadings"`
}

type MeterReadingSearchCriteria struct {
	TenantID      string   `form:"tenantId" json:"tenantId,omitempty"`
	ConnectionNos []string `form:"connectionNos" json:"connectionNos,omitempty"`
	IDs           []string `form:"ids" json:"ids,omitempty"`
	Limit         int      `form:"limit" json:"limit,omitempty"`
	Offset        int      `form:"offset" json:"offset,omitempty"`
}

// Demand represents a billing-service demand at the calculator boundary.
type Demand struct {
	ID              string         `json:"id,omitempty"`
	TenantID        string         `json:"tenantId,omitempty"`
	ConsumerCode    string         `json:"consumerCode,omitempty"`
	BusinessService string         `json:"businessService,omitempty"`
	TaxPeriodFrom   int64          `json:"taxPeriodFrom,omitempty"`
	TaxPeriodTo     int64          `json:"taxPeriodTo,omitempty"`
	DemandDetails   []DemandDetail `json:"demandDetails,omitempty"`
	Status          string         `json:"status,omitempty"`
	AuditDetails    *AuditDetails  `json:"auditDetails,omitempty"`
}

type DemandDetail struct {
	ID                string  `json:"id,omitempty"`
	DemandID          string  `json:"demandId,omitempty"`
	TaxHeadMasterCode string  `json:"taxHeadMasterCode,omitempty"`
	TaxAmount         float64 `json:"taxAmount,omitempty"`
	CollectionAmount  float64 `json:"collectionAmount,omitempty"`
}

type DemandResponse struct {
	ResponseInfo *ResponseInfo `json:"ResponseInfo,omitempty"`
	Demands      []Demand      `json:"Demands"`
}

type GetBillCriteria struct {
	TenantID     string   `form:"tenantId" json:"tenantId,omitempty"`
	ConsumerCode []string `form:"consumerCodes" json:"consumerCode,omitempty"`
}

type BulkBillCriteria struct {
	TenantID      string   `json:"tenantId,omitempty"`
	BillCycle     string   `json:"billCycle,omitempty"`
	Locality      string   `json:"locality,omitempty"`
	ConsumerCodes []string `json:"consumerCodes,omitempty"`
}

type BulkBillReq struct {
	RequestInfo      *RequestInfo      `json:"RequestInfo,omitempty"`
	BulkBillCriteria *BulkBillCriteria `json:"bulkBillCriteria,omitempty"`
}

type AdhocTaxReq struct {
	RequestInfo   *RequestInfo `json:"RequestInfo,omitempty"`
	AdhocPenalty  float64      `json:"adhocPenalty,omitempty"`
	AdhocRebate   float64      `json:"adhocRebate,omitempty"`
	ConsumerCode  string       `json:"consumerCode,omitempty"`
	TenantID      string       `json:"tenantId,omitempty"`
	PenaltyReason string       `json:"adhocPenaltyReason,omitempty"`
	RebateReason  string       `json:"adhocRebateReason,omitempty"`
}
