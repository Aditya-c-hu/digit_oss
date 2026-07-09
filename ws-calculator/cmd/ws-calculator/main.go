package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/egov/ws-calculator/config"
	"github.com/egov/ws-calculator/internal/billing"
	"github.com/egov/ws-calculator/internal/domain"
	"github.com/egov/ws-calculator/internal/mdms"
	"github.com/egov/ws-calculator/internal/repository/postgres"
	"github.com/egov/ws-calculator/internal/service"
	httptransport "github.com/egov/ws-calculator/internal/transport/http"
	wskafka "github.com/egov/ws-calculator/internal/transport/kafka"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// @title           ws-calculator API
// @version         1.7.4
// @description     Water Connection Calculator - Go port of the DIGIT ws-calculator Spring Boot module.
// @host            localhost:8091
// @BasePath        /ws-calculator
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
		logger.Warn("postgres ping failed - continuing", "err", err)
	}

	repo := postgres.New(pool)
	producer := wskafka.NewProducer(cfg.KafkaBrokers, logger)
	defer producer.Close()

	mdmsClient := mdms.New(cfg.MdmsHost, cfg.MdmsURL, cfg.IsMDMSEnabled)
	billingClient := billing.New(cfg.BillingHost, cfg.DemandCreatePath, cfg.DemandUpdatePath, cfg.DemandSearchPath, cfg.IsBillingEnabled)

	calcSvc := service.NewCalculationService(repo, producer, cfg)
	demandSvc := service.NewDemandService(calcSvc, billingClient)
	meterSvc := service.NewMeterService(repo)
	h := httptransport.New(calcSvc, demandSvc, meterSvc)

	loadSlabs := func() { loadMDMSMasters(logger, cfg, mdmsClient, calcSvc) }
	loadSlabs() // initial load on startup, mirroring Java MasterDataService

	stopRefresh, refreshWG := startMasterDataRefresher(loadSlabs)

	// Kafka consumers for the calculator side: meter-reading creates,
	// payment events, and bulk-bill jobs all run as parallel goroutines.
	consumers := wskafka.NewConsumerGroup(cfg.KafkaBrokers, cfg.KafkaGroupID, logger)
	defer consumers.Stop()
	registerConsumers(consumers, cfg, logger, meterSvc)

	srv := newHTTPServer(logger, cfg, h)
	startHTTPServer(logger, cfg, srv)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")
	close(stopRefresh)
	refreshWG.Wait()
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}

func loadMDMSMasters(logger *slog.Logger, cfg *config.Config, mdmsClient *mdms.Client, calcSvc *service.CalculationService) {
	if !mdmsClient.Enabled {
		return
	}
	lctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	loadBillingSlabs(lctx, logger, cfg, mdmsClient, calcSvc)
	loadTimeMasters(lctx, logger, cfg, mdmsClient, calcSvc)
	loadFeeMasters(lctx, logger, cfg, mdmsClient, calcSvc)
}

func loadBillingSlabs(ctx context.Context, logger *slog.Logger, cfg *config.Config, mdmsClient *mdms.Client, calcSvc *service.CalculationService) {
	slabs, err := mdmsClient.LoadBillingSlabs(ctx, nil, cfg.StateLevelTenantID, cfg.BillingSlabModule, cfg.BillingSlabMaster)
	if err != nil {
		logger.Error("billing slab load failed", "err", err)
		return
	}
	if len(slabs) == 0 {
		return
	}
	calcSvc.RefreshSlabs(slabs)
	logger.Info("billing slabs loaded from MDMS", "count", len(slabs))
}

func loadTimeMasters(ctx context.Context, logger *slog.Logger, cfg *config.Config, mdmsClient *mdms.Client, calcSvc *service.CalculationService) {
	penalty, perr := mdmsClient.LoadMaster(ctx, nil, cfg.StateLevelTenantID, cfg.BillingSlabModule, "Penalty")
	interest, ierr := mdmsClient.LoadMaster(ctx, nil, cfg.StateLevelTenantID, cfg.BillingSlabModule, "Interest")
	if perr != nil || ierr != nil {
		logger.Warn("penalty/interest master load issue", "penaltyErr", perr, "interestErr", ierr)
	}
	if len(penalty) == 0 && len(interest) == 0 {
		return
	}
	calcSvc.RefreshTimeMasters(penalty, interest)
	logger.Info("penalty/interest masters loaded from MDMS")
}

func loadFeeMasters(ctx context.Context, logger *slog.Logger, cfg *config.Config, mdmsClient *mdms.Client, calcSvc *service.CalculationService) {
	feeSlab, _ := mdmsClient.LoadMaster(ctx, nil, cfg.StateLevelTenantID, cfg.BillingSlabModule, "FeeSlab")
	roadType, _ := mdmsClient.LoadMaster(ctx, nil, cfg.StateLevelTenantID, cfg.BillingSlabModule, "RoadType")
	plotSlab, _ := mdmsClient.LoadMaster(ctx, nil, cfg.StateLevelTenantID, cfg.BillingSlabModule, "PlotSizeSlab")
	usageType, _ := mdmsClient.LoadMaster(ctx, nil, cfg.StateLevelTenantID, cfg.BillingSlabModule, "PropertyUsageType")
	if len(feeSlab) == 0 && len(roadType) == 0 && len(plotSlab) == 0 && len(usageType) == 0 {
		return
	}
	calcSvc.RefreshFeeMasters(feeSlab, roadType, plotSlab, usageType)
	logger.Info("fee masters loaded from MDMS", "feeSlab", len(feeSlab))
}

func startMasterDataRefresher(loadSlabs func()) (chan struct{}, *sync.WaitGroup) {
	stopRefresh := make(chan struct{})
	refreshWG := &sync.WaitGroup{}
	refreshWG.Add(1)
	go func() {
		defer refreshWG.Done()
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stopRefresh:
				return
			case <-ticker.C:
				loadSlabs()
			}
		}
	}()
	return stopRefresh, refreshWG
}

func registerConsumers(consumers *wskafka.ConsumerGroup, cfg *config.Config, logger *slog.Logger, meterSvc *service.MeterService) {
	consumers.Subscribe("create-meter-reading", func(ctx context.Context, value []byte) error {
		var req domain.MeterConnectionRequest
		if err := json.Unmarshal(value, &req); err != nil {
			return err
		}
		_, err := meterSvc.Create(ctx, &req)
		return err
	})

	consumers.Subscribe(cfg.OnPaymentTopic, func(ctx context.Context, value []byte) error {
		logger.Info("payment event received", "size", len(value))
		return nil
	})

	consumers.Subscribe(cfg.BillGenTopic, func(ctx context.Context, value []byte) error {
		logger.Info("bill-gen event received", "size", len(value))
		return nil
	})
}

func newHTTPServer(logger *slog.Logger, cfg *config.Config, h *httptransport.Handler) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
	h.Register(engine, cfg.ContextPath)

	return &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           recoverMiddleware(logger, requestLogger(logger, maxBytes(engine, maxRequestBytes))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func startHTTPServer(logger *slog.Logger, cfg *config.Config, srv *http.Server) {
	go func() {
		logger.Info("ws-calculator listening", "addr", srv.Addr, "context", cfg.ContextPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()
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
