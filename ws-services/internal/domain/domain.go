package domain

// AuditDetails captures who/when created/modified a record.
type AuditDetails struct {
	CreatedBy        string `json:"createdBy,omitempty"`
	LastModifiedBy   string `json:"lastModifiedBy,omitempty"`
	CreatedTime      int64  `json:"createdTime,omitempty"`
	LastModifiedTime int64  `json:"lastModifiedTime,omitempty"`
}

// RequestInfo mirrors the eGov platform RequestInfo header that travels with every request.
type RequestInfo struct {
	APIID         string `json:"apiId,omitempty"`
	Ver           string `json:"ver,omitempty"`
	TS            int64  `json:"ts,omitempty"`
	Action        string `json:"action,omitempty"`
	Did           string `json:"did,omitempty"`
	Key           string `json:"key,omitempty"`
	MsgID         string `json:"msgId,omitempty"`
	AuthToken     string `json:"authToken,omitempty"`
	UserInfo      *User  `json:"userInfo,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

// ResponseInfo is returned with every response.
type ResponseInfo struct {
	APIID    string `json:"apiId,omitempty"`
	Ver      string `json:"ver,omitempty"`
	TS       int64  `json:"ts,omitempty"`
	ResMsgID string `json:"resMsgId,omitempty"`
	MsgID    string `json:"msgId,omitempty"`
	Status   string `json:"status,omitempty"`
}

type Role struct {
	Name        string `json:"name,omitempty"`
	Code        string `json:"code,omitempty"`
	TenantID    string `json:"tenantId,omitempty"`
	Description string `json:"description,omitempty"`
}

// User as carried by RequestInfo.
type User struct {
	ID           int64  `json:"id,omitempty"`
	UUID         string `json:"uuid,omitempty"`
	UserName     string `json:"userName,omitempty"`
	Name         string `json:"name,omitempty"`
	MobileNumber string `json:"mobileNumber,omitempty"`
	EmailID      string `json:"emailId,omitempty"`
	Type         string `json:"type,omitempty"`
	TenantID     string `json:"tenantId,omitempty"`
	Roles        []Role `json:"roles,omitempty"`
}

// RequestInfoWrapper is used as a body wrapper for search calls.
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

// Document attached to a connection application.
type Document struct {
	ID                string `json:"id,omitempty"`
	DocumentType      string `json:"documentType,omitempty"`
	FileStoreID       string `json:"fileStoreId,omitempty"`
	DocumentUID       string `json:"documentUid,omitempty"`
	Status            string `json:"status,omitempty"`
	AdditionalDetails any    `json:"additionalDetails,omitempty"`
}

// PlumberInfo is the assigned plumber for the connection.
type PlumberInfo struct {
	ID                    string `json:"id,omitempty"`
	Name                  string `json:"name,omitempty"`
	LicenseNo             string `json:"licenseNo,omitempty"`
	MobileNumber          string `json:"mobileNumber,omitempty"`
	Gender                string `json:"gender,omitempty"`
	FatherOrHusbandName   string `json:"fatherOrHusbandName,omitempty"`
	CorrespondenceAddress string `json:"correspondenceAddress,omitempty"`
	Relationship          string `json:"relationship,omitempty"`
	RelationShip          string `json:"-"`
}

type RoadCuttingInfo struct {
	ID              string  `json:"id,omitempty"`
	RoadType        string  `json:"roadType,omitempty"`
	RoadCuttingArea float64 `json:"roadCuttingArea,omitempty"`
	Active          bool    `json:"active,omitempty"`
}

type OwnerInfo struct {
	UUID                 string  `json:"uuid,omitempty"`
	Status               string  `json:"status,omitempty"`
	IsPrimaryHolder      bool    `json:"isPrimaryHolder,omitempty"`
	ConnectionHolderType string  `json:"connectionHolderType,omitempty"`
	HolderShipPercentage float64 `json:"holdershipPercentage,omitempty"`
	Relationship         string  `json:"relationship,omitempty"`
	Name                 string  `json:"name,omitempty"`
	MobileNumber         string  `json:"mobileNumber,omitempty"`
	EmailID              string  `json:"emailId,omitempty"`
	UserName             string  `json:"userName,omitempty"`
}

// Connection is the parent record carried by every WaterConnection.
type Connection struct {
	ID                         string            `json:"id,omitempty"`
	TenantID                   string            `json:"tenantId,omitempty"`
	PropertyID                 string            `json:"propertyId,omitempty"`
	ApplicationNo              string            `json:"applicationNo,omitempty"`
	ApplicationStatus          string            `json:"applicationStatus,omitempty"`
	Status                     string            `json:"status,omitempty"`
	ConnectionNo               string            `json:"connectionNo,omitempty"`
	OldConnectionNo            string            `json:"oldConnectionNo,omitempty"`
	Documents                  []Document        `json:"documents,omitempty"`
	PlumberInfo                []PlumberInfo     `json:"plumberInfo,omitempty"`
	RoadType                   string            `json:"roadType,omitempty"`
	RoadCuttingArea            float64           `json:"roadCuttingArea,omitempty"`
	RoadCuttingInfo            []RoadCuttingInfo `json:"roadCuttingInfo,omitempty"`
	ConnectionExecutionDate    int64             `json:"connectionExecutionDate,omitempty"`
	ConnectionCategory         string            `json:"connectionCategory,omitempty"`
	ConnectionType             string            `json:"connectionType,omitempty"`
	AdditionalDetails          any               `json:"additionalDetails,omitempty"`
	AuditDetails               *AuditDetails     `json:"auditDetails,omitempty"`
	ProcessInstance            *ProcessInstance  `json:"processInstance,omitempty"`
	ApplicationType            string            `json:"applicationType,omitempty"`
	DateEffectiveFrom          int64             `json:"dateEffectiveFrom,omitempty"`
	ConnectionHolders          []OwnerInfo       `json:"connectionHolders,omitempty"`
	OldApplication             bool              `json:"oldApplication,omitempty"`
	Channel                    string            `json:"channel,omitempty"`
	DisconnectionExecutionDate int64             `json:"disconnectionExecutionDate,omitempty"`
	Action                     string            `json:"action,omitempty"`
	Locality                   string            `json:"locality,omitempty"`
}

// WaterConnection extends Connection with water-specific fields.
type WaterConnection struct {
	Connection
	WaterSource              string  `json:"waterSource,omitempty"`
	MeterID                  string  `json:"meterId,omitempty"`
	MeterInstallationDate    int64   `json:"meterInstallationDate,omitempty"`
	ProposedPipeSize         float64 `json:"proposedPipeSize,omitempty"`
	ProposedTaps             int     `json:"proposedTaps,omitempty"`
	PipeSize                 float64 `json:"pipeSize,omitempty"`
	NoOfTaps                 int     `json:"noOfTaps,omitempty"`
	IsDisconnectionTemporary bool    `json:"isDisconnectionTemporary,omitempty"`
	DisconnectionReason      string  `json:"disconnectionReason,omitempty"`
}

// WaterConnectionRequest is the body for create / update endpoints.
type WaterConnectionRequest struct {
	RequestInfo     *RequestInfo     `json:"RequestInfo,omitempty"`
	WaterConnection *WaterConnection `json:"WaterConnection,omitempty"`
	CreateCall      bool             `json:"-"`
}

// WaterConnectionResponse carries the connection list back.
type WaterConnectionResponse struct {
	ResponseInfo    *ResponseInfo     `json:"ResponseInfo,omitempty"`
	WaterConnection []WaterConnection `json:"WaterConnection"`
	TotalCount      int               `json:"totalCount,omitempty"`
}

// SearchCriteria mirrors the search query parameters of the Java service.
type SearchCriteria struct {
	TenantID                  string   `form:"tenantId" json:"tenantId,omitempty"`
	IDs                       []string `form:"ids" json:"ids,omitempty"`
	ApplicationNumber         []string `form:"applicationNumber" json:"applicationNumber,omitempty"`
	ApplicationStatus         []string `form:"applicationStatus" json:"applicationStatus,omitempty"`
	ConnectionNumber          []string `form:"connectionNumber" json:"connectionNumber,omitempty"`
	OldConnectionNumber       string   `form:"oldConnectionNumber" json:"oldConnectionNumber,omitempty"`
	MobileNumber              string   `form:"mobileNumber" json:"mobileNumber,omitempty"`
	PropertyID                string   `form:"propertyId" json:"propertyId,omitempty"`
	Status                    string   `form:"status" json:"status,omitempty"`
	FromDate                  int64    `form:"fromDate" json:"fromDate,omitempty"`
	ToDate                    int64    `form:"toDate" json:"toDate,omitempty"`
	Offset                    int      `form:"offset" json:"offset,omitempty"`
	Limit                     int      `form:"limit" json:"limit,omitempty"`
	ApplicationType           string   `form:"applicationType" json:"applicationType,omitempty"`
	Locality                  string   `form:"locality" json:"locality,omitempty"`
	IsPropertyDetailsRequired bool     `form:"isPropertyDetailsRequired" json:"isPropertyDetailsRequired,omitempty"`
	SearchType                string   `form:"searchType" json:"searchType,omitempty"`
	DoorNo                    string   `form:"doorNo" json:"doorNo,omitempty"`
	OwnerName                 string   `form:"ownerName" json:"ownerName,omitempty"`
	Assignee                  string   `form:"assignee" json:"assignee,omitempty"`
	IsCountCall               bool     `form:"-" json:"-"`
}

// ProcessInstance is a thin representation of the workflow process state.
type ProcessInstance struct {
	ID              string `json:"id,omitempty"`
	BusinessID      string `json:"businessId,omitempty"`
	Action          string `json:"action,omitempty"`
	ModuleName      string `json:"moduleName,omitempty"`
	TenantID        string `json:"tenantId,omitempty"`
	BusinessService string `json:"businessService,omitempty"`
	Comment         string `json:"comment,omitempty"`
	State           *State `json:"state,omitempty"`
}

type State struct {
	ApplicationStatus string `json:"applicationStatus,omitempty"`
	State             string `json:"state,omitempty"`
	IsTerminateState  bool   `json:"isTerminateState,omitempty"`
}
