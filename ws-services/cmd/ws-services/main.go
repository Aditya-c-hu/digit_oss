package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/egov/ws-services/config"
	"github.com/egov/ws-services/internal/domain"
	"github.com/egov/ws-services/internal/encryption"
	"github.com/egov/ws-services/internal/idgen"
	"github.com/egov/ws-services/internal/mdms"
	"github.com/egov/ws-services/internal/property"
	"github.com/egov/ws-services/internal/repository/postgres"
	"github.com/egov/ws-services/internal/service"
	httptransport "github.com/egov/ws-services/internal/transport/http"
	wskafka "github.com/egov/ws-services/internal/transport/kafka"
	"github.com/egov/ws-services/internal/user"
	"github.com/egov/ws-services/internal/validator"
	"github.com/egov/ws-services/internal/workflow"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// @title           ws-services API
// @version         1.7.4
// @description     Water Connection Service - Go port of the DIGIT ws-services Spring Boot module.
// @host            localhost:8090
// @BasePath        /ws-services
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dbCancel()
	pool, err := pgxpool.New(dbCtx, "postgres://"+cfg.DBUser+":"+cfg.DBPassword+"@"+cfg.DBHost+":"+cfg.DBPort+"/"+cfg.DBName+"?sslmode="+cfg.DBSSLMode)
	if err != nil {
		logger.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(dbCtx); err != nil {
		logger.Warn("postgres ping failed - continuing anyway", "err", err)
	}

	repo := postgres.New(pool)
	producer := wskafka.NewProducer(cfg.KafkaBrokers, logger)
	defer producer.Close()
	wf := workflow.New(cfg.WfHost, cfg.WfTransitionPath, cfg.WfBusinessServiceSearchPath, cfg.IsExternalWorkflowEnabled)
	idgenClient := idgen.New(cfg.IDGenHost, cfg.IDGenPath, cfg.IsIDGenEnabled)
	mdmsClient := mdms.New(cfg.MdmsHost, cfg.MdmsURL, cfg.IsMDMSEnabled)
	val := validator.New(mdmsClient)
	propClient := property.New(cfg.PropertyHost, cfg.PropertySearchPath, cfg.IsPropertyEnabled)
	userClient := user.New(cfg.UserHost, cfg.UserSearchPath, cfg.UserCreatePath, cfg.IsUserEnabled)
	encClient := encryption.New(cfg.EncHost, cfg.EncEncryptPath, cfg.EncDecryptPath, cfg.StateLevelTenantID, cfg.IsEncryptionEnabled)

	svc := service.New(service.Dependencies{
		Repo:      repo,
		Producer:  producer,
		Workflow:  wf,
		IDGen:     idgenClient,
		Validator: val,
		Property:  propClient,
		User:      userClient,
		Encryptor: encClient,
		Cfg:       cfg,
	})
	h := httptransport.New(svc)

	// Kafka consumers run on independent goroutines. This is the Go counterpart
	// of @KafkaListener in spring-kafka — each Subscribe call spins another worker.
	consumers := wskafka.NewConsumerGroup(cfg.KafkaBrokers, cfg.KafkaGroupID, logger)
	defer consumers.Stop()

	consumers.Subscribe(cfg.WorkflowUpdateTopic, func(ctx context.Context, value []byte) error {
		var req domain.WaterConnectionRequest
		if err := json.Unmarshal(value, &req); err != nil {
			return err
		}
		_, err := svc.Update(ctx, &req)
		return err
	})

	consumers.Subscribe(cfg.EditNotificationTopic, func(ctx context.Context, value []byte) error {
		logger.Info("edit-notification received", "size", len(value))
		return nil
	})

	consumers.Subscribe(cfg.FileStoreIdsTopic, func(ctx context.Context, value []byte) error {
		logger.Info("filestore-ids received", "size", len(value))
		return nil
	})

	consumers.Subscribe(cfg.ReceiptBusinessTopic, func(ctx context.Context, value []byte) error {
		logger.Info("receipt event received", "size", len(value))
		return nil
	})

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Any("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
	h.Register(engine, cfg.ContextPath)

	srv := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           recoverMiddleware(logger, requestLogger(logger, maxBytes(engine, maxRequestBytes))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logger.Info("ws-services listening", "addr", srv.Addr, "context", cfg.ContextPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}

// statusRecorder captures the response status for logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func requestLogger(l *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		l.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"latency_ms", time.Since(start).Milliseconds(),
		)
	})
}

// maxRequestBytes caps request bodies to mitigate memory-exhaustion DoS.
const maxRequestBytes = 1 << 20 // 1 MiB

func maxBytes(next http.Handler, n int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, n)
		next.ServeHTTP(w, r)
	})
}

func recoverMiddleware(l *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				l.Error("panic recovered", "err", rec, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"Errors":[{"code":"INTERNAL_SERVER_ERROR","message":"unexpected error"}]}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
