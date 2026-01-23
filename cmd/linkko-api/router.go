package main

import (
	"context"
	"net/http"
	"time"

	"linkko-api/internal/auth"
	"linkko-api/internal/config"
	"linkko-api/internal/http/docs"
	"linkko-api/internal/http/handler"
	"linkko-api/internal/http/middleware"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/ratelimit"
	"linkko-api/internal/repo"
	"linkko-api/internal/telemetry"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// RouterDeps contém as dependências necessárias para construir o router.
type RouterDeps struct {
	Cfg             *config.Config
	Log             *logger.Logger
	Resolver        *auth.KeyResolver
	S2SStore        *auth.S2STokenStore
	IdempotencyRepo *repo.IdempotencyRepo
	RateLimiter     *ratelimit.RedisRateLimiter
	Metrics         *telemetry.Metrics
	Pool            *pgxpool.Pool // Necessário para readiness check e debug handler

	// Handlers
	ContactHandler   *handler.ContactHandler
	TaskHandler      *handler.TaskHandler
	CompanyHandler   *handler.CompanyHandler
	PipelineHandler  *handler.PipelineHandler
	DealHandler      *handler.DealHandler
	ActivityHandler  *handler.ActivityHandler
	PortfolioHandler *handler.PortfolioHandler
	DebugHandler     *handler.DebugHandler
}

// buildRouter constrói o chi.Router com todos os middlewares e rotas.
func buildRouter(deps RouterDeps) chi.Router {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.RequestLoggingMiddleware(deps.Log))
	r.Use(middleware.RecoveryMiddleware(deps.Log))
	r.Use(telemetry.OTelMiddleware(deps.Cfg.OTELServiceName))
	if deps.Metrics != nil {
		r.Use(telemetry.MetricsMiddleware(deps.Metrics))
	}

	// Public routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/openapi.yaml", docs.OpenAPIHandler().ServeHTTP)
	r.Get("/docs", docs.ScalarDocsHandler("/openapi.yaml").ServeHTTP)

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if deps.Pool == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ready","note":"pool is nil"}`))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := deps.Pool.Ping(ctx); err != nil {
			deps.Log.Error(ctx, "readiness check failed: database unavailable", zap.Error(err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"error","message":"database unavailable"}`))
			return
		}

		// Redis check is implicit if RateLimiter is working, but here we don't have direct access to redis client
		// In production serve.go, it pings redis directly. To keep it testable, we might skip or use RateLimiter

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	// Debug routes (dev-only)
	if deps.Cfg.AppEnv == "dev" || deps.Cfg.AppEnv == "development" {
		r.Route("/debug", func(r chi.Router) {
			if deps.DebugHandler != nil {
				r.With(auth.AuthMiddleware(deps.Resolver, deps.S2SStore)).Get("/auth", deps.DebugHandler.GetAuthDebug)
				r.With(auth.AuthMiddleware(deps.Resolver, deps.S2SStore)).Get("/auth/workspaces/{workspaceId}", deps.DebugHandler.GetAuthDebugWithWorkspace)
				r.Get("/db/ping", deps.DebugHandler.PingDB)
			}
		})
	}

	// Protected routes with workspace isolation
	r.Route("/v1/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.AuthMiddleware(deps.Resolver, deps.S2SStore))
		r.Use(middleware.WorkspaceMiddleware)
		r.Use(middleware.RateLimitMiddleware(deps.RateLimiter, deps.Cfg.RateLimitPerWorkspacePerMin))

		// Contacts
		if deps.ContactHandler != nil {
			r.Route("/contacts", func(r chi.Router) {
				r.Get("/", deps.ContactHandler.ListContacts)
				r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/", deps.ContactHandler.CreateContact)
				r.Route("/{contactId}", func(r chi.Router) {
					r.Get("/", deps.ContactHandler.GetContact)
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Patch("/", deps.ContactHandler.UpdateContact)
					r.Delete("/", deps.ContactHandler.DeleteContact)
				})
			})
		}

		// Tasks
		if deps.TaskHandler != nil {
			r.Route("/tasks", func(r chi.Router) {
				r.Get("/", deps.TaskHandler.ListTasks)
				r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/", deps.TaskHandler.CreateTask)
				r.Route("/{taskId}", func(r chi.Router) {
					r.Get("/", deps.TaskHandler.GetTask)
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Patch("/", deps.TaskHandler.UpdateTask)
					r.Delete("/", deps.TaskHandler.DeleteTask)
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/:move", deps.TaskHandler.MoveTask)
				})
			})
		}

		// Companies
		if deps.CompanyHandler != nil {
			r.Route("/companies", func(r chi.Router) {
				r.Get("/", deps.CompanyHandler.ListCompanies)
				r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/", deps.CompanyHandler.CreateCompany)
				r.Route("/{companyId}", func(r chi.Router) {
					r.Get("/", deps.CompanyHandler.GetCompany)
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Patch("/", deps.CompanyHandler.UpdateCompany)
					r.Delete("/", deps.CompanyHandler.DeleteCompany)
				})
			})
		}

		// Pipelines
		if deps.PipelineHandler != nil {
			r.Route("/pipelines", func(r chi.Router) {
				r.Get("/", deps.PipelineHandler.ListPipelines)
				r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/", deps.PipelineHandler.CreatePipeline)
				r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/:create-with-stages", deps.PipelineHandler.CreatePipelineWithStages)
				r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/:seed-default", deps.PipelineHandler.SeedDefaultPipeline)
				r.Route("/{pipelineId}", func(r chi.Router) {
					r.Get("/", deps.PipelineHandler.GetPipeline)
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Patch("/", deps.PipelineHandler.UpdatePipeline)
					r.Delete("/", deps.PipelineHandler.DeletePipeline)
					r.Route("/stages", func(r chi.Router) {
						r.Get("/", deps.PipelineHandler.ListStages)
						r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/", deps.PipelineHandler.CreateStage)
						r.Route("/{stageId}", func(r chi.Router) {
							r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Patch("/", deps.PipelineHandler.UpdateStage)
							r.Delete("/", deps.PipelineHandler.DeleteStage)
						})
					})
				})
			})
		}

		// Deals
		if deps.DealHandler != nil {
			r.Route("/deals", func(r chi.Router) {
				r.Get("/", deps.DealHandler.ListDeals)
				r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/", deps.DealHandler.CreateDeal)
				r.Route("/{dealId}", func(r chi.Router) {
					r.Get("/", deps.DealHandler.GetDeal)
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Patch("/", deps.DealHandler.UpdateDeal)
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/:move", deps.DealHandler.UpdateDealStage)
				})
			})
		}

		// Timeline
		if deps.ActivityHandler != nil {
			r.Route("/timeline", func(r chi.Router) {
				r.Get("/", deps.ActivityHandler.ListTimeline)
				r.Route("/notes", func(r chi.Router) {
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/", deps.ActivityHandler.CreateNote)
				})
				r.Route("/calls", func(r chi.Router) {
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/", deps.ActivityHandler.CreateCall)
				})
			})
		}

		// Portfolio
		if deps.PortfolioHandler != nil {
			r.Route("/portfolio", func(r chi.Router) {
				r.Get("/", deps.PortfolioHandler.ListPortfolioItems)
				r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Post("/", deps.PortfolioHandler.CreatePortfolioItem)
				r.Route("/{itemID}", func(r chi.Router) {
					r.Get("/", deps.PortfolioHandler.GetPortfolioItem)
					r.With(middleware.IdempotencyMiddleware(deps.IdempotencyRepo)).Patch("/", deps.PortfolioHandler.UpdatePortfolioItem)
					r.Delete("/", deps.PortfolioHandler.DeletePortfolioItem)
				})
			})
		}
	})

	return r
}
