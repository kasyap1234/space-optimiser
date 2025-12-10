# =============================================================================
# Stage 1: Build the Go binary
# =============================================================================
FROM golang:1.25-alpine AS builder

# Install build dependencies in a single layer
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build with maximum optimizations:
# - CGO_ENABLED=0: Pure Go binary, no C dependencies
# - GOOS=linux GOARCH=amd64: Target platform
# - -trimpath: Remove file system paths from binary
# - -ldflags="-w -s": Strip debug info and symbol table
# - -buildvcs=false: Skip VCS stamping for reproducibility
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-w -s -extldflags '-static'" \
    -buildvcs=false \
    -o /app/server \
    .

# =============================================================================
# Stage 2: Minimal runtime image
# =============================================================================
FROM scratch

# Import CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/server /server

# Copy static assets
COPY --from=builder /app/static /static

# Set timezone (optional, can be overridden at runtime)
ENV TZ=UTC

# Expose the application port
EXPOSE 8080

# Run as non-root by setting user (scratch doesn't have users, but good for documentation)
# The binary handles the PORT env var internally

# Health check is not supported in scratch, implement in orchestrator if needed

ENTRYPOINT ["/server"]