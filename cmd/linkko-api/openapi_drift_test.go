package main

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"

	"linkko-api/internal/config"
	"linkko-api/internal/http/docs"
	"linkko-api/internal/http/handler"
	"linkko-api/internal/observability/logger"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
)

func TestOpenAPIDriftCheck(t *testing.T) {
	// 1. Build router with minimal/noop deps
	cfg := &config.Config{
		OTELServiceName: "test",
		AppEnv:          "test",
	}
	log, _ := logger.New("test", "error")

	deps := RouterDeps{
		Cfg:              cfg,
		Log:              log,
		ContactHandler:   &handler.ContactHandler{},
		TaskHandler:      &handler.TaskHandler{},
		CompanyHandler:   &handler.CompanyHandler{},
		PipelineHandler:  &handler.PipelineHandler{},
		DealHandler:      &handler.DealHandler{},
		ActivityHandler:  &handler.ActivityHandler{},
		PortfolioHandler: &handler.PortfolioHandler{},
		DebugHandler:     &handler.DebugHandler{},
	}
	r := buildRouter(deps)

	// 2. Load OpenAPI spec
	specBytes := docs.GetSpecBytes()
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(specBytes)
	if err != nil {
		t.Fatalf("failed to load OpenAPI spec: %v", err)
	}

	// 3. Map OpenAPI paths and methods
	documentedRoutes := make(map[string]bool)
	for path, pathItem := range doc.Paths.Map() {
		for method := range pathItem.Operations() {
			documentedRoutes[fmt.Sprintf("%s %s", strings.ToUpper(method), path)] = true
		}
	}

	// 4. Collect implemented routes via chi.Walk
	implementedRoutes := make(map[string]bool)
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		// Ignore técnico/debug
		if strings.HasPrefix(route, "/debug") {
			return nil
		}

		// Only consider business methods
		m := strings.ToUpper(method)
		if m != "GET" && m != "POST" && m != "PUT" && m != "PATCH" && m != "DELETE" {
			return nil
		}

		normalizedPath := normalizeChiPath(route)
		implementedRoutes[fmt.Sprintf("%s %s", m, normalizedPath)] = true
		return nil
	}

	if err := chi.Walk(r, walkFunc); err != nil {
		t.Fatalf("failed to walk chi router: %v", err)
	}

	// 5. Compare
	var missingRoutes []string
	for route := range implementedRoutes {
		if !documentedRoutes[route] {
			missingRoutes = append(missingRoutes, route)
		}
	}

	if len(missingRoutes) > 0 {
		sort.Strings(missingRoutes)
		t.Errorf("Drift detected! The following routes are implemented but NOT documented in OpenAPI:\n%s",
			strings.Join(missingRoutes, "\n"))
	}
}

// normalizeChiPath removes regex from chi parameters and trailing slashes
func normalizeChiPath(path string) string {
	// Remove regex: {id:[0-9]+} -> {id}
	re := regexp.MustCompile(`\{([^:]+):[^}]+\}`)
	normalized := re.ReplaceAllString(path, "{$1}")

	// Remove trailing slash except for root "/"
	if len(normalized) > 1 && strings.HasSuffix(normalized, "/") {
		normalized = normalized[:len(normalized)-1]
	}
	return normalized
}
