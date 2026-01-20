#!/bin/bash
# =============================================================================
# flow-cli Container Entrypoint
# Manages Ollama lifecycle and flow-cli execution
# =============================================================================

set -euo pipefail

# Colors for output (if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
OLLAMA_HOST="${OLLAMA_HOST:-0.0.0.0}"
OLLAMA_PORT="${OLLAMA_PORT:-11434}"
OLLAMA_STARTUP_TIMEOUT="${OLLAMA_STARTUP_TIMEOUT:-120}"
FLOW_DEFAULT_MODEL="${FLOW_LLM_MODEL:-}"
FLOW_AUTO_PULL_MODEL="${FLOW_AUTO_PULL_MODEL:-true}"

# Models to pre-pull on first run (space-separated)
FLOW_PRELOAD_MODELS="${FLOW_PRELOAD_MODELS:-}"

# -----------------------------------------------------------------------------
# Signal Handlers
# -----------------------------------------------------------------------------
OLLAMA_PID=""

cleanup() {
    log_info "Shutting down..."
    if [ -n "$OLLAMA_PID" ] && kill -0 "$OLLAMA_PID" 2>/dev/null; then
        log_info "Stopping Ollama (PID: $OLLAMA_PID)..."
        kill -TERM "$OLLAMA_PID" 2>/dev/null || true
        wait "$OLLAMA_PID" 2>/dev/null || true
    fi
    log_success "Cleanup complete"
    exit 0
}

trap cleanup SIGTERM SIGINT SIGQUIT

# -----------------------------------------------------------------------------
# Ollama Management Functions
# -----------------------------------------------------------------------------

start_ollama() {
    log_info "Starting Ollama server..."

    # Check if Ollama is already running
    if curl -s "http://localhost:${OLLAMA_PORT}/api/tags" >/dev/null 2>&1; then
        log_success "Ollama is already running"
        return 0
    fi

    # Start Ollama in background
    OLLAMA_HOST="${OLLAMA_HOST}" ollama serve &
    OLLAMA_PID=$!

    log_info "Waiting for Ollama to be ready (timeout: ${OLLAMA_STARTUP_TIMEOUT}s)..."

    local count=0
    while ! curl -s "http://localhost:${OLLAMA_PORT}/api/tags" >/dev/null 2>&1; do
        if ! kill -0 "$OLLAMA_PID" 2>/dev/null; then
            log_error "Ollama process died unexpectedly"
            return 1
        fi

        if [ $count -ge "$OLLAMA_STARTUP_TIMEOUT" ]; then
            log_error "Ollama failed to start within ${OLLAMA_STARTUP_TIMEOUT} seconds"
            return 1
        fi

        sleep 1
        count=$((count + 1))

        # Progress indicator every 10 seconds
        if [ $((count % 10)) -eq 0 ]; then
            log_info "Still waiting... (${count}s)"
        fi
    done

    log_success "Ollama is ready (PID: $OLLAMA_PID)"
    return 0
}

check_gpu() {
    if command -v nvidia-smi &>/dev/null; then
        log_info "Checking GPU availability..."
        if nvidia-smi &>/dev/null; then
            log_success "NVIDIA GPU detected:"
            nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader | while read line; do
                log_info "  GPU: $line"
            done
            return 0
        else
            log_warn "nvidia-smi found but GPU not accessible"
            return 1
        fi
    else
        log_info "No NVIDIA GPU detected, running in CPU mode"
        return 1
    fi
}

pull_model() {
    local model="$1"

    if [ -z "$model" ]; then
        return 0
    fi

    log_info "Checking model: $model"

    # Check if model exists
    if ollama list 2>/dev/null | grep -q "^${model}"; then
        log_success "Model '$model' is already available"
        return 0
    fi

    log_info "Pulling model '$model'... (this may take a while)"
    if ollama pull "$model"; then
        log_success "Model '$model' pulled successfully"
        return 0
    else
        log_error "Failed to pull model '$model'"
        return 1
    fi
}

preload_models() {
    # Pull default model if specified
    if [ -n "$FLOW_DEFAULT_MODEL" ] && [ "$FLOW_AUTO_PULL_MODEL" = "true" ]; then
        pull_model "$FLOW_DEFAULT_MODEL"
    fi

    # Pull any pre-specified models
    if [ -n "$FLOW_PRELOAD_MODELS" ]; then
        for model in $FLOW_PRELOAD_MODELS; do
            pull_model "$model" || log_warn "Skipping model: $model"
        done
    fi
}

list_available_models() {
    log_info "Available models:"
    if ollama list 2>/dev/null; then
        return 0
    else
        log_warn "No models available. Pull a model with: ollama pull <model>"
        return 1
    fi
}

# -----------------------------------------------------------------------------
# Main Entrypoint Logic
# -----------------------------------------------------------------------------

main() {
    echo ""
    echo "=========================================="
    echo "  flow-cli Container"
    echo "=========================================="
    echo ""

    # Check GPU availability
    check_gpu || true

    # Start Ollama
    if ! start_ollama; then
        log_error "Failed to start Ollama"
        exit 1
    fi

    # Preload models
    preload_models

    # List available models
    echo ""
    list_available_models
    echo ""

    # Determine what to run
    if [ $# -eq 0 ]; then
        # No arguments: start interactive chat
        log_info "Starting interactive chat mode..."
        exec flow chat
    elif [ "$1" = "ollama" ]; then
        # Direct ollama commands
        shift
        exec ollama "$@"
    elif [ "$1" = "serve" ]; then
        # Just run Ollama server (useful for sidecar mode)
        log_info "Running in server-only mode"
        wait $OLLAMA_PID
    elif [ "$1" = "shell" ] || [ "$1" = "bash" ]; then
        # Drop to shell
        exec /bin/bash
    else
        # Pass all arguments to flow
        exec flow "$@"
    fi
}

# Run main with all arguments
main "$@"
