package config

import "testing"

func TestParseJDBCPostgres(t *testing.T) {
	h, p, n, ssl, ok := parseJDBCPostgres("jdbc:postgresql://db-host:6432/rainmaker_new?sslmode=require")
	if !ok || h != "db-host" || p != "6432" || n != "rainmaker_new" || ssl != "require" {
		t.Fatalf("got host=%q port=%q name=%q ssl=%q ok=%v", h, p, n, ssl, ok)
	}
	// default port when absent
	if _, p2, _, _, _ := parseJDBCPostgres("jdbc:postgresql://h/db"); p2 != "5432" {
		t.Errorf("default port = %q, want 5432", p2)
	}
}

// TestLoadAcceptsSpringEnv proves the Go service is a drop-in for the Java pod's
// environment: the Spring Boot env keys are honoured when the Go-native keys are absent.
func TestLoadAcceptsSpringEnv(t *testing.T) {
	t.Setenv("SPRING_DATASOURCE_URL", "jdbc:postgresql://pg.svc:5432/rainmaker_new")
	t.Setenv("SPRING_DATASOURCE_USERNAME", "wsuser")
	t.Setenv("SPRING_DATASOURCE_PASSWORD", "wspass")
	t.Setenv("KAFKA_CONFIG_BOOTSTRAP_SERVER_CONFIG", "kafka-1:9092,kafka-2:9092")
	t.Setenv("SPRING_KAFKA_CONSUMER_GROUP_ID", "egov-ws-services")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBHost != "pg.svc" || cfg.DBPort != "5432" || cfg.DBName != "rainmaker_new" {
		t.Errorf("DB from SPRING_DATASOURCE_URL = %s:%s/%s", cfg.DBHost, cfg.DBPort, cfg.DBName)
	}
	if cfg.DBUser != "wsuser" || cfg.DBPassword != "wspass" {
		t.Errorf("DB creds = %s/%s", cfg.DBUser, cfg.DBPassword)
	}
	if len(cfg.KafkaBrokers) != 2 || cfg.KafkaBrokers[0] != "kafka-1:9092" {
		t.Errorf("brokers = %v", cfg.KafkaBrokers)
	}
	if cfg.KafkaGroupID != "egov-ws-services" {
		t.Errorf("group = %q", cfg.KafkaGroupID)
	}
}

// TestGoNativeEnvWins confirms the explicit DB_* keys take precedence over Spring.
func TestGoNativeEnvWins(t *testing.T) {
	t.Setenv("SPRING_DATASOURCE_URL", "jdbc:postgresql://spring-host:5432/spring_db")
	t.Setenv("DB_HOST", "go-host")
	t.Setenv("DB_NAME", "go_db")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBHost != "go-host" || cfg.DBName != "go_db" {
		t.Errorf("Go-native DB_* did not win: host=%q name=%q", cfg.DBHost, cfg.DBName)
	}
}
