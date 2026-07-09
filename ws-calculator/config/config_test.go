package config

import "testing"

// TestDefaultsMatchJava locks the defaults to the Java application.properties
// values so the no-env case matches the Spring service it replaces.
func TestDefaultsMatchJava(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerPort != "8083" {
		t.Errorf("ServerPort default = %q, want 8083 (Java)", cfg.ServerPort)
	}
	if cfg.KafkaGroupID != "egov-ws-calc-services" {
		t.Errorf("KafkaGroupID default = %q, want egov-ws-calc-services (Java)", cfg.KafkaGroupID)
	}
	if cfg.DBName != "rainmaker_new" {
		t.Errorf("DBName default = %q, want rainmaker_new (Java)", cfg.DBName)
	}
}

// TestLoadAcceptsSpringEnv proves the Spring Boot env keys are honoured.
func TestLoadAcceptsSpringEnv(t *testing.T) {
	t.Setenv("SPRING_DATASOURCE_URL", "jdbc:postgresql://pg.svc:5432/rainmaker_new")
	t.Setenv("SPRING_DATASOURCE_USERNAME", "calcuser")
	t.Setenv("SPRING_DATASOURCE_PASSWORD", "calcpass")
	t.Setenv("KAFKA_CONFIG_BOOTSTRAP_SERVER_CONFIG", "kafka-1:9092")
	t.Setenv("SPRING_KAFKA_CONSUMER_GROUP_ID", "egov-ws-calc-services")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBHost != "pg.svc" || cfg.DBName != "rainmaker_new" {
		t.Errorf("DB from SPRING_DATASOURCE_URL = %s/%s", cfg.DBHost, cfg.DBName)
	}
	if cfg.DBUser != "calcuser" || cfg.DBPassword != "calcpass" {
		t.Errorf("DB creds = %s/%s", cfg.DBUser, cfg.DBPassword)
	}
	if len(cfg.KafkaBrokers) != 1 || cfg.KafkaBrokers[0] != "kafka-1:9092" {
		t.Errorf("brokers = %v", cfg.KafkaBrokers)
	}
}
