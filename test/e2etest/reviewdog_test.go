package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestReviewdogE2E tests that the goinstaller tool can generate a working
// installation script for the reviewdog repository and that the script
// can successfully install the reviewdog binary.
func TestReviewdogE2E(t *testing.T) {
	// Create a temporary directory for all test artifacts
	tempDir := t.TempDir()

	// Build the goinstaller tool to a temporary location
	goinstallerPath := filepath.Join(tempDir, "goinstaller")
	cmd := exec.Command("go", "build", "-o", goinstallerPath)
	cmd.Dir = "../.." // Go up two levels to reach the root directory
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build goinstaller: %v", err)
	}

	// Generate the installation script for reviewdog
	installerPath := filepath.Join(tempDir, "reviewdog-install.sh")
	cmd = exec.Command(goinstallerPath, "--repo=reviewdog/reviewdog")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate installation script: %v", err)
	}

	// Write the installation script to a file
	if err := os.WriteFile(installerPath, stdout.Bytes(), 0755); err != nil {
		t.Fatalf("Failed to write installation script: %v", err)
	}

	// Create a temporary bin directory
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Run the installation script
	cmd = exec.Command("sh", installerPath, "-b", binDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run installation script: %v\nStderr: %s", err, stderr.String())
	}

	// Check that the reviewdog binary was installed
	reviewdogPath := filepath.Join(binDir, "reviewdog")
	if _, err := os.Stat(reviewdogPath); os.IsNotExist(err) {
		t.Fatalf("reviewdog binary was not installed at %s", reviewdogPath)
	}

	// Check that the reviewdog binary works
	cmd = exec.Command(reviewdogPath, "-version")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run reviewdog -version: %v", err)
	}

	version := stdout.String()
	fmt.Printf("Installed reviewdog version: %s\n", version)
	if version == "" {
		t.Fatal("reviewdog -version returned empty output")
	}

	t.Logf("Successfully installed and ran reviewdog version: %s", version)
}
