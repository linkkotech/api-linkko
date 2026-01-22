package logger

import (
	"context"
	"fmt"
	"strings"

	"linkko-api/internal/observability/requestid"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Context keys for extracting values from context
type contextKey string

const (
	loggerContextKey      contextKey = "logger"
	workspaceIDContextKey contextKey = "workspace_id"
	userIDContextKey      contextKey = "user_id"
	rootErrorContextKey   contextKey = "root_err"
)

type rootErrorContainer struct {
	err error
}

// Logger wraps zap.Logger to enforce structured logging standards
type Logger struct {
	zap         *zap.Logger
	serviceName string
}

// Field represents a structured log field
type Field = zapcore.Field

// New creates a new Logger instance with required base fields
// level: "debug", "info", "warn", "error"
func New(serviceName string, level string) (*Logger, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("serviceName is required")
	}

	zapLevel := parseLevel(level)

	// Configure zap to output JSON with RFC3339Nano timestamps
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339NanoTimeEncoder, // RFC3339Nano as required
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	z, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}

	// Add service name as base field on all logs
	z = z.With(zap.String("service", serviceName))

	return &Logger{
		zap:         z,
		serviceName: serviceName,
	}, nil
}

// WithContext returns a logger that includes context values (request_id, workspace_id, user_id)
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := []Field{}

	if requestID := requestid.GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	if workspaceID := GetWorkspaceIDFromContext(ctx); workspaceID != "" {
		fields = append(fields, zap.String("workspace_id", workspaceID))
	}

	if userID := GetUserIDFromContext(ctx); userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	if len(fields) == 0 {
		return l
	}

	return &Logger{
		zap:         l.zap.With(fields...),
		serviceName: l.serviceName,
	}
}

// Module returns a field for the module/component
func Module(name string) Field {
	return zap.String("module", name)
}

// Action returns a field for the action/operation
func Action(name string) Field {
	return zap.String("action", name)
}

// Info logs an info message with mandatory module and action
// WHY this design: module and action are enforced by convention, not by panic.
// Rationale: In production, a missing module/action should degrade gracefully to defaults,
// not crash the service. The developer experience is to always provide them,
// but runtime resilience is prioritized.
func (l *Logger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, zapcore.InfoLevel, msg, fields...)
}

// Warn logs a warning message with mandatory module and action
func (l *Logger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, zapcore.WarnLevel, msg, fields...)
}

// Error logs an error message with mandatory module and action
func (l *Logger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, zapcore.ErrorLevel, msg, fields...)
}

// Debug logs a debug message with mandatory module and action
func (l *Logger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, zapcore.DebugLevel, msg, fields...)
}

// log is the internal logging implementation
func (l *Logger) log(ctx context.Context, level zapcore.Level, msg string, fields ...Field) {
	// Extract context values
	contextFields := []Field{}

	if requestID := GetRequestIDFromContext(ctx); requestID != "" {
		contextFields = append(contextFields, zap.String("request_id", requestID))
	}

	if workspaceID := GetWorkspaceIDFromContext(ctx); workspaceID != "" {
		contextFields = append(contextFields, zap.String("workspace_id", workspaceID))
	}

	if userID := GetUserIDFromContext(ctx); userID != "" {
		contextFields = append(contextFields, zap.String("user_id", userID))
	}

	// Sanitize fields to prevent logging secrets
	sanitizedFields := sanitizeFields(fields)

	// Ensure module and action are present
	// If missing, provide defaults with warning (production-safe degradation)
	hasModule := false
	hasAction := false
	for _, f := range sanitizedFields {
		if f.Key == "module" {
			hasModule = true
		}
		if f.Key == "action" {
			hasAction = true
		}
	}

	if !hasModule {
		sanitizedFields = append(sanitizedFields, zap.String("module", "unknown"))
	}
	if !hasAction {
		sanitizedFields = append(sanitizedFields, zap.String("action", "unknown"))
	}

	// Combine context fields and provided fields
	allFields := append(contextFields, sanitizedFields...)

	// Log at appropriate level
	switch level {
	case zapcore.DebugLevel:
		l.zap.Debug(msg, allFields...)
	case zapcore.InfoLevel:
		l.zap.Info(msg, allFields...)
	case zapcore.WarnLevel:
		l.zap.Warn(msg, allFields...)
	case zapcore.ErrorLevel:
		l.zap.Error(msg, allFields...)
	}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// sanitizeFields removes forbidden keys to prevent leaking secrets
// SECURITY GUARDRAIL: blocks authorization, token, password, database_url, etc.
func sanitizeFields(fields []Field) []Field {
	forbiddenKeys := map[string]bool{
		"authorization": true,
		"token":         true,
		"password":      true,
		"secret":        true,
		"api_key":       true,
		"database_url":  true,
		"jwt":           true,
		"bearer":        true,
		"credential":    true,
		// PII that should never be logged directly
		"email":       true,
		"phone":       true,
		"full_name":   true,
		"first_name":  true,
		"last_name":   true,
		"address":     true,
		"credit_card": true,
		"ssn":         true,
	}

	sanitized := make([]Field, 0, len(fields))
	for _, field := range fields {
		keyLower := strings.ToLower(field.Key)
		if forbiddenKeys[keyLower] {
			// Replace with sanitized marker
			sanitized = append(sanitized, zap.String(field.Key, "[REDACTED]"))
		} else {
			sanitized = append(sanitized, field)
		}
	}
	return sanitized
}

// parseLevel converts string level to zapcore.Level
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Context value getters

func GetRequestIDFromContext(ctx context.Context) string {
	return requestid.GetRequestID(ctx)
}

func GetWorkspaceIDFromContext(ctx context.Context) string {
	if v := ctx.Value(workspaceIDContextKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func GetUserIDFromContext(ctx context.Context) string {
	if v := ctx.Value(userIDContextKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// Context value setters

func SetRequestIDInContext(ctx context.Context, requestID string) context.Context {
	return requestid.SetRequestID(ctx, requestID)
}

func SetWorkspaceIDInContext(ctx context.Context, workspaceID string) context.Context {
	return context.WithValue(ctx, workspaceIDContextKey, workspaceID)
}

func SetUserIDInContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

// GetLogger retrieves logger from context or returns a new one
func GetLogger(ctx context.Context) *Logger {
	if v := ctx.Value(loggerContextKey); v != nil {
		if logger, ok := v.(*Logger); ok {
			return logger
		}
	}
	// Fallback: return basic logger (should not happen in production)
	logger, _ := New("linkko-crm-api", "info")
	return logger
}

// SetLoggerInContext stores logger in context
func SetLoggerInContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// InitRootErrorContext initializes context with a pointer to hold the root error
func InitRootErrorContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, rootErrorContextKey, &rootErrorContainer{})
}

// SetRootError sets the root cause error in the context container
func SetRootError(ctx context.Context, err error) {
	if container, ok := ctx.Value(rootErrorContextKey).(*rootErrorContainer); ok {
		container.err = err
	}
}

// GetRootError retrieves the root cause error from the context container
func GetRootError(ctx context.Context) error {
	if container, ok := ctx.Value(rootErrorContextKey).(*rootErrorContainer); ok {
		return container.err
	}
	return nil
}
