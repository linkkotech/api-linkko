package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_GetAllowedIssuers_SingleIssuer(t *testing.T) {
	cfg := &Config{
		JWTAllowedIssuers: "linkko-crm-web",
	}

	issuers := cfg.GetAllowedIssuers()

	assert.Len(t, issuers, 1)
	assert.Equal(t, "linkko-crm-web", issuers[0])
}

func TestConfig_GetAllowedIssuers_MultipleIssuers(t *testing.T) {
	cfg := &Config{
		JWTAllowedIssuers: "linkko-crm-web,linkko-admin-portal,linkko-mcp-server",
	}

	issuers := cfg.GetAllowedIssuers()

	assert.Len(t, issuers, 3)
	assert.Equal(t, "linkko-crm-web", issuers[0])
	assert.Equal(t, "linkko-admin-portal", issuers[1])
	assert.Equal(t, "linkko-mcp-server", issuers[2])
}

func TestConfig_GetAllowedIssuers_WithWhitespace(t *testing.T) {
	cfg := &Config{
		JWTAllowedIssuers: "  linkko-crm-web  , linkko-admin-portal , linkko-mcp-server  ",
	}

	issuers := cfg.GetAllowedIssuers()

	assert.Len(t, issuers, 3)
	assert.Equal(t, "linkko-crm-web", issuers[0])
	assert.Equal(t, "linkko-admin-portal", issuers[1])
	assert.Equal(t, "linkko-mcp-server", issuers[2])
}

func TestConfig_GetAllowedIssuers_WithEmptyEntries(t *testing.T) {
	cfg := &Config{
		JWTAllowedIssuers: "linkko-crm-web,,linkko-admin-portal,  ,linkko-mcp-server",
	}

	issuers := cfg.GetAllowedIssuers()

	// Empty entries should be ignored
	assert.Len(t, issuers, 3)
	assert.Equal(t, "linkko-crm-web", issuers[0])
	assert.Equal(t, "linkko-admin-portal", issuers[1])
	assert.Equal(t, "linkko-mcp-server", issuers[2])
}

func TestConfig_GetAllowedIssuers_EmptyString(t *testing.T) {
	cfg := &Config{
		JWTAllowedIssuers: "",
	}

	issuers := cfg.GetAllowedIssuers()

	assert.Len(t, issuers, 0)
}

func TestConfig_GetAllowedIssuers_OnlyWhitespace(t *testing.T) {
	cfg := &Config{
		JWTAllowedIssuers: "   ,  ,   ",
	}

	issuers := cfg.GetAllowedIssuers()

	// All whitespace entries should be ignored
	assert.Len(t, issuers, 0)
}

func TestConfig_GetAllowedIssuers_TrailingComma(t *testing.T) {
	cfg := &Config{
		JWTAllowedIssuers: "linkko-crm-web,linkko-admin-portal,",
	}

	issuers := cfg.GetAllowedIssuers()

	// Trailing comma should be ignored
	assert.Len(t, issuers, 2)
	assert.Equal(t, "linkko-crm-web", issuers[0])
	assert.Equal(t, "linkko-admin-portal", issuers[1])
}

func TestConfig_GetAllowedIssuers_LeadingComma(t *testing.T) {
	cfg := &Config{
		JWTAllowedIssuers: ",linkko-crm-web,linkko-admin-portal",
	}

	issuers := cfg.GetAllowedIssuers()

	// Leading comma should be ignored
	assert.Len(t, issuers, 2)
	assert.Equal(t, "linkko-crm-web", issuers[0])
	assert.Equal(t, "linkko-admin-portal", issuers[1])
}

func TestConfig_GetAllowedIssuers_DuplicateIssuers(t *testing.T) {
	cfg := &Config{
		JWTAllowedIssuers: "linkko-crm-web,linkko-admin-portal,linkko-crm-web",
	}

	issuers := cfg.GetAllowedIssuers()

	// Duplicates are allowed (deduplication happens at resolver level)
	assert.Len(t, issuers, 3)
	assert.Equal(t, "linkko-crm-web", issuers[0])
	assert.Equal(t, "linkko-admin-portal", issuers[1])
	assert.Equal(t, "linkko-crm-web", issuers[2])
}
