package docs

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPISpecIsValid(t *testing.T) {
	specBytes := GetSpecBytes()
	if len(specBytes) == 0 {
		t.Fatal("embedded openapi.yaml is empty or was not loaded")
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromData(specBytes)
	if err != nil {
		t.Fatalf("failed to load OpenAPI spec: %v", err)
	}

	err = doc.Validate(context.Background())
	if err != nil {
		t.Fatalf("OpenAPI spec validation failed: %v", err)
	}
}
