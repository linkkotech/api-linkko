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
	JWTHS256Secret      string `env:"JWT_HS256_SECRET,required"`    // Base64-encoded HMAC secret
	JWTAllowedIssuers   string `env:"JWT_ALLOWED_ISSUERS,required"` // CSV list of allowed issuers (e.g., "linkko-crm-web,linkko-mcp-server")
	JWTAudience         string `env:"JWT_AUDIENCE,required"`        // Expected JWT audience
	JWTClockSkewSeconds int    `env:"JWT_CLOCK_SKEW_SECONDS" envDefault:"60"`

	// Legacy JWT Configuration (deprecated)
	JWTSecretCRMV1    string `env:"JWT_SECRET_CRM_V1"`     // Deprecated: use JWT_HS256_SECRET
	JWTPublicKeyMCPV1 string `env:"JWT_PUBLIC_KEY_MCP_V1"` // Deprecated: use S2S tokens
	JWTIssuer         string `env:"JWT_ISSUER"`            // Deprecated: use JWT_ALLOWED_ISSUERS (CSV)

	// S2S (Service-to-Service) Tokens
	S2STokenCRM string `env:"S2S_TOKEN_CRM"`
	S2STokenMCP string `env:"S2S_TOKEN_MCP"`

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

	// Validate JWT_HS256_SECRET (must be valid Base64)
	if c.JWTHS256Secret == "" {
		// Fallback to legacy variable
		if c.JWTSecretCRMV1 != "" {
			c.JWTHS256Secret = c.JWTSecretCRMV1
		} else {
			return fmt.Errorf("JWT_HS256_SECRET is required")
		}
	}

	// Validate JWT_ALLOWED_ISSUERS (CSV list)
	if c.JWTAllowedIssuers == "" {
		// Fallback to legacy single issuer variable
		if c.JWTIssuer != "" {
			c.JWTAllowedIssuers = c.JWTIssuer
		} else {
			c.JWTAllowedIssuers = "linkko-crm-web" // default
		}
	}

	// Validate that parsed issuers list is not empty
	issuers := c.GetAllowedIssuers()
	if len(issuers) == 0 {
		return fmt.Errorf("JWT_ALLOWED_ISSUERS must contain at least one valid issuer")
	}

	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}

	if c.JWTAudience == "" {
		return fmt.Errorf("JWT_AUDIENCE is required")
	}

	if c.OTELSamplingRatio < 0 || c.OTELSamplingRatio > 1 {
		return fmt.Errorf("OTEL_SAMPLING_RATIO must be between 0 and 1")
	}

	if c.JWTClockSkewSeconds < 0 {
		return fmt.Errorf("JWT_CLOCK_SKEW_SECONDS must be non-negative")
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
