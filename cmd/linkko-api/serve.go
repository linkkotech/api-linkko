package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"linkko-api/internal/auth"
	"linkko-api/internal/config"
	"linkko-api/internal/database"
	"linkko-api/internal/http/handler"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/ratelimit"
	"linkko-api/internal/repo"
	"linkko-api/internal/service"
	"linkko-api/internal/telemetry"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  `Start the Linkko API HTTP server with all middlewares and observability`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	log, err := logger.New(cfg.OTELServiceName, "info")
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	log.Info(context.Background(), "starting linkko api",
		zap.String("version", "1.0.0"),
		zap.String("service", cfg.OTELServiceName),
	)

	// Run database migrations
	log.Info(ctx, "running database migrations")
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Info(ctx, "migrations completed successfully")

	// Initialize telemetry strictly as opt-in
	var tracerProvider *sdktrace.TracerProvider
	var meterProvider *sdkmetric.MeterProvider
	var metrics *telemetry.Metrics

	if cfg.TelemetryEnabled() {
		log.Info(ctx, "initializing telemetry", zap.String("endpoint", cfg.OTELExporterEndpoint))

		// Initialize tracer
		tp, err := telemetry.InitTracer(ctx, cfg.OTELServiceName, cfg.OTELExporterEndpoint, cfg.OTELSamplingRatio)
		if err != nil {
			log.Warn(ctx, "failed to initialize tracer, continuing without tracing", zap.Error(err))
		} else {
			tracerProvider = tp
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
					log.Error(shutdownCtx, "failed to shutdown tracer provider", zap.Error(err))
				}
			}()
		}

		// Initialize metrics
		mp, m, err := telemetry.InitMetrics(ctx, cfg.OTELServiceName, cfg.OTELExporterEndpoint)
		if err != nil {
			log.Warn(ctx, "failed to initialize metrics, continuing without metrics", zap.Error(err))
		} else {
			meterProvider = mp
			metrics = m
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := meterProvider.Shutdown(shutdownCtx); err != nil {
					log.Error(shutdownCtx, "failed to shutdown meter provider", zap.Error(err))
				}
			}()
		}

		log.Info(ctx, "telemetry initialized", zap.Bool("tracing", tracerProvider != nil), zap.Bool("metrics", metrics != nil))
	} else {
		log.Info(ctx, "telemetry disabled (opt-in only or missing endpoint)")
	}

	// Connect to database
	log.Info(ctx, "connecting to database")
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()
	log.Info(ctx, "database connected")

	// Connect to Redis
	log.Info(ctx, "connecting to redis")
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	// Ping Redis to ensure connectivity
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	log.Info(ctx, "redis connected")

	// Initialize JWT key store and resolver
	log.Info(ctx, "initializing JWT authentication")
	keyStore := auth.NewKeyStore()

	// Load HS256 key for CRM web (JWT_HS256_SECRET must be Base64-encoded)
	log.Info(ctx, "Loading JWT_HS256_SECRET for HS256...")
	secretBytes, err := base64.StdEncoding.DecodeString(cfg.JWTHS256Secret)
	if err != nil {
		return fmt.Errorf("JWT_HS256_SECRET must be valid Base64-encoded: %w", err)
	}
	if len(secretBytes) < 32 {
		return fmt.Errorf("JWT_HS256_SECRET decoded bytes must be at least 32 bytes (256 bits), got %d bytes", len(secretBytes))
	}
	log.Info(ctx, "JWT_HS256_SECRET loaded successfully",
		zap.Int("decoded_bytes", len(secretBytes)),
	)

	// Parse allowed issuers from CSV
	allowedIssuers := cfg.GetAllowedIssuers()
	if len(allowedIssuers) == 0 {
		return fmt.Errorf("JWT_ALLOWED_ISSUERS must contain at least one valid issuer")
	}

	// Load HS256 key for all allowed issuers (same secret for all)
	for _, issuer := range allowedIssuers {
		keyStore.LoadHS256Key(issuer, "v1", secretBytes)
	}

	// Load RS256 key for MCP server (if configured)
	if cfg.JWTPublicKeyMCPV1 != "" {
		if err := keyStore.LoadRS256Key("linkko-mcp-server", "v1", cfg.JWTPublicKeyMCPV1); err != nil {
			return fmt.Errorf("failed to load MCP public key: %w", err)
		}
	}

	// Create validators with clock skew
	clockSkew := time.Duration(cfg.JWTClockSkewSeconds) * time.Second

	// Create resolver with allowed issuers
	resolver := auth.NewKeyResolver(allowedIssuers, []string{cfg.JWTAudience})

	// Register HS256 validator for all allowed issuers
	for _, issuer := range allowedIssuers {
		hs256Validator := auth.NewHS256Validator(keyStore, issuer, clockSkew)
		resolver.RegisterValidator(issuer, hs256Validator)
	}

	// Register RS256 validator if configured
	if cfg.JWTPublicKeyMCPV1 != "" {
		rs256Validator := auth.NewRS256Validator(keyStore, "linkko-mcp-server", clockSkew)
		resolver.RegisterValidator("linkko-mcp-server", rs256Validator)
		// Add MCP issuer to allowed list if not already present
		mcpIssuer := "linkko-mcp-server"
		hasRs256Issuer := false
		for _, issuer := range allowedIssuers {
			if issuer == mcpIssuer {
				hasRs256Issuer = true
				break
			}
		}
		if !hasRs256Issuer {
			allowedIssuers = append(allowedIssuers, mcpIssuer)
		}
	}

	log.Info(ctx, "JWT authentication initialized",
		zap.Strings("allowed_issuers", allowedIssuers),
		zap.Int("clock_skew_seconds", cfg.JWTClockSkewSeconds),
	)

	// Initialize S2S token store
	s2sStore := auth.NewS2STokenStore()
	if cfg.S2STokenCRM != "" {
		s2sStore.RegisterToken(cfg.S2STokenCRM, "crm-web")
		log.Info(ctx, "S2S token registered", zap.String("client", "crm-web"))
	}
	if cfg.S2STokenMCP != "" {
		s2sStore.RegisterToken(cfg.S2STokenMCP, "mcp")
		log.Info(ctx, "S2S token registered", zap.String("client", "mcp"))
	}

	// Initialize repositories
	idempotencyRepo := repo.NewIdempotencyRepo(pool)
	workspaceRepo := repo.NewWorkspaceRepository(pool)
	auditRepo := repo.NewAuditRepo(pool)
	contactRepo := repo.NewContactRepository(pool)
	taskRepo := repo.NewTaskRepository(pool)
	companyRepo := repo.NewCompanyRepository(pool)
	pipelineRepo := repo.NewPipelineRepository(pool)
	dealRepo := repo.NewDealRepository(pool)
	activityRepo := repo.NewActivityRepository(pool)
	portfolioRepo := repo.NewPortfolioRepository(pool)

	// Initialize services
	contactService := service.NewContactService(contactRepo, auditRepo, workspaceRepo, companyRepo, log)
	taskService := service.NewTaskService(taskRepo, auditRepo, workspaceRepo, log)
	companyService := service.NewCompanyService(companyRepo, auditRepo, workspaceRepo, log)
	pipelineService := service.NewPipelineService(pipelineRepo, auditRepo, workspaceRepo, log)
	dealService := service.NewDealService(dealRepo, pipelineRepo, workspaceRepo, auditRepo, log)
	activityService := service.NewActivityService(activityRepo, workspaceRepo, auditRepo, log)
	portfolioService := service.NewPortfolioService(portfolioRepo, workspaceRepo, auditRepo, log)

	// Initialize handlers
	contactHandler := handler.NewContactHandler(contactService)
	taskHandler := handler.NewTaskHandler(taskService)
	companyHandler := handler.NewCompanyHandler(companyService)
	pipelineHandler := handler.NewPipelineHandler(pipelineService)
	dealHandler := handler.NewDealHandler(dealService)
	activityHandler := handler.NewActivityHandler(activityService)
	portfolioHandler := handler.NewPortfolioHandler(portfolioService)
	debugHandler := handler.NewDebugHandler(pool)

	// Initialize rate limiter
	var rateLimitCounter metric.Int64Counter
	if metrics != nil {
		rateLimitCounter = metrics.RateLimitRejections
	}
	rateLimiter := ratelimit.NewRedisRateLimiter(redisClient, rateLimitCounter)

	// Build router
	r := buildRouter(RouterDeps{
		Cfg:              cfg,
		Log:              log,
		Resolver:         resolver,
		S2SStore:         s2sStore,
		IdempotencyRepo:  idempotencyRepo,
		RateLimiter:      rateLimiter,
		Metrics:          metrics,
		Pool:             pool,
		ContactHandler:   contactHandler,
		TaskHandler:      taskHandler,
		CompanyHandler:   companyHandler,
		PipelineHandler:  pipelineHandler,
		DealHandler:      dealHandler,
		ActivityHandler:  activityHandler,
		PortfolioHandler: portfolioHandler,
		DebugHandler:     debugHandler,
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info(ctx, "starting http server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(ctx, "failed to start server", zap.Error(err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info(ctx, "shutdown signal received, starting graceful shutdown")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error(shutdownCtx, "server shutdown error", zap.Error(err))
	}

	log.Info(shutdownCtx, "shutdown complete")
	return nil
}
