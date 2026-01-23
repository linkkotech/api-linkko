# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates build-base

# Set working directory
WORKDIR /app

# Copy go mod files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# -ldflags="-w -s" reduces binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o linkko-api \
    ./cmd/linkko-api

# Final Stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Copy binary and entrypoint
COPY --from=builder /app/linkko-api /usr/local/bin/linkko-api
COPY --from=builder /app/scripts/entrypoint.sh /usr/local/bin/entrypoint.sh

# Ensure migrations are available (even if embedded, copying doesn't hurt)
COPY --from=builder /app/internal/database/migrations /app/internal/database/migrations

# Fix permissions
RUN chmod +x /usr/local/bin/linkko-api /usr/local/bin/entrypoint.sh && \
    chown -R appuser:appuser /app

# Environment defaults
ENV PORT=3002

# Switch to non-root user
USER appuser

# Expose the API port
EXPOSE 3002

# Use our entrypoint script
ENTRYPOINT ["entrypoint.sh"]

# Default command
CMD ["serve"]
