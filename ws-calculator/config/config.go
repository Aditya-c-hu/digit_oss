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

	// Topics
	DemandSaveTopic        string
	DemandUpdateTopic      string
	BillGenTopic           string
	NotificationSMSTopic   string
	NotificationEmailTopic string
	UserEventsTopic        string
	OnPaymentTopic         string

	// Tax / billing
	TaxPeriod    string
	TimeZone     string
	BillingCycle string

	// One-time application/connection fees (used by _estimate). Defaults stand
	// in for MDMS fee masters until the MDMS fee loader is enabled.
	FormFee         float64
	ScrutinyFee     float64
	SecurityCharge  float64
	RoadCuttingRate float64

	// Penalty / interest (PayService parity). When MDMS is enabled the Penalty /
	// Interest masters override these. Penalty/interest are a flat rate% of the
	// water charge, applied once the connection is overdue beyond applicableDays
	// (matches Java getApplicablePenalty/Interest — no day-proration).
	PenaltyRate            float64
	PenaltyFlat            float64
	PenaltyApplicableDays  float64
	InterestRate           float64
	InterestFlat           float64
	InterestApplicableDays float64

	// External hosts
	WSHost            string
	WSSearchEndpoint  string
	IsBillingEnabled  bool
	BillingHost       string
	DemandCreatePath  string
	DemandUpdatePath  string
	DemandSearchPath  string
	FetchBillPath     string
	IsMDMSEnabled     bool
	MdmsHost          string
	MdmsURL           string
	BillingSlabModule string
	BillingSlabMaster string

	StateLevelTenantID string
}

// Load reads configuration from environment variables, then an
// application.properties file, then code defaults. Standard-library only.
func Load() (*Config, error) {
	s := newSettings([]string{
		"application.properties",
		"config/application.properties",
		"/etc/ws-calculator/application.properties",
	})

	db := resolveDB(s, "rainmaker_new")
	return &Config{
		ServerPort:             s.get("server.port", "8083"),
		ContextPath:            s.get("server.context-path", "/ws-calculator"),
		DBHost:                 db.host,
		DBPort:                 db.port,
		DBUser:                 db.user,
		DBPassword:             db.pass,
		DBName:                 db.name,
		DBSSLMode:              db.ssl,
		KafkaBrokers:           resolveBrokers(s),
		KafkaGroupID:           resolveGroupID(s, "egov-ws-calc-services"),
		DemandSaveTopic:        s.get("egov.demand.save.topic", "egov.demand.save"),
		DemandUpdateTopic:      s.get("egov.demand.update.topic", "egov.demand.update"),
		BillGenTopic:           s.get("ws.bill.gen.topic", "ws-bill-gen"),
		NotificationSMSTopic:   s.get("kafka.topics.notification.sms", "egov.core.notification.sms"),
		NotificationEmailTopic: s.get("kafka.topics.notification.email", "egov.core.notification.email"),
		UserEventsTopic:        s.get("egov.usr.events.create.topic", "persist-user-events-async"),
		OnPaymentTopic:         s.get("egov.payment.topic", "egov.collection.payment-create"),
		TaxPeriod:              s.get("egov.demand.billexpirytime", ""),
		FormFee:                s.getFloat("egov.ws.fee.form", 100.0),
		ScrutinyFee:            s.getFloat("egov.ws.fee.scrutiny", 50.0),
		SecurityCharge:         s.getFloat("egov.ws.fee.security", 500.0),
		RoadCuttingRate:        s.getFloat("egov.ws.fee.roadcutting.rate", 200.0),
		PenaltyRate:            s.getFloat("egov.ws.penalty.rate", 10.0),
		PenaltyFlat:            s.getFloat("egov.ws.penalty.flat", 0.0),
		PenaltyApplicableDays:  s.getFloat("egov.ws.penalty.applicableafterdays", 30.0),
		InterestRate:           s.getFloat("egov.ws.interest.rate", 12.0),
		InterestFlat:           s.getFloat("egov.ws.interest.flat", 0.0),
		InterestApplicableDays: s.getFloat("egov.ws.interest.applicableafterdays", 30.0),
		WSHost:                 s.get("egov.waterservice.host", ""),
		WSSearchEndpoint:       s.get("egov.waterservice.search.endpoint", "/ws-services/wc/_search"),
		IsBillingEnabled:       s.getBool("is.billing.enabled", false),
		BillingHost:            s.get("egov.billing.service.host", ""),
		DemandCreatePath:       s.get("egov.demand.createendpoint", "/billing-service/demand/_create"),
		DemandUpdatePath:       s.get("egov.demand.updateendpoint", "/billing-service/demand/_update"),
		DemandSearchPath:       s.get("egov.demand.searchendpoint", "/billing-service/demand/_search"),
		FetchBillPath:          s.get("egov.fetch.bill.endpoint", "/billing-service/bill/v2/_fetchbill"),
		IsMDMSEnabled:          s.getBool("is.mdms.enabled", false),
		MdmsHost:               s.get("egov.mdms.host", ""),
		MdmsURL:                s.get("egov.mdms.search.endpoint", "/egov-mdms-service/v1/_search"),
		BillingSlabModule:      s.get("egov.ws.billingslab.module", "ws-services-calculation"),
		BillingSlabMaster:      s.get("egov.ws.billingslab.master", "WCBillingSlab"),
		StateLevelTenantID:     s.get("state.level.tenant.id", "pb"),
	}, nil
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

func (s *settings) getBool(key string, def bool) bool {
	if v := s.get(key, ""); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func (s *settings) getFloat(key string, def float64) float64 {
	if v := s.get(key, ""); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
