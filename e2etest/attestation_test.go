package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	testRepo   = "actionutils/sigspy"
	testBinary = "sigspy"
)

// generateInstallScript generates an installation script for the given repository with the specified options
func generateInstallScript(t *testing.T, tempDir, repo string, options ...string) (string, error) {
	installerPath := filepath.Join(tempDir, testBinary+"-install.sh")

	// Build the command with the repository and any additional options
	args := []string{"--repo=" + repo}
	args = append(args, options...)

	var stdout, stderr bytes.Buffer
	generateCmd := exec.Command(goinstallerPath, args...)
	generateCmd.Stdout = &stdout
	generateCmd.Stderr = &stderr

	if err := generateCmd.Run(); err != nil {
		return "", err
	}

	// Write the installation script to a file
	if err := os.WriteFile(installerPath, stdout.Bytes(), 0755); err != nil {
		return "", err
	}

	return installerPath, nil
}

// createBinDir creates a temporary bin directory for installing binaries
func createBinDir(t *testing.T) string {
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	return binDir
}

// runInstallScript runs the installation script with the specified options
func runInstallScript(t *testing.T, installerPath, binDir string, options ...string) (string, error) {
	// Build the command with the bin directory and any additional options
	args := []string{installerPath, "-b", binDir, "-d"}
	args = append(args, options...)

	var stdout, stderr bytes.Buffer
	installCmd := exec.Command("sh", args...)
	installCmd.Stdout = &stdout
	installCmd.Stderr = &stderr

	err := installCmd.Run()

	// Combine stdout and stderr for easier analysis
	output := stderr.String() + stdout.String()

	return output, err
}

// verifyBinaryInstalled checks if the binary was installed correctly
func verifyBinaryInstalled(t *testing.T, binDir, binary string) {
	binaryPath := filepath.Join(binDir, binary)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("%s binary was not installed at %s", binary, binaryPath)
	}
}

// TestAttestationWithSigspy tests attestation verification with actionutils/sigspy
// which is a project that has GitHub attestations
func TestAttestationWithSigspy(t *testing.T) {
	// Skip if GitHub CLI is not installed
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("GitHub CLI not installed, skipping attestation tests")
	}

	// Create a temporary directory for all test artifacts
	tempDir := t.TempDir()

	// Generate the installation script with attestation verification
	installerPath, err := generateInstallScript(t, tempDir, testRepo,
		"--require-attestation", "--gh-attestation-verify-flags=--owner=actionutils")
	if err != nil {
		t.Fatalf("Failed to generate installation script: %v", err)
	}

	// Create a bin directory for installation
	binDir := createBinDir(t)

	// Run the installation script with attestation verification
	output, err := runInstallScript(t, installerPath, binDir, "-a")

	// We should see attestation-related messages in the output
	if !bytes.Contains([]byte(output), []byte("attestation")) {
		t.Fatalf("Expected output to mention attestation, but got: %s", output)
	}

	// The installation should succeed
	if err != nil {
		// If the GitHub CLI is not available, we can't run this test
		if bytes.Contains([]byte(output), []byte("GitHub CLI not available")) {
			t.Skip("Skipping test because GitHub CLI is not available")
		} else {
			t.Fatalf("Expected installation to succeed with attestation verification, but it failed: %v\nOutput: %s", err, output)
		}
	}

	// Verify the binary was installed
	verifyBinaryInstalled(t, binDir, testBinary)

	t.Logf("Successfully installed %s with attestation verification", testBinary)
}

// TestSkipAttestationWithSigspy tests skipping attestation verification with actionutils/sigspy
func TestSkipAttestationWithSigspy(t *testing.T) {
	// Create a temporary directory for all test artifacts
	tempDir := t.TempDir()

	// Generate the installation script with attestation verification skipped
	installerPath, err := generateInstallScript(t, tempDir, testRepo, "--skip-attestation")
	if err != nil {
		t.Fatalf("Failed to generate installation script: %v", err)
	}

	// Create a bin directory for installation
	binDir := createBinDir(t)

	// Run the installation script with attestation verification skipped
	output, err := runInstallScript(t, installerPath, binDir, "-s")

	// This should succeed since we're skipping attestation verification
	if err != nil {
		t.Fatalf("Failed to run installation script with attestation skipped: %v\nOutput: %s", err, output)
	}

	// Verify the binary was installed
	verifyBinaryInstalled(t, binDir, testBinary)

	t.Logf("Successfully installed %s with attestation verification skipped", testBinary)
}
