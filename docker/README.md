# flow-cli Docker Deployment

This directory contains Docker configurations for running flow-cli with self-hosted LLM inference via Ollama.

## Quick Start

### CPU Mode (Default)

```bash
# Build and run with default settings
cd docker
cp .env.example .env
docker compose up -d flow

# Attach to interactive session
docker attach flow-cli

# Or run a single command
docker compose run --rm flow run "explain this codebase"
```

### GPU Mode (NVIDIA)

Requires [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html).

```bash
# Build and run with GPU support
docker compose --profile gpu up -d flow-gpu

# Attach to interactive session
docker attach flow-cli-gpu
```

## Deployment Options

### Option 1: All-in-One (Recommended for Single User)

Single container with flow-cli + Ollama bundled together.

```bash
# CPU
docker compose up -d flow

# GPU
docker compose --profile gpu up -d flow-gpu
```

**Pros:**
- Simple deployment
- No network configuration needed
- Single container to manage

**Cons:**
- Resources shared between CLI and LLM
- Can't scale Ollama independently

### Option 2: Sidecar Mode (Recommended for Teams)

Separate containers for flow-cli and Ollama.

```bash
# CPU sidecar
docker compose --profile sidecar up -d

# GPU sidecar
docker compose --profile sidecar-gpu up -d
```

**Pros:**
- Independent scaling
- Share Ollama across multiple flow-cli instances
- Better resource isolation

**Cons:**
- More complex setup
- Container networking required

### Option 3: External Ollama

Use flow-cli with an existing Ollama installation.

```bash
# Build minimal image
docker compose build flow-sidecar

# Run with external Ollama
docker run -it --rm \
  -e FLOW_LLM_ENDPOINT=http://host.docker.internal:11434 \
  -v $(pwd):/data \
  flow-cli:minimal chat
```

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `FLOW_LLM_MODEL` | (empty) | Default model (e.g., `llama3.2`, `codellama`) |
| `FLOW_PRELOAD_MODELS` | (empty) | Space-separated models to pull on start |
| `FLOW_AUTO_PULL_MODEL` | `true` | Auto-pull model if not present |
| `FLOW_SECURITY_AUTO_APPROVE` | `false` | Skip confirmation prompts |
| `PROJECT_DIR` | `.` | Directory to mount as `/data` |
| `MEMORY_LIMIT` | `16G` | Container memory limit |
| `OLLAMA_PORT` | `11434` | Exposed Ollama API port |

### Model Recommendations

| Model | Size | Memory | Use Case |
|-------|------|--------|----------|
| `llama3.2:3b` | 2GB | 4GB | Light usage, fast responses |
| `llama3.2:latest` | 4GB | 8GB | General purpose |
| `codellama:7b` | 4GB | 8GB | Code-focused tasks |
| `deepseek-coder:6.7b` | 4GB | 8GB | Code generation |
| `qwen2.5-coder:7b` | 4GB | 8GB | Code understanding |
| `codellama:13b` | 7GB | 16GB | Better code quality |
| `llama3.1:70b` | 40GB | 48GB+ | Best quality (GPU required) |

## Building

### Build All Targets

```bash
# Build all image variants
docker compose build

# Build specific target
docker compose build flow      # CPU production
docker compose build flow-gpu  # GPU production
docker compose build flow-sidecar  # Minimal CLI-only
```

### Build Arguments

```bash
# Build with version info
docker compose build \
  --build-arg VERSION=$(git describe --tags) \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

## Usage

### Interactive Chat

```bash
# Attach to running container
docker attach flow-cli

# Or start new interactive session
docker compose run --rm flow chat
```

### Single Command

```bash
# Run a single prompt
docker compose run --rm flow run "explain the main function"

# Architecture mode
docker compose run --rm flow arch

# With a specific model
docker compose run --rm -e FLOW_LLM_MODEL=codellama flow run "review this code"
```

### Working with Projects

Mount your project directory:

```bash
# Via environment variable
PROJECT_DIR=/path/to/your/project docker compose up -d flow

# Or directly
docker run -it --rm \
  -v /path/to/your/project:/data \
  flow-cli:latest chat
```

### Model Management

```bash
# List models
docker compose exec flow ollama list

# Pull a new model
docker compose exec flow ollama pull codellama:13b

# Remove a model
docker compose exec flow ollama rm codellama:7b

# Show model info
docker compose exec flow ollama show llama3.2
```

## Volumes

| Volume | Path | Purpose |
|--------|------|---------|
| `ollama_models` | `/home/flow/.ollama` | Ollama models (largest) |
| `flow_config` | `/home/flow/.flow` | flow-cli config & sessions |
| `/data` | Mounted | Your project workspace |

### Volume Locations

```bash
# Find volume locations
docker volume inspect flow-ollama-models

# Backup models
docker run --rm -v flow-ollama-models:/data -v $(pwd):/backup \
  alpine tar czf /backup/ollama-models.tar.gz /data

# Restore models
docker run --rm -v flow-ollama-models:/data -v $(pwd):/backup \
  alpine tar xzf /backup/ollama-models.tar.gz -C /
```

## GPU Support

### Prerequisites

1. Install [NVIDIA Driver](https://www.nvidia.com/drivers)
2. Install [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)

```bash
# Ubuntu/Debian
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
  sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
  sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit
sudo nvidia-ctk runtime configure --runtime=docker
sudo systemctl restart docker
```

### Verify GPU Access

```bash
# Test GPU access in container
docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu24.04 nvidia-smi
```

### Running with GPU

```bash
# Start GPU-enabled container
docker compose --profile gpu up -d flow-gpu

# Check GPU is being used
docker compose exec flow-gpu nvidia-smi
```

## Health Checks

```bash
# Check container health
docker compose ps

# Check Ollama API
curl http://localhost:11434/api/tags

# View logs
docker compose logs -f flow

# Check resource usage
docker stats flow-cli
```

## Troubleshooting

### Ollama Won't Start

```bash
# Check logs
docker compose logs flow | grep -i ollama

# Increase startup timeout
OLLAMA_STARTUP_TIMEOUT=300 docker compose up -d flow
```

### Out of Memory

```bash
# Use smaller model
FLOW_LLM_MODEL=llama3.2:3b docker compose up -d flow

# Increase memory limit
MEMORY_LIMIT=32G docker compose up -d flow
```

### GPU Not Detected

```bash
# Verify NVIDIA runtime
docker info | grep -i nvidia

# Check GPU visibility
docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu24.04 nvidia-smi
```

### Model Pull Fails

```bash
# Pull manually with more output
docker compose exec flow ollama pull llama3.2 --verbose

# Check disk space
docker system df
```

### Permission Denied

```bash
# Check volume permissions
docker compose exec flow ls -la /data

# Fix permissions (if needed)
docker compose exec --user root flow chown -R flow:flow /data
```

## Security Considerations

1. **Do not expose Ollama port publicly** without authentication
2. **Use `FLOW_SECURITY_AUTO_APPROVE=false`** in production
3. **Mount only necessary directories** to `/data`
4. **Review models** before pulling - they execute code on your system
5. **Keep images updated** for security patches

## Architecture

```
+------------------+     +------------------+
|   flow-cli       |     |     Ollama       |
|  (Go binary)     |---->|   (LLM Server)   |
+------------------+     +------------------+
        |                        |
        v                        v
+------------------+     +------------------+
|  /data (mount)   |     | /home/flow/.ollama|
| (your project)   |     |    (models)      |
+------------------+     +------------------+
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
jobs:
  review:
    runs-on: ubuntu-latest
    services:
      ollama:
        image: ollama/ollama:latest
        ports:
          - 11434:11434
    steps:
      - uses: actions/checkout@v4
      - name: Pull model
        run: |
          curl -X POST http://localhost:11434/api/pull \
            -d '{"name": "llama3.2:3b"}'
      - name: Run flow-cli review
        run: |
          docker run --rm \
            --network host \
            -v ${{ github.workspace }}:/data \
            -e FLOW_LLM_ENDPOINT=http://localhost:11434 \
            -e FLOW_LLM_MODEL=llama3.2:3b \
            flow-cli:minimal run "review changes in this PR"
```
