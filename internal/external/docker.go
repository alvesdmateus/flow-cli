package external

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// DockerClient provides Docker operations.
type DockerClient struct {
	dockerPath string
	timeout    time.Duration
}

// DockerConfig configures the Docker client.
type DockerConfig struct {
	DockerPath string        // Path to docker binary (default: "docker")
	Timeout    time.Duration // Default timeout for operations
}

// ContainerInfo contains information about a container.
type ContainerInfo struct {
	ID      string            `json:"Id"`
	Name    string            `json:"Name"`
	Image   string            `json:"Image"`
	Status  string            `json:"Status"`
	State   string            `json:"State"`
	Created time.Time         `json:"Created"`
	Ports   []PortMapping     `json:"Ports"`
	Labels  map[string]string `json:"Labels"`
}

// PortMapping represents a port mapping.
type PortMapping struct {
	HostIP        string `json:"HostIp"`
	HostPort      string `json:"HostPort"`
	ContainerPort string `json:"ContainerPort"`
	Protocol      string `json:"Protocol"`
}

// ImageInfo contains information about an image.
type ImageInfo struct {
	ID          string    `json:"Id"`
	Repository  string    `json:"Repository"`
	Tag         string    `json:"Tag"`
	Size        int64     `json:"Size"`
	Created     time.Time `json:"Created"`
	VirtualSize int64     `json:"VirtualSize"`
}

// BuildOptions configures docker build.
type BuildOptions struct {
	Dockerfile string            // Path to Dockerfile
	Context    string            // Build context path
	Tags       []string          // Image tags
	BuildArgs  map[string]string // Build arguments
	NoCache    bool              // Disable cache
	Pull       bool              // Always pull base image
	Target     string            // Target build stage
	Platform   string            // Target platform
}

// RunOptions configures docker run.
type RunOptions struct {
	Image       string            // Image to run
	Name        string            // Container name
	Detach      bool              // Run in background
	Remove      bool              // Remove container after exit
	Ports       map[string]string // Port mappings (host:container)
	Volumes     map[string]string // Volume mappings (host:container)
	Env         map[string]string // Environment variables
	Network     string            // Network to connect to
	Entrypoint  string            // Override entrypoint
	Command     []string          // Command to run
	WorkDir     string            // Working directory
	User        string            // User to run as
	Interactive bool              // Keep STDIN open
	TTY         bool              // Allocate a TTY
}

// LogOptions configures docker logs.
type LogOptions struct {
	ContainerID string        // Container ID or name
	Follow      bool          // Follow log output
	Tail        string        // Number of lines from end (e.g., "100", "all")
	Since       string        // Show logs since timestamp
	Until       string        // Show logs until timestamp
	Timestamps  bool          // Show timestamps
}

// BuildResult contains the result of a docker build.
type BuildResult struct {
	ImageID string
	Tags    []string
	Output  string
	Success bool
}

// RunResult contains the result of a docker run.
type RunResult struct {
	ContainerID string
	Output      string
	ExitCode    int
}

// NewDockerClient creates a new Docker client.
func NewDockerClient(config *DockerConfig) *DockerClient {
	dockerPath := "docker"
	timeout := 5 * time.Minute

	if config != nil {
		if config.DockerPath != "" {
			dockerPath = config.DockerPath
		}
		if config.Timeout > 0 {
			timeout = config.Timeout
		}
	}

	return &DockerClient{
		dockerPath: dockerPath,
		timeout:    timeout,
	}
}

// IsAvailable checks if Docker is available.
func (dc *DockerClient) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, "version", "--format", "{{.Server.Version}}")
	return cmd.Run() == nil
}

// Version returns Docker version information.
func (dc *DockerClient) Version() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Docker version: %w", err)
	}

	return string(output), nil
}

// Build builds a Docker image.
func (dc *DockerClient) Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	args := []string{"build"}

	// Add tags
	for _, tag := range opts.Tags {
		args = append(args, "-t", tag)
	}

	// Add Dockerfile path
	if opts.Dockerfile != "" {
		args = append(args, "-f", opts.Dockerfile)
	}

	// Add build args
	for key, value := range opts.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add options
	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	if opts.Pull {
		args = append(args, "--pull")
	}
	if opts.Target != "" {
		args = append(args, "--target", opts.Target)
	}
	if opts.Platform != "" {
		args = append(args, "--platform", opts.Platform)
	}

	// Add context path
	contextPath := opts.Context
	if contextPath == "" {
		contextPath = "."
	}
	args = append(args, contextPath)

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	result := &BuildResult{
		Tags:    opts.Tags,
		Output:  output,
		Success: err == nil,
	}

	if err != nil {
		return result, fmt.Errorf("docker build failed: %w\n%s", err, output)
	}

	// Extract image ID from output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Successfully built") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				result.ImageID = parts[2]
			}
		}
	}

	return result, nil
}

// Run runs a Docker container.
func (dc *DockerClient) Run(ctx context.Context, opts RunOptions) (*RunResult, error) {
	args := []string{"run"}

	// Container name
	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	}

	// Detach mode
	if opts.Detach {
		args = append(args, "-d")
	}

	// Remove after exit
	if opts.Remove {
		args = append(args, "--rm")
	}

	// Interactive and TTY
	if opts.Interactive {
		args = append(args, "-i")
	}
	if opts.TTY {
		args = append(args, "-t")
	}

	// Port mappings
	for host, container := range opts.Ports {
		args = append(args, "-p", fmt.Sprintf("%s:%s", host, container))
	}

	// Volume mappings
	for host, container := range opts.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", host, container))
	}

	// Environment variables
	for key, value := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Network
	if opts.Network != "" {
		args = append(args, "--network", opts.Network)
	}

	// Working directory
	if opts.WorkDir != "" {
		args = append(args, "-w", opts.WorkDir)
	}

	// User
	if opts.User != "" {
		args = append(args, "-u", opts.User)
	}

	// Entrypoint
	if opts.Entrypoint != "" {
		args = append(args, "--entrypoint", opts.Entrypoint)
	}

	// Image
	args = append(args, opts.Image)

	// Command
	args = append(args, opts.Command...)

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	result := &RunResult{
		Output: output,
	}

	// Get exit code
	if exitError, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitError.ExitCode()
	} else if err != nil {
		return result, fmt.Errorf("docker run failed: %w\n%s", err, output)
	}

	// If detached, output is container ID
	if opts.Detach {
		result.ContainerID = strings.TrimSpace(stdout.String())
	}

	return result, nil
}

// Logs retrieves logs from a container.
func (dc *DockerClient) Logs(ctx context.Context, opts LogOptions) (string, error) {
	args := []string{"logs"}

	if opts.Follow {
		args = append(args, "-f")
	}
	if opts.Tail != "" {
		args = append(args, "--tail", opts.Tail)
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}
	if opts.Until != "" {
		args = append(args, "--until", opts.Until)
	}
	if opts.Timestamps {
		args = append(args, "-t")
	}

	args = append(args, opts.ContainerID)

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %w\n%s", err, string(output))
	}

	return string(output), nil
}

// StreamLogs streams logs from a container.
func (dc *DockerClient) StreamLogs(ctx context.Context, opts LogOptions, callback func(line string)) error {
	args := []string{"logs", "-f"}

	if opts.Tail != "" {
		args = append(args, "--tail", opts.Tail)
	}
	if opts.Timestamps {
		args = append(args, "-t")
	}

	args = append(args, opts.ContainerID)

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker logs: %w", err)
	}

	// Stream stdout and stderr
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for scanner.Scan() {
			callback(scanner.Text())
		}
	}()

	return cmd.Wait()
}

// ListContainers lists containers.
func (dc *DockerClient) ListContainers(ctx context.Context, all bool) ([]ContainerInfo, error) {
	args := []string{"ps", "--format", "{{json .}}"}
	if all {
		args = append(args, "-a")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	var containers []ContainerInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		container := ContainerInfo{
			ID:     getString(raw, "ID"),
			Name:   getString(raw, "Names"),
			Image:  getString(raw, "Image"),
			Status: getString(raw, "Status"),
			State:  getString(raw, "State"),
		}
		containers = append(containers, container)
	}

	return containers, nil
}

// ListImages lists images.
func (dc *DockerClient) ListImages(ctx context.Context) ([]ImageInfo, error) {
	args := []string{"images", "--format", "{{json .}}"}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker images failed: %w", err)
	}

	var images []ImageInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		image := ImageInfo{
			ID:         getString(raw, "ID"),
			Repository: getString(raw, "Repository"),
			Tag:        getString(raw, "Tag"),
		}
		images = append(images, image)
	}

	return images, nil
}

// Stop stops a container.
func (dc *DockerClient) Stop(ctx context.Context, containerID string, timeout int) error {
	args := []string{"stop"}
	if timeout > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", timeout))
	}
	args = append(args, containerID)

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout+10)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker stop failed: %w\n%s", err, string(output))
	}

	return nil
}

// Remove removes a container.
func (dc *DockerClient) Remove(ctx context.Context, containerID string, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, containerID)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm failed: %w\n%s", err, string(output))
	}

	return nil
}

// RemoveImage removes an image.
func (dc *DockerClient) RemoveImage(ctx context.Context, imageID string, force bool) error {
	args := []string{"rmi"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, imageID)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rmi failed: %w\n%s", err, string(output))
	}

	return nil
}

// Pull pulls an image.
func (dc *DockerClient) Pull(ctx context.Context, image string) error {
	args := []string{"pull", image}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull failed: %w\n%s", err, string(output))
	}

	return nil
}

// Exec executes a command in a running container.
func (dc *DockerClient) Exec(ctx context.Context, containerID string, command []string, interactive bool) (string, error) {
	args := []string{"exec"}
	if interactive {
		args = append(args, "-it")
	}
	args = append(args, containerID)
	args = append(args, command...)

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker exec failed: %w\n%s", err, string(output))
	}

	return string(output), nil
}

// Inspect inspects a container.
func (dc *DockerClient) Inspect(ctx context.Context, containerID string) (*ContainerInfo, error) {
	args := []string{"inspect", containerID}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse inspect output: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("container not found: %s", containerID)
	}

	raw := results[0]
	info := &ContainerInfo{
		ID:    getString(raw, "Id"),
		Name:  getString(raw, "Name"),
		Image: getString(raw, "Image"),
	}

	if state, ok := raw["State"].(map[string]interface{}); ok {
		info.State = getString(state, "Status")
	}

	return info, nil
}

// ComposeUp runs docker-compose up.
func (dc *DockerClient) ComposeUp(ctx context.Context, composePath string, detach bool) error {
	args := []string{"compose", "-f", composePath, "up"}
	if detach {
		args = append(args, "-d")
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose up failed: %w\n%s", err, string(output))
	}

	return nil
}

// ComposeDown runs docker-compose down.
func (dc *DockerClient) ComposeDown(ctx context.Context, composePath string, volumes bool) error {
	args := []string{"compose", "-f", composePath, "down"}
	if volumes {
		args = append(args, "-v")
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dc.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose down failed: %w\n%s", err, string(output))
	}

	return nil
}

// Helper function to get string from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
