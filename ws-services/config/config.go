package config

import (
	"bufio"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort   string
	ContextPath  string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBSSLMode    string
	KafkaBrokers []string
	KafkaGroupID string

	// Pagination
	DefaultLimit  int
	DefaultOffset int
	MaxLimit      int

	// IDGen
	IsIDGenEnabled bool
	IDGenHost      string
	IDGenPath      string
	WCIDName       string
	WCIDFormat     string
	WCAPIDName     string
	WCAPIDFormat   string
	WDCIDName      string
	WDCIDFormat    string

	// Workflow
	IsExternalWorkflowEnabled     bool
	BusinessServiceValue          string
	ModifyWSBusinessServiceName   string
	DisconnectBusinessServiceName string
	WfHost                        string
	WfTransitionPath              string
	WfBusinessServiceSearchPath   string
	WfProcessSearchPath           string

	// Persistence mode: when true, egov-persister owns the DB insert/update via
	// the save/update topics and the service does not write directly (avoids
	// double persistence). When false (local/offline), the service writes the DB.
	IsPersisterEnabled bool

	// Topics
	OnWaterSavedTopic         string
	OnWaterUpdatedTopic       string
	WorkflowUpdateTopic       string
	CreateMeterReadingTopic   string
	EditNotificationTopic     string
	FileStoreIdsTopic         string
	SaveFileStoreIdsTopic     string
	ReceiptBusinessTopic      string
	ReceiptDisconnectionTopic string
	SmsNotifTopic             string
	EmailNotifTopic           string
	SaveUserEventsTopic       string
	DocumentAuditTopic        string

	// External services
	IsUserEnabled       bool
	UserHost            string
	UserSearchPath      string
	UserCreatePath      string
	UserUpdatePath      string
	IsPropertyEnabled   bool
	PropertyHost        string
	PropertySearchPath  string
	IsEncryptionEnabled bool
	EncHost             string
	EncEncryptPath      string
	EncDecryptPath      string
	IsMDMSEnabled       bool
	MdmsHost            string
	MdmsURL             string
	CalculatorHost      string
	CalculateEndpoint   string
	EstimationEndpoint  string
	BillingServiceHost  string
	FetchBillEndpoint   string
	CollectionHost      string
	PaymentSearchPath   string

	// Notification
	IsSMSEnabled        bool
	IsEmailEnabled      bool
	IsUserEventsEnabled bool

	StateLevelTenantID string
}

// Load reads configuration from (in precedence order): environment variables,
// an application.properties file, then code defaults. Replaces the previous
// viper dependency with the standard library to minimise the Black Duck
// dependency surface.
func Load() (*Config, error) {
	s := newSettings([]string{
		"application.properties",
		"config/application.properties",
		"/etc/ws-services/application.properties",
	})

	db := resolveDB(s, "rainmaker_new")
	cfg := &Config{
		ServerPort:                    s.get("server.port", "8090"),
		ContextPath:                   s.get("server.context-path", "/ws-services"),
		DBHost:                        db.host,
		DBPort:                        db.port,
		DBUser:                        db.user,
		DBPassword:                    db.pass,
		DBName:                        db.name,
		DBSSLMode:                     db.ssl,
		KafkaBrokers:                  resolveBrokers(s),
		KafkaGroupID:                  resolveGroupID(s, "egov-ws-services"),
		DefaultLimit:                  s.getInt("egov.waterservice.pagination.default.limit", 50),
		DefaultOffset:                 s.getInt("egov.waterservice.pagination.default.offset", 0),
		MaxLimit:                      s.getInt("egov.waterservice.pagination.max.limit", 500),
		IsIDGenEnabled:                s.getBool("is.idgen.enabled", false),
		IDGenHost:                     s.get("egov.idgen.host", ""),
		IDGenPath:                     s.get("egov.idgen.path", "/egov-idgen/id/_generate"),
		WCIDName:                      s.get("egov.idgen.wcid.name", "ws.connectionNumber"),
		WCIDFormat:                    s.get("egov.idgen.wcid.format", "WS/[CITY.CODE]/[fy:yyyy-yy]/[SEQ_EGOV_COMMON]"),
		WCAPIDName:                    s.get("egov.idgen.wcapid.name", "ws.applicationNumber"),
		WCAPIDFormat:                  s.get("egov.idgen.wcapid.format", "WS/[CITY.CODE]/[fy:yyyy-yy]/[SEQ_EGOV_COMMON]"),
		WDCIDName:                     s.get("egov.idgen.wdcid.name", ""),
		WDCIDFormat:                   s.get("egov.idgen.wdcid.format", ""),
		IsExternalWorkflowEnabled:     s.getBool("is.external.workflow.enabled", false),
		BusinessServiceValue:          s.get("create.ws.workflow.name", "NewWS1"),
		ModifyWSBusinessServiceName:   s.get("modify.ws.workflow.name", "ModifyWSConnection"),
		DisconnectBusinessServiceName: s.get("egov.disconnect.businessservice", "WSDisconnection"),
		WfHost:                        s.get("workflow.context.path", ""),
		WfTransitionPath:              s.get("workflow.transition.path", "/egov-workflow-v2/egov-wf/process/_transition"),
		WfBusinessServiceSearchPath:   s.get("workflow.businessservice.search.path", "/egov-workflow-v2/egov-wf/businessservice/_search"),
		WfProcessSearchPath:           s.get("workflow.process.search.path", "/egov-workflow-v2/egov-wf/process/_search"),
		IsPersisterEnabled:            s.getBool("is.persister.enabled", false),
		OnWaterSavedTopic:             s.get("egov.waterservice.createwaterconnection.topic", "save-ws-connection"),
		OnWaterUpdatedTopic:           s.get("egov.waterservice.updatewaterconnection.topic", "update-ws-connection"),
		WorkflowUpdateTopic:           s.get("egov.waterservice.updatewaterconnection.workflow.topic", "update-ws-workflow"),
		CreateMeterReadingTopic:       s.get("ws.meterreading.create.topic", "create-meter-reading"),
		EditNotificationTopic:         s.get("ws.editnotification.topic", "editnotification"),
		FileStoreIdsTopic:             s.get("ws.consume.filestoreids.topic", "ws-filestoreids-process"),
		SaveFileStoreIdsTopic:         s.get("egov.waterservice.savefilestoreIds.topic", "save-ws-filestoreids"),
		ReceiptBusinessTopic:          s.get("egov.receipt.businessservice.topic", ""),
		ReceiptDisconnectionTopic:     s.get("egov.receipt.disconnection.businessservice.topic", ""),
		SmsNotifTopic:                 s.get("kafka.topics.notification.sms", "egov.core.notification.sms"),
		EmailNotifTopic:               s.get("kafka.topics.notification.email", "egov.core.notification.email"),
		SaveUserEventsTopic:           s.get("egov.usr.events.create.topic", "persist-user-events-async"),
		DocumentAuditTopic:            s.get("egov.water.connection.document.access.audit.kafka.topic", ""),
		IsUserEnabled:                 s.getBool("is.user.enabled", false),
		UserHost:                      s.get("egov.user.host", ""),
		UserSearchPath:                s.get("egov.user.search.path", "/user/_search"),
		UserCreatePath:                s.get("egov.user.create.path", "/user/users/_createnovalidate"),
		UserUpdatePath:                s.get("egov.user.update.path", "/user/users/_updatenovalidate"),
		IsPropertyEnabled:             s.getBool("is.property.enabled", false),
		PropertyHost:                  s.get("egov.property.host", ""),
		PropertySearchPath:            s.get("egov.property.search.path", "/property-services/property/_search"),
		IsEncryptionEnabled:           s.getBool("is.encryption.enabled", false),
		EncHost:                       s.get("egov.enc.host", ""),
		EncEncryptPath:                s.get("egov.enc.encrypt.path", "/egov-enc-service/crypto/v1/_encrypt"),
		EncDecryptPath:                s.get("egov.enc.decrypt.path", "/egov-enc-service/crypto/v1/_decrypt"),
		IsMDMSEnabled:                 s.getBool("is.mdms.enabled", false),
		MdmsHost:                      s.get("egov.mdms.host", ""),
		MdmsURL:                       s.get("egov.mdms.search.endpoint", "/egov-mdms-service/v1/_search"),
		CalculatorHost:                s.get("egov.ws.calculation.host", ""),
		CalculateEndpoint:             s.get("egov.ws.calculation.endpoint", "/ws-calculator/waterCalculator/_calculate"),
		EstimationEndpoint:            s.get("egov.ws.estimate.endpoint", "/ws-calculator/waterCalculator/_estimate"),
		BillingServiceHost:            s.get("egov.billing.service.host", ""),
		FetchBillEndpoint:             s.get("egov.fetch.bill.endpoint", "/billing-service/bill/v2/_fetchbill"),
		CollectionHost:                s.get("egov.collection.host", ""),
		PaymentSearchPath:             s.get("egov.collectiom.payment.search", "/collection-services/payments/_search"),
		IsSMSEnabled:                  s.getBool("notification.sms.enabled", false),
		IsEmailEnabled:                s.getBool("notification.email.enabled", false),
		IsUserEventsEnabled:           s.getBool("egov.user.event.notification.enabled", false),
		StateLevelTenantID:            s.get("state.level.tenant.id", "pb"),
	}
	return cfg, nil
}

// dbConn is the resolved database connection, populated from either the
// Go-native DB_* keys or the original Spring Boot SPRING_DATASOURCE_* env.
type dbConn struct{ host, port, name, user, pass, ssl string }

// resolveDB resolves the DB connection so the service is a drop-in for the Java
// pod's environment. Precedence: Go-native env (DB_HOST, ...) > Spring Boot env
// (SPRING_DATASOURCE_URL/USERNAME/PASSWORD) > properties file > default.
func resolveDB(s *settings, defName string) dbConn {
	d := dbConn{
		host: s.get("db.host", "postgres"),
		port: s.get("db.port", "5432"),
		name: s.get("db.name", defName),
		user: s.get("db.user", "postgres"),
		pass: s.get("db.password", "postgres"),
		ssl:  s.get("db.sslmode", "disable"),
	}
	if _, ok := os.LookupEnv("DB_HOST"); !ok {
		if jdbc := envFirst("SPRING_DATASOURCE_URL"); jdbc != "" {
			if h, p, n, ssl, ok := parseJDBCPostgres(jdbc); ok {
				d.host, d.port, d.name = h, p, n
				if ssl != "" && os.Getenv("DB_SSLMODE") == "" {
					d.ssl = ssl
				}
			}
		}
	}
	if os.Getenv("DB_USER") == "" {
		if v := envFirst("SPRING_DATASOURCE_USERNAME"); v != "" {
			d.user = v
		}
	}
	if os.Getenv("DB_PASSWORD") == "" {
		if v := envFirst("SPRING_DATASOURCE_PASSWORD"); v != "" {
			d.pass = v
		}
	}
	return d
}

// resolveBrokers accepts the Go-native KAFKA_BROKERS as well as the Spring/DIGIT
// keys KAFKA_CONFIG_BOOTSTRAP_SERVER_CONFIG and SPRING_KAFKA_BOOTSTRAP_SERVERS.
func resolveBrokers(s *settings) []string {
	v := s.get("kafka.brokers", "")
	if v == "" {
		v = envFirst("KAFKA_CONFIG_BOOTSTRAP_SERVER_CONFIG", "SPRING_KAFKA_BOOTSTRAP_SERVERS")
	}
	if v == "" {
		v = "kafka:9092"
	}
	return strings.Split(v, ",")
}

// resolveGroupID accepts KAFKA_GROUP_ID (Go-native) or SPRING_KAFKA_CONSUMER_GROUP_ID (Java).
func resolveGroupID(s *settings, def string) string {
	v := s.get("kafka.group.id", "")
	if v == "" {
		v = envFirst("SPRING_KAFKA_CONSUMER_GROUP_ID")
	}
	if v == "" {
		v = def
	}
	return v
}

// envFirst returns the first non-empty value among the given env var names.
func envFirst(keys ...string) string {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok && v != "" {
			return v
		}
	}
	return ""
}

// parseJDBCPostgres extracts host, port, db name and optional sslmode from a
// Spring datasource URL such as jdbc:postgresql://host:port/db?sslmode=require.
func parseJDBCPostgres(jdbc string) (host, port, name, sslmode string, ok bool) {
	raw := strings.TrimPrefix(jdbc, "jdbc:")
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", "", "", "", false
	}
	host = u.Hostname()
	port = u.Port()
	if port == "" {
		port = "5432"
	}
	name = strings.TrimPrefix(u.Path, "/")
	sslmode = u.Query().Get("sslmode")
	return host, port, name, sslmode, true
}

// settings resolves keys from env vars first, then a properties file.
type settings struct {
	vals map[string]string
}

func newSettings(files []string) *settings {
	s := &settings{vals: map[string]string{}}
	for _, p := range files {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			s.vals[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
		f.Close()
	}
	return s
}

var envReplacer = strings.NewReplacer(".", "_", "-", "_")

func envKey(key string) string { return strings.ToUpper(envReplacer.Replace(key)) }

func (s *settings) get(key, def string) string {
	if v, ok := os.LookupEnv(envKey(key)); ok {
		return v
	}
	if v, ok := s.vals[key]; ok {
		return v
	}
	return def
}

func (s *settings) getInt(key string, def int) int {
	if v := s.get(key, ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func (s *settings) getBool(key string, def bool) bool {
	if v := s.get(key, ""); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
