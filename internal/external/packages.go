package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// PackageManager represents a package manager type.
type PackageManager string

const (
	PackageManagerNPM   PackageManager = "npm"
	PackageManagerYarn  PackageManager = "yarn"
	PackageManagerPnpm  PackageManager = "pnpm"
	PackageManagerPip   PackageManager = "pip"
	PackageManagerCargo PackageManager = "cargo"
	PackageManagerGo    PackageManager = "go"
)

// PackageClient provides package manager operations.
type PackageClient struct {
	manager PackageManager
	timeout time.Duration
}

// PackageInfo contains information about a package.
type PackageInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author,omitempty"`
	License      string            `json:"license,omitempty"`
	Homepage     string            `json:"homepage,omitempty"`
	Repository   string            `json:"repository,omitempty"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	DevDeps      map[string]string `json:"dev_dependencies,omitempty"`
	Keywords     []string          `json:"keywords,omitempty"`
}

// InstalledPackage represents an installed package.
type InstalledPackage struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Wanted    string `json:"wanted,omitempty"`
	Latest    string `json:"latest,omitempty"`
	Location  string `json:"location,omitempty"`
	Outdated  bool   `json:"outdated"`
}

// SearchResult represents a package search result.
type SearchResult struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author,omitempty"`
	Downloads   int64  `json:"downloads,omitempty"`
}

// NewPackageClient creates a new package client.
func NewPackageClient(manager PackageManager) *PackageClient {
	return &PackageClient{
		manager: manager,
		timeout: 2 * time.Minute,
	}
}

// IsAvailable checks if the package manager is available.
func (pc *PackageClient) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch pc.manager {
	case PackageManagerNPM:
		cmd = exec.CommandContext(ctx, "npm", "--version")
	case PackageManagerYarn:
		cmd = exec.CommandContext(ctx, "yarn", "--version")
	case PackageManagerPnpm:
		cmd = exec.CommandContext(ctx, "pnpm", "--version")
	case PackageManagerPip:
		cmd = exec.CommandContext(ctx, "pip", "--version")
	case PackageManagerCargo:
		cmd = exec.CommandContext(ctx, "cargo", "--version")
	case PackageManagerGo:
		cmd = exec.CommandContext(ctx, "go", "version")
	default:
		return false
	}

	return cmd.Run() == nil
}

// Version returns the package manager version.
func (pc *PackageClient) Version() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch pc.manager {
	case PackageManagerNPM:
		cmd = exec.CommandContext(ctx, "npm", "--version")
	case PackageManagerYarn:
		cmd = exec.CommandContext(ctx, "yarn", "--version")
	case PackageManagerPnpm:
		cmd = exec.CommandContext(ctx, "pnpm", "--version")
	case PackageManagerPip:
		cmd = exec.CommandContext(ctx, "pip", "--version")
	case PackageManagerCargo:
		cmd = exec.CommandContext(ctx, "cargo", "--version")
	case PackageManagerGo:
		cmd = exec.CommandContext(ctx, "go", "version")
	default:
		return "", fmt.Errorf("unsupported package manager: %s", pc.manager)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Info retrieves information about a package.
func (pc *PackageClient) Info(ctx context.Context, packageName string) (*PackageInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, pc.timeout)
	defer cancel()

	switch pc.manager {
	case PackageManagerNPM, PackageManagerYarn, PackageManagerPnpm:
		return pc.npmInfo(ctx, packageName)
	case PackageManagerPip:
		return pc.pipInfo(ctx, packageName)
	case PackageManagerCargo:
		return pc.cargoInfo(ctx, packageName)
	case PackageManagerGo:
		return pc.goInfo(ctx, packageName)
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", pc.manager)
	}
}

func (pc *PackageClient) npmInfo(ctx context.Context, packageName string) (*PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "npm", "view", packageName, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("npm view failed: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse npm output: %w", err)
	}

	info := &PackageInfo{
		Name:        getString(raw, "name"),
		Version:     getString(raw, "version"),
		Description: getString(raw, "description"),
		License:     getString(raw, "license"),
		Homepage:    getString(raw, "homepage"),
	}

	if author, ok := raw["author"].(string); ok {
		info.Author = author
	} else if authorMap, ok := raw["author"].(map[string]interface{}); ok {
		info.Author = getString(authorMap, "name")
	}

	if repo, ok := raw["repository"].(map[string]interface{}); ok {
		info.Repository = getString(repo, "url")
	} else if repoStr, ok := raw["repository"].(string); ok {
		info.Repository = repoStr
	}

	if keywords, ok := raw["keywords"].([]interface{}); ok {
		for _, k := range keywords {
			if s, ok := k.(string); ok {
				info.Keywords = append(info.Keywords, s)
			}
		}
	}

	if deps, ok := raw["dependencies"].(map[string]interface{}); ok {
		info.Dependencies = make(map[string]string)
		for k, v := range deps {
			if s, ok := v.(string); ok {
				info.Dependencies[k] = s
			}
		}
	}

	return info, nil
}

func (pc *PackageClient) pipInfo(ctx context.Context, packageName string) (*PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "pip", "show", packageName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pip show failed: %w", err)
	}

	info := &PackageInfo{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Name":
			info.Name = value
		case "Version":
			info.Version = value
		case "Summary":
			info.Description = value
		case "Author":
			info.Author = value
		case "License":
			info.License = value
		case "Home-page":
			info.Homepage = value
		case "Requires":
			if value != "" {
				info.Dependencies = make(map[string]string)
				for _, dep := range strings.Split(value, ", ") {
					info.Dependencies[dep] = ""
				}
			}
		}
	}

	return info, nil
}

func (pc *PackageClient) cargoInfo(ctx context.Context, packageName string) (*PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "cargo", "search", packageName, "--limit", "1")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cargo search failed: %w", err)
	}

	// Parse cargo search output
	// Format: package_name = "version"    # description
	info := &PackageInfo{Name: packageName}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, packageName) {
			// Extract version
			re := regexp.MustCompile(`"([^"]+)"`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				info.Version = matches[1]
			}
			// Extract description
			if idx := strings.Index(line, "#"); idx != -1 {
				info.Description = strings.TrimSpace(line[idx+1:])
			}
			break
		}
	}

	return info, nil
}

func (pc *PackageClient) goInfo(ctx context.Context, packageName string) (*PackageInfo, error) {
	// Use go list to get module info
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", packageName)
	output, err := cmd.Output()
	if err != nil {
		// Try go.dev API if local lookup fails
		return pc.goDevInfo(ctx, packageName)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse go list output: %w", err)
	}

	return &PackageInfo{
		Name:    getString(raw, "Path"),
		Version: getString(raw, "Version"),
	}, nil
}

func (pc *PackageClient) goDevInfo(ctx context.Context, packageName string) (*PackageInfo, error) {
	// Query pkg.go.dev API
	client := NewHTTPClient(nil)
	resp, err := client.GET(ctx, fmt.Sprintf("https://pkg.go.dev/%s?tab=versions", packageName), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch go package info: %w", err)
	}

	// Basic parsing - in production would use proper HTML parsing
	info := &PackageInfo{Name: packageName}

	// Try to extract version from response
	re := regexp.MustCompile(`v\d+\.\d+\.\d+`)
	if matches := re.FindString(resp.Body); matches != "" {
		info.Version = matches
	}

	return info, nil
}

// Install installs a package.
func (pc *PackageClient) Install(ctx context.Context, packageName string, dev bool) error {
	ctx, cancel := context.WithTimeout(ctx, pc.timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch pc.manager {
	case PackageManagerNPM:
		args := []string{"install", packageName}
		if dev {
			args = append(args, "--save-dev")
		}
		cmd = exec.CommandContext(ctx, "npm", args...)
	case PackageManagerYarn:
		args := []string{"add", packageName}
		if dev {
			args = append(args, "--dev")
		}
		cmd = exec.CommandContext(ctx, "yarn", args...)
	case PackageManagerPnpm:
		args := []string{"add", packageName}
		if dev {
			args = append(args, "--save-dev")
		}
		cmd = exec.CommandContext(ctx, "pnpm", args...)
	case PackageManagerPip:
		cmd = exec.CommandContext(ctx, "pip", "install", packageName)
	case PackageManagerCargo:
		cmd = exec.CommandContext(ctx, "cargo", "add", packageName)
	case PackageManagerGo:
		cmd = exec.CommandContext(ctx, "go", "get", packageName)
	default:
		return fmt.Errorf("unsupported package manager: %s", pc.manager)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install failed: %w\n%s", err, string(output))
	}

	return nil
}

// Uninstall removes a package.
func (pc *PackageClient) Uninstall(ctx context.Context, packageName string) error {
	ctx, cancel := context.WithTimeout(ctx, pc.timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch pc.manager {
	case PackageManagerNPM:
		cmd = exec.CommandContext(ctx, "npm", "uninstall", packageName)
	case PackageManagerYarn:
		cmd = exec.CommandContext(ctx, "yarn", "remove", packageName)
	case PackageManagerPnpm:
		cmd = exec.CommandContext(ctx, "pnpm", "remove", packageName)
	case PackageManagerPip:
		cmd = exec.CommandContext(ctx, "pip", "uninstall", "-y", packageName)
	case PackageManagerCargo:
		cmd = exec.CommandContext(ctx, "cargo", "remove", packageName)
	default:
		return fmt.Errorf("unsupported package manager: %s", pc.manager)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uninstall failed: %w\n%s", err, string(output))
	}

	return nil
}

// List lists installed packages.
func (pc *PackageClient) List(ctx context.Context) ([]InstalledPackage, error) {
	ctx, cancel := context.WithTimeout(ctx, pc.timeout)
	defer cancel()

	switch pc.manager {
	case PackageManagerNPM:
		return pc.npmList(ctx)
	case PackageManagerPip:
		return pc.pipList(ctx)
	case PackageManagerCargo:
		return pc.cargoList(ctx)
	case PackageManagerGo:
		return pc.goList(ctx)
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", pc.manager)
	}
}

func (pc *PackageClient) npmList(ctx context.Context) ([]InstalledPackage, error) {
	cmd := exec.CommandContext(ctx, "npm", "list", "--json", "--depth=0")
	output, err := cmd.Output()
	if err != nil {
		// npm list returns non-zero if there are issues, but still provides output
		if len(output) == 0 {
			return nil, fmt.Errorf("npm list failed: %w", err)
		}
	}

	var result struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse npm list output: %w", err)
	}

	var packages []InstalledPackage
	for name, info := range result.Dependencies {
		packages = append(packages, InstalledPackage{
			Name:    name,
			Version: info.Version,
		})
	}

	return packages, nil
}

func (pc *PackageClient) pipList(ctx context.Context) ([]InstalledPackage, error) {
	cmd := exec.CommandContext(ctx, "pip", "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pip list failed: %w", err)
	}

	var result []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pip list output: %w", err)
	}

	var packages []InstalledPackage
	for _, pkg := range result {
		packages = append(packages, InstalledPackage{
			Name:    pkg.Name,
			Version: pkg.Version,
		})
	}

	return packages, nil
}

func (pc *PackageClient) cargoList(ctx context.Context) ([]InstalledPackage, error) {
	cmd := exec.CommandContext(ctx, "cargo", "install", "--list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cargo install --list failed: %w", err)
	}

	var packages []InstalledPackage
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, " ") {
			continue
		}

		// Format: package_name v1.2.3:
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := parts[0]
			version := strings.TrimPrefix(parts[1], "v")
			version = strings.TrimSuffix(version, ":")
			packages = append(packages, InstalledPackage{
				Name:    name,
				Version: version,
			})
		}
	}

	return packages, nil
}

func (pc *PackageClient) goList(ctx context.Context) ([]InstalledPackage, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}

	var packages []InstalledPackage
	decoder := json.NewDecoder(bytes.NewReader(output))

	for decoder.More() {
		var mod struct {
			Path    string `json:"Path"`
			Version string `json:"Version"`
			Main    bool   `json:"Main"`
		}

		if err := decoder.Decode(&mod); err != nil {
			continue
		}

		if !mod.Main && mod.Version != "" {
			packages = append(packages, InstalledPackage{
				Name:    mod.Path,
				Version: mod.Version,
			})
		}
	}

	return packages, nil
}

// Outdated lists outdated packages.
func (pc *PackageClient) Outdated(ctx context.Context) ([]InstalledPackage, error) {
	ctx, cancel := context.WithTimeout(ctx, pc.timeout)
	defer cancel()

	switch pc.manager {
	case PackageManagerNPM:
		return pc.npmOutdated(ctx)
	case PackageManagerPip:
		return pc.pipOutdated(ctx)
	default:
		return nil, fmt.Errorf("outdated not supported for: %s", pc.manager)
	}
}

func (pc *PackageClient) npmOutdated(ctx context.Context) ([]InstalledPackage, error) {
	cmd := exec.CommandContext(ctx, "npm", "outdated", "--json")
	output, _ := cmd.Output() // npm outdated returns non-zero if outdated packages exist

	if len(output) == 0 {
		return nil, nil // No outdated packages
	}

	var result map[string]struct {
		Current string `json:"current"`
		Wanted  string `json:"wanted"`
		Latest  string `json:"latest"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse npm outdated output: %w", err)
	}

	var packages []InstalledPackage
	for name, info := range result {
		packages = append(packages, InstalledPackage{
			Name:     name,
			Version:  info.Current,
			Wanted:   info.Wanted,
			Latest:   info.Latest,
			Outdated: true,
		})
	}

	return packages, nil
}

func (pc *PackageClient) pipOutdated(ctx context.Context) ([]InstalledPackage, error) {
	cmd := exec.CommandContext(ctx, "pip", "list", "--outdated", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pip list --outdated failed: %w", err)
	}

	var result []struct {
		Name          string `json:"name"`
		Version       string `json:"version"`
		LatestVersion string `json:"latest_version"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pip outdated output: %w", err)
	}

	var packages []InstalledPackage
	for _, pkg := range result {
		packages = append(packages, InstalledPackage{
			Name:     pkg.Name,
			Version:  pkg.Version,
			Latest:   pkg.LatestVersion,
			Outdated: true,
		})
	}

	return packages, nil
}

// Search searches for packages.
func (pc *PackageClient) Search(ctx context.Context, query string) ([]SearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, pc.timeout)
	defer cancel()

	switch pc.manager {
	case PackageManagerNPM:
		return pc.npmSearch(ctx, query)
	case PackageManagerPip:
		return pc.pipSearch(ctx, query)
	case PackageManagerCargo:
		return pc.cargoSearch(ctx, query)
	default:
		return nil, fmt.Errorf("search not supported for: %s", pc.manager)
	}
}

func (pc *PackageClient) npmSearch(ctx context.Context, query string) ([]SearchResult, error) {
	cmd := exec.CommandContext(ctx, "npm", "search", query, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("npm search failed: %w", err)
	}

	var results []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse npm search output: %w", err)
	}

	var searchResults []SearchResult
	for _, r := range results {
		searchResults = append(searchResults, SearchResult{
			Name:        r.Name,
			Version:     r.Version,
			Description: r.Description,
		})
	}

	return searchResults, nil
}

func (pc *PackageClient) pipSearch(ctx context.Context, query string) ([]SearchResult, error) {
	// pip search is disabled, use PyPI API instead
	client := NewHTTPClient(nil)
	resp, err := client.GET(ctx, fmt.Sprintf("https://pypi.org/pypi/%s/json", query), nil)
	if err != nil {
		return nil, fmt.Errorf("PyPI search failed: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, nil // Package not found
	}

	var result struct {
		Info struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Summary     string `json:"summary"`
			Author      string `json:"author"`
		} `json:"info"`
	}

	if err := resp.JSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse PyPI response: %w", err)
	}

	return []SearchResult{{
		Name:        result.Info.Name,
		Version:     result.Info.Version,
		Description: result.Info.Summary,
		Author:      result.Info.Author,
	}}, nil
}

func (pc *PackageClient) cargoSearch(ctx context.Context, query string) ([]SearchResult, error) {
	cmd := exec.CommandContext(ctx, "cargo", "search", query, "--limit", "10")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cargo search failed: %w", err)
	}

	var results []SearchResult
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: package_name = "version"    # description
		re := regexp.MustCompile(`^(\S+)\s+=\s+"([^"]+)"(?:\s+#\s+(.*))?`)
		if matches := re.FindStringSubmatch(line); len(matches) >= 3 {
			results = append(results, SearchResult{
				Name:        matches[1],
				Version:     matches[2],
				Description: matches[3],
			})
		}
	}

	return results, nil
}

// Update updates packages.
func (pc *PackageClient) Update(ctx context.Context, packageName string) error {
	ctx, cancel := context.WithTimeout(ctx, pc.timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch pc.manager {
	case PackageManagerNPM:
		if packageName == "" {
			cmd = exec.CommandContext(ctx, "npm", "update")
		} else {
			cmd = exec.CommandContext(ctx, "npm", "update", packageName)
		}
	case PackageManagerYarn:
		if packageName == "" {
			cmd = exec.CommandContext(ctx, "yarn", "upgrade")
		} else {
			cmd = exec.CommandContext(ctx, "yarn", "upgrade", packageName)
		}
	case PackageManagerPip:
		if packageName == "" {
			return fmt.Errorf("pip requires a package name for update")
		}
		cmd = exec.CommandContext(ctx, "pip", "install", "--upgrade", packageName)
	case PackageManagerCargo:
		cmd = exec.CommandContext(ctx, "cargo", "update")
	case PackageManagerGo:
		if packageName == "" {
			cmd = exec.CommandContext(ctx, "go", "get", "-u", "./...")
		} else {
			cmd = exec.CommandContext(ctx, "go", "get", "-u", packageName)
		}
	default:
		return fmt.Errorf("unsupported package manager: %s", pc.manager)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update failed: %w\n%s", err, string(output))
	}

	return nil
}
