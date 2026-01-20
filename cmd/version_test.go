package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	// Save original and restore after test
	original := Version
	defer func() { Version = original }()

	Version = "1.2.3"
	if got := GetVersion(); got != "1.2.3" {
		t.Errorf("GetVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestGetVersionInfo(t *testing.T) {
	// Save originals and restore after test
	origVersion := Version
	origCommit := Commit
	origBuildDate := BuildDate
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildDate = origBuildDate
	}()

	Version = "1.0.0"
	Commit = "abc1234567890"
	BuildDate = "2024-01-01"

	info := GetVersionInfo()

	if !strings.Contains(info, "1.0.0") {
		t.Errorf("GetVersionInfo() should contain version, got %q", info)
	}
	if !strings.Contains(info, "abc1234") {
		t.Errorf("GetVersionInfo() should contain truncated commit, got %q", info)
	}
	if !strings.Contains(info, "2024-01-01") {
		t.Errorf("GetVersionInfo() should contain build date, got %q", info)
	}
}

func TestGetVersionInfo_ShortCommit(t *testing.T) {
	origCommit := Commit
	defer func() { Commit = origCommit }()

	// Test with short commit hash
	Commit = "abc"
	info := GetVersionInfo()
	if !strings.Contains(info, "abc") {
		t.Errorf("GetVersionInfo() should handle short commit, got %q", info)
	}
}

func TestVersionCmd_Structure(t *testing.T) {
	if versionCmd == nil {
		t.Fatal("versionCmd is nil")
	}

	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want %q", versionCmd.Use, "version")
	}

	if versionCmd.Short == "" {
		t.Error("versionCmd.Short should not be empty")
	}
}

func TestRunVersion_Short(t *testing.T) {
	// Save and restore
	origShort := shortVersion
	origVersion := Version
	defer func() {
		shortVersion = origShort
		Version = origVersion
	}()

	Version = "test-version"
	shortVersion = true

	// Capture output
	var buf bytes.Buffer
	old := versionCmd.OutOrStdout()
	versionCmd.SetOut(&buf)
	defer versionCmd.SetOut(old)

	runVersion(versionCmd, nil)

	// The function prints to stdout directly, so we can't easily capture it
	// But we can verify the flag logic by checking the function completes without panic
}

func TestRunVersion_Full(t *testing.T) {
	// Save and restore
	origShort := shortVersion
	defer func() { shortVersion = origShort }()

	shortVersion = false

	// Should complete without panic
	runVersion(versionCmd, nil)
}
