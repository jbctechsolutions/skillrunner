# Dockerfile for Skillrunner (GoReleaser)
# Uses pre-built binary from GoReleaser

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

# Copy pre-built binary from GoReleaser
COPY skillrunner /usr/local/bin/skillrunner
RUN ln -s /usr/local/bin/skillrunner /usr/local/bin/sr

# Copy config example
COPY config.example.yaml /etc/skillrunner/config.example.yaml

# Switch to non-root user
USER sr
WORKDIR /home/sr

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD skillrunner status || exit 1

# Set entrypoint
ENTRYPOINT ["skillrunner"]
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="Skillrunner" \
      org.opencontainers.image.description="Local-first AI workflow orchestration with intelligent model routing" \
      org.opencontainers.image.vendor="JBC Tech Solutions" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.url="https://github.com/jbctechsolutions/skillrunner" \
      org.opencontainers.image.documentation="https://github.com/jbctechsolutions/skillrunner/blob/main/README.md"
