// Example integration for debug routes in cmd/linkko-api/serve.go
// Add this code after setting up the auth middleware

package main

import (
	"linkko-api/internal/auth"
	"linkko-api/internal/http/handler"
	"linkko-api/internal/http/middleware"

	"github.com/go-chi/chi/v5"
)

func setupDebugRoutes(r chi.Router) {
	// Initialize debug handler (checks APP_ENV internally)
	debugHandler := handler.NewDebugHandler()

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		// Auth middleware must be applied before debug routes
		// r.Use(auth.AuthMiddleware(keyStore))  // Already applied in parent

		// Debug route without workspace validation
		// GET /debug/auth
		// Returns: authMethod, actorId, actorType, workspace from token/header
		r.Get("/debug/auth", debugHandler.GetAuthDebug)

		// Workspace-specific routes
		r.Route("/workspaces/{workspaceId}", func(r chi.Router) {
			// Workspace middleware validates workspace match
			// r.Use(middleware.WorkspaceMiddleware)  // Already applied in parent

			// Debug route with workspace validation
			// GET /debug/auth/workspaces/{workspaceId}
			// Returns: same as above + workspaceIdFromPath
			// Will return 403 if workspace mismatch
			r.Get("/debug/auth", debugHandler.GetAuthDebugWithWorkspace)

			// ... other workspace routes
		})
	})
}

// Full example with complete router setup:
func setupRouter() chi.Router {
	r := chi.NewRouter()

	// Public routes (no auth required)
	r.Get("/health", healthHandler)
	r.Get("/ready", readyHandler)

	// Protected routes
	r.Group(func(r chi.Router) {
		// Auth middleware (JWT or S2S)
		r.Use(auth.AuthMiddleware(keyStore))

		// Debug routes (dev only)
		debugHandler := handler.NewDebugHandler()
		r.Get("/debug/auth", debugHandler.GetAuthDebug)

		// Workspace routes
		r.Route("/api/v1/workspaces/{workspaceId}", func(r chi.Router) {
			// Workspace validation (IDOR protection)
			r.Use(middleware.WorkspaceMiddleware)

			// Debug with workspace
			r.Get("/debug/auth", debugHandler.GetAuthDebugWithWorkspace)

			// Business routes
			r.Get("/contacts", contactHandler.ListContacts)
			r.Post("/tasks", taskHandler.CreateTask)
			// ... other routes
		})
	})

	return r
}

// Alternative: Add debug routes to existing router
func addDebugRoutes(r chi.Router) {
	debugHandler := handler.NewDebugHandler()

	// Assumes auth middleware is already applied
	r.Get("/debug/auth", debugHandler.GetAuthDebug)

	// Add to workspace routes
	// r.Get("/api/v1/workspaces/{workspaceId}/debug/auth", debugHandler.GetAuthDebugWithWorkspace)
}
