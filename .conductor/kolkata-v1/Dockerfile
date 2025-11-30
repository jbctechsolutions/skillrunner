# Multi-stage Dockerfile for Skillrunner
# Optimized for size and security

# Stage 1: Builder
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
# CGO_ENABLED=0 for static binary
# -ldflags to strip debug info and embed version
ARG VERSION=dev
ARG BUILD_TIME
ARG COMMIT

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.commit=${COMMIT}" \
    -o sr \
    ./cmd/skillrunner

# Stage 2: Runtime
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    curl \
    tzdata

# Create non-root user
RUN addgroup -g 1000 sr && \
    adduser -D -u 1000 -G sr sr

# Create directories
RUN mkdir -p /home/sr/.skillrunner && \
    chown -R sr:sr /home/sr

# Copy binary from builder
COPY --from=builder /build/sr /usr/local/bin/sr

# Copy config example
COPY config.example.yaml /etc/sr/config.example.yaml

# Switch to non-root user
USER sr
WORKDIR /home/sr

# Expose any ports if needed (none for CLI)
# EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD sr status || exit 1

# Set entrypoint
ENTRYPOINT ["sr"]
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="Skillrunner" \
      org.opencontainers.image.description="Local-first AI workflow orchestration with intelligent model routing" \
      org.opencontainers.image.vendor="JBC Tech Solutions" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.url="https://github.com/jbctechsolutions/skillrunner" \
      org.opencontainers.image.documentation="https://github.com/jbctechsolutions/skillrunner/blob/main/README.md"
