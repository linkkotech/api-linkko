package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPIHandler(t *testing.T) {
	handler := OpenAPIHandler()
	req := httptest.NewRequest("GET", "/openapi.yaml", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/yaml") {
		t.Errorf("expected content type application/yaml, got %s", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "openapi: 3.0.3") {
		t.Errorf("expected body to contain 'openapi: 3.0.3', got %s", body)
	}
}

func TestScalarDocsHandler(t *testing.T) {
	handler := ScalarDocsHandler("/openapi.yaml")
	req := httptest.NewRequest("GET", "/docs", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected content type text/html, got %s", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "@scalar/api-reference") {
		t.Errorf("expected body to contain '@scalar/api-reference', got %s", body)
	}
	if !strings.Contains(body, "/openapi.yaml") {
		t.Errorf("expected body to contain '/openapi.yaml', got %s", body)
	}
}
