package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"linkko-api/internal/http/client"
	"linkko-api/internal/observability/logger"

	"go.uber.org/zap"
)

// MCPClient is a client for interacting with the MCP (Model Context Protocol) server.
// It automatically propagates request_id via RequestIDTransport.
//
// WHY: Centralized client ensures consistent error handling, timeout configuration,
// and automatic correlation ID propagation across all MCP calls.
type MCPClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewMCPClient creates a new MCP client with proper defaults.
// baseURL should be the MCP server base URL (e.g., "http://linkko-mcp-server:8080")
func NewMCPClient(baseURL string) *MCPClient {
	return &MCPClient{
		httpClient: client.NewInternalHTTPClient(), // Includes RequestIDTransport
		baseURL:    baseURL,
	}
}

// NotifyContactCreated notifies MCP server about a new contact creation.
// This is a placeholder implementation demonstrating proper context propagation.
func (c *MCPClient) NotifyContactCreated(ctx context.Context, workspaceID, contactID string) error {
	log := logger.GetLogger(ctx)

	payload := map[string]string{
		"event":        "contact_created",
		"workspace_id": workspaceID,
		"contact_id":   contactID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/v1/notifications", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// X-Request-Id automatically added by RequestIDTransport from ctx

	log.Debug(ctx, "sending notification to mcp",
		logger.Module("mcp"),
		logger.Action("notify_contact_created"),
		zap.String("url", url),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error(ctx, "mcp request failed",
			logger.Module("mcp"),
			logger.Action("notify_contact_created"),
			zap.Error(err),
		)
		return fmt.Errorf("mcp request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		log.Warn(ctx, "mcp returned non-ok status",
			logger.Module("mcp"),
			logger.Action("notify_contact_created"),
			zap.Int("status", resp.StatusCode),
		)
		return fmt.Errorf("unexpected status from mcp: %d", resp.StatusCode)
	}

	log.Info(ctx, "mcp notification sent successfully",
		logger.Module("mcp"),
		logger.Action("notify_contact_created"),
		zap.Int("status", resp.StatusCode),
	)

	return nil
}

// GetAgentSuggestions queries MCP for AI-generated suggestions (placeholder).
// Demonstrates GET request with context propagation.
func (c *MCPClient) GetAgentSuggestions(ctx context.Context, workspaceID, prompt string) ([]string, error) {
	log := logger.GetLogger(ctx)

	url := fmt.Sprintf("%s/v1/agents/suggestions?workspace_id=%s&prompt=%s", c.baseURL, workspaceID, prompt)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// X-Request-Id automatically added by RequestIDTransport

	log.Debug(ctx, "fetching agent suggestions from mcp",
		logger.Module("mcp"),
		logger.Action("get_agent_suggestions"),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error(ctx, "mcp request failed",
			logger.Module("mcp"),
			logger.Action("get_agent_suggestions"),
			zap.Error(err),
		)
		return nil, fmt.Errorf("mcp request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from mcp: %d", resp.StatusCode)
	}

	var result struct {
		Suggestions []string `json:"suggestions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Info(ctx, "agent suggestions retrieved",
		logger.Module("mcp"),
		logger.Action("get_agent_suggestions"),
		zap.Int("count", len(result.Suggestions)),
	)

	return result.Suggestions, nil
}
