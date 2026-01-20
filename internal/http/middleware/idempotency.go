package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"linkko-api/internal/logger"
	"linkko-api/internal/repo"

	"go.uber.org/zap"
)

// IdempotencyMiddleware handles idempotent requests
func IdempotencyMiddleware(idempotencyRepo *repo.IdempotencyRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.GetLogger(r.Context())

			// Only apply to POST, PUT, PATCH methods
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			// Extract Idempotency-Key header
			idempotencyKey := r.Header.Get("Idempotency-Key")
			if idempotencyKey == "" {
				// No idempotency key, proceed normally
				next.ServeHTTP(w, r)
				return
			}

			// Validate key length
			if len(idempotencyKey) > 255 {
				log.Warn("idempotency key too long", zap.Int("length", len(idempotencyKey)))
				http.Error(w, "idempotency key must be 255 characters or less", http.StatusBadRequest)
				return
			}

			// Get workspace ID
			workspaceID, ok := GetWorkspaceID(r.Context())
			if !ok {
				log.Error("workspace_id not found in context for idempotency")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			// Hash the key
			keyHash := repo.HashKey(idempotencyKey)

			// Add key hash to response header for debugging
			w.Header().Set("X-Idempotency-Key-Hash", keyHash)

			// Check if key exists
			cached, err := idempotencyRepo.CheckKey(r.Context(), workspaceID, keyHash)
			if err != nil {
				log.Error("failed to check idempotency key", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			// If key exists, return cached response
			if cached != nil {
				log.Info("returning cached response for idempotent request",
					zap.String("key_hash", keyHash),
					zap.Int("status", cached.Status),
				)

				// Set cached headers
				for k, v := range cached.Headers {
					w.Header().Set(k, v)
				}
				w.Header().Set("X-Idempotency-Replay", "true")

				w.WriteHeader(cached.Status)
				w.Write(cached.Body)
				return
			}

			// Read request body for storage
			var requestBody []byte
			if r.Body != nil {
				requestBody, err = io.ReadAll(r.Body)
				if err != nil {
					log.Error("failed to read request body", zap.Error(err))
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				// Restore body for downstream handlers
				r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			// Create response recorder
			recorder := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           &bytes.Buffer{},
				headers:        make(map[string]string),
			}

			// Call next handler
			next.ServeHTTP(recorder, r)

			// Store result only for successful responses (2xx)
			if recorder.statusCode >= 200 && recorder.statusCode < 300 {
				// Capture important headers
				for _, key := range []string{"Content-Type", "Location"} {
					if val := recorder.Header().Get(key); val != "" {
						recorder.headers[key] = val
					}
				}

				err = idempotencyRepo.StoreResult(
					r.Context(),
					workspaceID,
					keyHash,
					idempotencyKey,
					r.Method,
					r.URL.Path,
					json.RawMessage(requestBody),
					recorder.statusCode,
					json.RawMessage(recorder.body.Bytes()),
					recorder.headers,
				)
				if err != nil {
					log.Error("failed to store idempotency result", zap.Error(err))
					// Don't fail the request, just log the error
				} else {
					log.Info("stored idempotent request result",
						zap.String("key_hash", keyHash),
						zap.Int("status", recorder.statusCode),
					)
				}
			}
		})
	}
}

// responseRecorder captures response for storage
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	headers    map[string]string
	written    bool
}

func (rr *responseRecorder) WriteHeader(code int) {
	if !rr.written {
		rr.statusCode = code
		rr.written = true
	}
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if !rr.written {
		rr.WriteHeader(http.StatusOK)
	}
	rr.body.Write(b)
	return rr.ResponseWriter.Write(b)
}
