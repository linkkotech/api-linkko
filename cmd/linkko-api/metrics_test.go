package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"linkko-api/internal/config"
	"linkko-api/internal/observability/logger"

	"github.com/stretchr/testify/assert"
)

func TestMetricsEndpoint(t *testing.T) {
	log, _ := logger.New("test", "error")

	t.Run("OpenAccessWhenNoTokenSet", func(t *testing.T) {
		deps := RouterDeps{
			Cfg: &config.Config{MetricsToken: ""},
			Log: log,
		}
		r := buildRouter(deps)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
		assert.Contains(t, w.Body.String(), "go_info")
	})

	t.Run("UnauthorizedWhenTokenSetAndMissingHeader", func(t *testing.T) {
		deps := RouterDeps{
			Cfg: &config.Config{MetricsToken: "secret-token"},
			Log: log,
		}
		r := buildRouter(deps)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "unauthorized")
	})

	t.Run("UnauthorizedWhenTokenSetAndHeaderMismatch", func(t *testing.T) {
		deps := RouterDeps{
			Cfg: &config.Config{MetricsToken: "secret-token"},
			Log: log,
		}
		r := buildRouter(deps)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("X-Metrics-Token", "wrong-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("SuccessWhenTokenSetAndHeaderMatches", func(t *testing.T) {
		deps := RouterDeps{
			Cfg: &config.Config{MetricsToken: "secret-token"},
			Log: log,
		}
		r := buildRouter(deps)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("X-Metrics-Token", "secret-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain"))
		assert.Contains(t, w.Body.String(), "go_gc_duration_seconds")
	})

	t.Run("SuccessWhenTokenSetAndBearerHeaderMatches", func(t *testing.T) {
		deps := RouterDeps{
			Cfg: &config.Config{MetricsToken: "secret-token"},
			Log: log,
		}
		r := buildRouter(deps)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("Authorization", "Bearer secret-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "go_gc_duration_seconds")
	})
}
