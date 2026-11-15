# =============================================================================
# Orbita Production Dockerfile
# Multi-stage build for a minimal, secure production image
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build
# -----------------------------------------------------------------------------
FROM golang:1.27-alpine@sha256:eb61c3873c97416fff0116553fff65d67a3549f72e11d9758849a689cdd8b523 AS builder

# Install build dependencies
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments for versioning
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME

# Build the CLI binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o /app/bin/orbita \
    ./cmd/orbita

# Build the worker binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o /app/bin/orbita-worker \
    ./cmd/worker

# -----------------------------------------------------------------------------
# Stage 2: Production image
# -----------------------------------------------------------------------------
FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS production

# Install runtime dependencies
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    ca-certificates \
    tzdata \
    tini

# Create non-root user for security
RUN addgroup -g 1000 orbita && \
    adduser -u 1000 -G orbita -s /bin/sh -D orbita

# Create directories
RUN mkdir -p /app/data /app/plugins /app/config && \
    chown -R orbita:orbita /app

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --chown=orbita:orbita --from=builder /app/bin/orbita /usr/local/bin/orbita
COPY --chown=orbita:orbita --from=builder /app/bin/orbita-worker /usr/local/bin/orbita-worker

# Copy migrations
COPY --chown=orbita:orbita --from=builder /app/migrations /app/migrations

# Set permissions
RUN chmod +x /usr/local/bin/orbita /usr/local/bin/orbita-worker

# Switch to non-root user
USER orbita

# Environment variables
ENV ORBITA_DATA_DIR=/app/data \
    ORBITA_PLUGIN_DIR=/app/plugins \
    ORBITA_CONFIG_DIR=/app/config \
    TZ=UTC

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["orbita", "health"]

# Use tini as init system for proper signal handling
ENTRYPOINT ["/sbin/tini", "--"]

# Default command
CMD ["orbita", "serve"]

# Labels
LABEL org.opencontainers.image.title="Orbita" \
      org.opencontainers.image.authors="Felix Geelhaar" \
      org.opencontainers.image.description="CLI-first adaptive productivity operating system" \
      org.opencontainers.image.vendor="Orbita" \
      org.opencontainers.image.source="https://github.com/felixgeelhaar/orbita"

# Expose MCP server port (if enabled)
EXPOSE 8080

# -----------------------------------------------------------------------------
# Stage 3: Worker image (optional separate image for workers)
# -----------------------------------------------------------------------------
FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS worker

# Install runtime dependencies
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    ca-certificates \
    tzdata \
    tini

# Create non-root user
RUN addgroup -g 1000 orbita && \
    adduser -u 1000 -G orbita -s /bin/sh -D orbita

# Set working directory
WORKDIR /app

# Copy worker binary from builder
COPY --chown=orbita:orbita --from=builder /app/bin/orbita-worker /usr/local/bin/orbita-worker

# Set permissions
RUN chmod +x /usr/local/bin/orbita-worker

# Switch to non-root user
USER orbita

# Environment variables
ENV TZ=UTC

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["pgrep", "orbita-worker"]

# Use tini as init system
ENTRYPOINT ["/sbin/tini", "--"]

# Default command
CMD ["orbita-worker"]

# Labels
LABEL org.opencontainers.image.title="Orbita Worker" \
      org.opencontainers.image.description="Background worker for Orbita" \
      org.opencontainers.image.vendor="Orbita"

# -----------------------------------------------------------------------------
# Stage 4: Development image with hot reload
# -----------------------------------------------------------------------------
FROM golang:1.27-alpine@sha256:eb61c3873c97416fff0116553fff65d67a3549f72e11d9758849a689cdd8b523 AS development

# Install development tools
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    make \
    bash \
    curl

# Install Air for hot reload
RUN go install github.com/air-verse/air@latest

# Create non-root user
RUN addgroup -g 1000 orbita && \
    adduser -u 1000 -G orbita -s /bin/sh -D orbita

# Set working directory
WORKDIR /app

# Copy go mod files
COPY --chown=orbita:orbita go.mod go.sum ./
RUN go mod download

# Set ownership
RUN chown -R orbita:orbita /app

# Switch to non-root user for development
USER orbita

# Environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux

# Default command (hot reload with Air)
CMD ["air", "-c", ".air.toml"]
