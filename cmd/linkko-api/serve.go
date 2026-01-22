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
	"linkko-api/internal/http/middleware"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/ratelimit"
	"linkko-api/internal/repo"
	"linkko-api/internal/service"
	"linkko-api/internal/telemetry"

	"github.com/go-chi/chi/v5"
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

	// Initialize telemetry
	log.Info(ctx, "initializing telemetry")

	var tracerProvider *sdktrace.TracerProvider
	var meterProvider *sdkmetric.MeterProvider
	var metrics *telemetry.Metrics

	// Inicializar telemetria apenas se habilitada
	if cfg.OTELEnabled {
		// Inicializar tracer
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

		// Inicializar metrics
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
		log.Info(ctx, "telemetry disabled")
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
	contactService := service.NewContactService(contactRepo, auditRepo, workspaceRepo, companyRepo)
	taskService := service.NewTaskService(taskRepo, auditRepo, workspaceRepo)
	companyService := service.NewCompanyService(companyRepo, auditRepo, workspaceRepo)
	pipelineService := service.NewPipelineService(pipelineRepo, auditRepo, workspaceRepo)
	dealService := service.NewDealService(dealRepo, pipelineRepo, workspaceRepo, auditRepo)
	activityService := service.NewActivityService(activityRepo, workspaceRepo, auditRepo)
	portfolioService := service.NewPortfolioService(portfolioRepo, workspaceRepo, auditRepo)

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

	// Create router
	r := chi.NewRouter()

	// Global middlewares (applied to all routes)
	// CRITICAL: Order matters - RequestID → Logging → Recovery → Telemetry
	r.Use(middleware.RequestIDMiddleware)                // 1. Generate/read request ID
	r.Use(middleware.RequestLoggingMiddleware(log))      // 2. Log all requests with request_id and latency
	r.Use(middleware.RecoveryMiddleware(log))            // 3. Catch panics and write 500
	r.Use(telemetry.OTelMiddleware(cfg.OTELServiceName)) // 4. OpenTelemetry tracing
	if metrics != nil {
		r.Use(telemetry.MetricsMiddleware(metrics)) // 5. Prometheus metrics (optional)
	}

	// Public routes
	// /health - Liveness probe (no dependencies checked)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// /ready - Readiness probe (checks critical dependencies)
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Check database connectivity
		if err := pool.Ping(ctx); err != nil {
			log.Error(ctx, "readiness check failed: database unavailable", zap.Error(err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"error","message":"database unavailable"}`))
			return
		}

		// Check Redis connectivity
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Error(ctx, "readiness check failed: redis unavailable", zap.Error(err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"error","message":"redis unavailable"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// Debug routes (dev-only, no authentication required)
	r.Route("/debug", func(r chi.Router) {
		r.Get("/auth", debugHandler.GetAuthDebug)
		r.Get("/auth/workspaces/{workspaceId}", debugHandler.GetAuthDebugWithWorkspace)
		r.Get("/db/ping", debugHandler.PingDB)
	})

	// Protected routes with workspace isolation
	r.Route("/v1/workspaces/{workspaceId}", func(r chi.Router) {
		// Apply authentication (JWT and S2S), workspace validation, and rate limiting
		r.Use(auth.AuthMiddleware(resolver, s2sStore))
		r.Use(middleware.WorkspaceMiddleware)
		r.Use(middleware.RateLimitMiddleware(rateLimiter, cfg.RateLimitPerWorkspacePerMin))

		// Contacts endpoints
		r.Route("/contacts", func(r chi.Router) {
			r.Get("/", contactHandler.ListContacts)
			r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/", contactHandler.CreateContact)

			r.Route("/{contactId}", func(r chi.Router) {
				r.Get("/", contactHandler.GetContact)
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Patch("/", contactHandler.UpdateContact)
				r.Delete("/", contactHandler.DeleteContact)
			})
		})

		// Tasks endpoints (NEW)
		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", taskHandler.ListTasks)
			r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/", taskHandler.CreateTask)

			r.Route("/{taskId}", func(r chi.Router) {
				r.Get("/", taskHandler.GetTask)
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Patch("/", taskHandler.UpdateTask)
				r.Delete("/", taskHandler.DeleteTask)

				// Kanban drag-and-drop (action endpoint)
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/:move", taskHandler.MoveTask)
			})
		})

		// Companies endpoints (NEW)
		r.Route("/companies", func(r chi.Router) {
			r.Get("/", companyHandler.ListCompanies)
			r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/", companyHandler.CreateCompany)

			r.Route("/{companyId}", func(r chi.Router) {
				r.Get("/", companyHandler.GetCompany)
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Patch("/", companyHandler.UpdateCompany)
				r.Delete("/", companyHandler.DeleteCompany)
			})
		})

		// Pipelines endpoints (NEW)
		r.Route("/pipelines", func(r chi.Router) {
			r.Get("/", pipelineHandler.ListPipelines)
			r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/", pipelineHandler.CreatePipeline)

			// Action endpoints
			r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/:create-with-stages", pipelineHandler.CreatePipelineWithStages)
			r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/:seed-default", pipelineHandler.SeedDefaultPipeline)

			r.Route("/{pipelineId}", func(r chi.Router) {
				r.Get("/", pipelineHandler.GetPipeline)
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Patch("/", pipelineHandler.UpdatePipeline)
				r.Delete("/", pipelineHandler.DeletePipeline)

				// Stages nested endpoints
				r.Route("/stages", func(r chi.Router) {
					r.Get("/", pipelineHandler.ListStages)
					r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/", pipelineHandler.CreateStage)

					r.Route("/{stageId}", func(r chi.Router) {
						r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Patch("/", pipelineHandler.UpdateStage)
						r.Delete("/", pipelineHandler.DeleteStage)
					})
				})
			})
		})

		// Deals endpoints (NEW)
		r.Route("/deals", func(r chi.Router) {
			r.Get("/", dealHandler.ListDeals)
			r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/", dealHandler.CreateDeal)

			r.Route("/{dealId}", func(r chi.Router) {
				r.Get("/", dealHandler.GetDeal)
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Patch("/", dealHandler.UpdateDeal)

				// Stage update (action)
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/:move", dealHandler.UpdateDealStage)
			})
		})

		// Timeline / Activities endpoints (NEW)
		r.Route("/timeline", func(r chi.Router) {
			r.Get("/", activityHandler.ListTimeline)

			r.Route("/notes", func(r chi.Router) {
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/", activityHandler.CreateNote)
			})

			r.Route("/calls", func(r chi.Router) {
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/", activityHandler.CreateCall)
			})
		})

		// Portfolio endpoints (NEW)
		r.Route("/portfolio", func(r chi.Router) {
			r.Get("/", portfolioHandler.ListPortfolioItems)
			r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Post("/", portfolioHandler.CreatePortfolioItem)

			r.Route("/{itemID}", func(r chi.Router) {
				r.Get("/", portfolioHandler.GetPortfolioItem)
				r.With(middleware.IdempotencyMiddleware(idempotencyRepo)).Patch("/", portfolioHandler.UpdatePortfolioItem)
				r.Delete("/", portfolioHandler.DeletePortfolioItem)
			})
		})
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
