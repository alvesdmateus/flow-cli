# =============================================================================
# flow-cli Dockerfile
# Multi-stage build with Ollama integration for self-hosted LLM inference
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build the Go binary
# -----------------------------------------------------------------------------
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set build arguments for version injection
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

# Create non-root user for the build
RUN adduser -D -g '' appuser

WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with version info
# CGO disabled for static binary, stripped for smaller size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w \
        -X github.com/mateus/flow-cli/cmd.Version=${VERSION} \
        -X github.com/mateus/flow-cli/cmd.Commit=${COMMIT} \
        -X github.com/mateus/flow-cli/cmd.BuildDate=${BUILD_DATE}" \
    -o /build/flow .

# -----------------------------------------------------------------------------
# Stage 2: Production image with Ollama
# -----------------------------------------------------------------------------
FROM ubuntu:24.04 AS production

# Labels for container metadata
LABEL org.opencontainers.image.title="flow-cli"
LABEL org.opencontainers.image.description="AI-powered coding assistant CLI with self-hosted LLM"
LABEL org.opencontainers.image.source="https://github.com/mateus/flow-cli"
LABEL org.opencontainers.image.vendor="flow-cli"

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    wget \
    git \
    tini \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Install Ollama
RUN curl -fsSL https://ollama.com/install.sh | sh

# Create non-root user
RUN groupadd --gid 1000 flow \
    && useradd --uid 1000 --gid flow --shell /bin/bash --create-home flow

# Create necessary directories
RUN mkdir -p /app /data /home/flow/.flow /home/flow/.ollama \
    && chown -R flow:flow /app /data /home/flow

# Copy binary from builder
COPY --from=builder /build/flow /usr/local/bin/flow

# Copy entrypoint script
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set environment variables
ENV FLOW_LLM_PROVIDER=ollama
ENV FLOW_LLM_ENDPOINT=http://localhost:11434
ENV FLOW_LLM_MODEL=""
ENV OLLAMA_HOST=0.0.0.0
ENV OLLAMA_MODELS=/home/flow/.ollama/models
ENV HOME=/home/flow

# Expose ports
# 11434 - Ollama API
# 8080  - Optional SearXNG proxy
EXPOSE 11434 8080

# Volumes for persistence
VOLUME ["/home/flow/.ollama", "/home/flow/.flow", "/data"]

# Health check for Ollama
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:11434/api/tags || exit 1

# Use tini as init system for proper signal handling
ENTRYPOINT ["/usr/bin/tini", "--", "/entrypoint.sh"]

# Default command: interactive chat
CMD ["chat"]

# -----------------------------------------------------------------------------
# Stage 3: Minimal CLI-only image (without Ollama)
# For use with external Ollama/OpenAI-compatible endpoints
# -----------------------------------------------------------------------------
FROM alpine:3.21 AS cli-only

LABEL org.opencontainers.image.title="flow-cli-minimal"
LABEL org.opencontainers.image.description="Minimal flow-cli image for external LLM endpoints"

# Install minimal runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    tini

# Create non-root user
RUN addgroup -g 1000 flow \
    && adduser -u 1000 -G flow -s /bin/sh -D flow

# Create directories
RUN mkdir -p /app /data /home/flow/.flow \
    && chown -R flow:flow /app /data /home/flow

# Copy binary from builder
COPY --from=builder /build/flow /usr/local/bin/flow

# Set environment variables
ENV FLOW_LLM_PROVIDER=ollama
ENV FLOW_LLM_ENDPOINT=http://ollama:11434
ENV HOME=/home/flow

# Switch to non-root user
USER flow

WORKDIR /data

# Volume for flow configuration
VOLUME ["/home/flow/.flow", "/data"]

ENTRYPOINT ["/sbin/tini", "--", "flow"]
CMD ["--help"]

# -----------------------------------------------------------------------------
# Stage 4: GPU-enabled image with NVIDIA CUDA support
# -----------------------------------------------------------------------------
FROM nvidia/cuda:12.4.1-runtime-ubuntu24.04 AS gpu

LABEL org.opencontainers.image.title="flow-cli-gpu"
LABEL org.opencontainers.image.description="GPU-accelerated flow-cli with Ollama and CUDA support"

ENV DEBIAN_FRONTEND=noninteractive

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    wget \
    git \
    tini \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Install Ollama
RUN curl -fsSL https://ollama.com/install.sh | sh

# Create non-root user
RUN groupadd --gid 1000 flow \
    && useradd --uid 1000 --gid flow --shell /bin/bash --create-home flow

# Create necessary directories
RUN mkdir -p /app /data /home/flow/.flow /home/flow/.ollama \
    && chown -R flow:flow /app /data /home/flow

# Copy binary from builder
COPY --from=builder /build/flow /usr/local/bin/flow

# Copy entrypoint script
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set environment variables
ENV FLOW_LLM_PROVIDER=ollama
ENV FLOW_LLM_ENDPOINT=http://localhost:11434
ENV FLOW_LLM_MODEL=""
ENV OLLAMA_HOST=0.0.0.0
ENV OLLAMA_MODELS=/home/flow/.ollama/models
ENV HOME=/home/flow
# Enable GPU memory fraction control
ENV CUDA_VISIBLE_DEVICES=all
ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=compute,utility

# Expose ports
EXPOSE 11434 8080

# Volumes for persistence
VOLUME ["/home/flow/.ollama", "/home/flow/.flow", "/data"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 \
    CMD curl -f http://localhost:11434/api/tags || exit 1

ENTRYPOINT ["/usr/bin/tini", "--", "/entrypoint.sh"]
CMD ["chat"]
