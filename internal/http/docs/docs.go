package docs

import (
	_ "embed"
	"fmt"
	"net/http"
)

// OpenAPISpec contém os bytes do arquivo embutido.
//
//go:embed openapi.yaml
var OpenAPISpec []byte

// GetSpecBytes retorna os bytes do spec OpenAPI embutido.
func GetSpecBytes() []byte {
	return OpenAPISpec
}

// OpenAPIHandler retorna o conteúdo do spec OpenAPI em YAML.
func OpenAPIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(OpenAPISpec)
	})
}

// ScalarDocsHandler retorna um HTML mínimo com Scalar API Reference via CDN.
func ScalarDocsHandler(specURL string) http.Handler {
	html := fmt.Sprintf(`<!doctype html>
<html>
  <head>
    <title>Linkko API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="%s"
      data-configuration='{"theme":"purple"}'></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`, specURL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})
}
