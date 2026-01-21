package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration
type Config struct {
	// Database
	DatabaseURL string `env:"DATABASE_URL,required"`

	// Redis
	RedisURL string `env:"REDIS_URL,required"`

	// JWT Configuration
	JWTSecretCRMV1    string `env:"JWT_SECRET_CRM_V1,required"`
	JWTPublicKeyMCPV1 string `env:"JWT_PUBLIC_KEY_MCP_V1,required"`
	JWTAllowedIssuers string `env:"JWT_ALLOWED_ISSUERS,required"`
	JWTAudience       string `env:"JWT_AUDIENCE,required"`

	// OpenTelemetry
	OTELEnabled          bool    `env:"OTEL_ENABLED" envDefault:"true"`
	OTELExporterEndpoint string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4317"`
	OTELServiceName      string  `env:"OTEL_SERVICE_NAME" envDefault:"linkko-api-go"`
	OTELSamplingRatio    float64 `env:"OTEL_SAMPLING_RATIO" envDefault:"0.1"`

	// Server
	Port string `env:"PORT" envDefault:"3002"`

	// Rate Limiting
	RateLimitPerWorkspacePerMin int `env:"RATE_LIMIT_PER_WORKSPACE_PER_MIN" envDefault:"100"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate performs custom validation on the configuration
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}

	if c.JWTSecretCRMV1 == "" {
		return fmt.Errorf("JWT_SECRET_CRM_V1 is required")
	}

	if len(c.JWTSecretCRMV1) < 32 {
		return fmt.Errorf("JWT_SECRET_CRM_V1 must be at least 32 characters")
	}

	if c.JWTPublicKeyMCPV1 == "" {
		return fmt.Errorf("JWT_PUBLIC_KEY_MCP_V1 is required")
	}

	if c.JWTAllowedIssuers == "" {
		return fmt.Errorf("JWT_ALLOWED_ISSUERS is required")
	}

	if c.JWTAudience == "" {
		return fmt.Errorf("JWT_AUDIENCE is required")
	}

	if c.OTELSamplingRatio < 0 || c.OTELSamplingRatio > 1 {
		return fmt.Errorf("OTEL_SAMPLING_RATIO must be between 0 and 1")
	}

	if c.RateLimitPerWorkspacePerMin <= 0 {
		return fmt.Errorf("RATE_LIMIT_PER_WORKSPACE_PER_MIN must be positive")
	}

	return nil
}

// GetAllowedIssuers returns the list of allowed JWT issuers
func (c *Config) GetAllowedIssuers() []string {
	issuers := strings.Split(c.JWTAllowedIssuers, ",")
	result := make([]string, 0, len(issuers))
	for _, issuer := range issuers {
		trimmed := strings.TrimSpace(issuer)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetJWTKeys returns a map of JWT keys by issuer and kid
func (c *Config) GetJWTKeys() map[string]map[string]string {
	return map[string]map[string]string{
		"linkko-crm-web": {
			"v1": c.JWTSecretCRMV1,
		},
		"linkko-mcp-server": {
			"v1": c.JWTPublicKeyMCPV1,
		},
	}
}
