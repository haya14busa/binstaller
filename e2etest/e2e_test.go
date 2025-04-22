package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var binstallerPath string

// TestMain builds the binstaller binary once before running all tests
func TestMain(m *testing.M) {
	// Create a temporary directory for the binstaller binary
	tempDir, err := os.MkdirTemp("", "binstaller-test")
	if err != nil {
		panic("Failed to create temp directory: " + err.Error())
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			panic("Failed to remove temp directory: " + err.Error())
		}
	}()

	// Build the binstaller tool to a temporary location
	execName := "binst"
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}
	binstallerPath = filepath.Join(tempDir, execName)
	cmd := exec.Command("go", "build", "-o", binstallerPath, "./cmd/binst")
	cmd.Dir = ".." // Go up one level to reach the root directory
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("Failed to build binstaller: " + err.Error())
	}

	// Run the tests
	os.Exit(m.Run())
}

// testInstallScript tests that the binstaller tool can generate a working
// installation script for the specified repository and that the script
// can successfully install the binary.
func testInstallScript(t *testing.T, repo, binaryName, versionFlag string) {
	// Create a temporary directory for all test artifacts
	tempDir := t.TempDir()

	// Init binstaller config
	configPath := filepath.Join(tempDir, binaryName+".binstaller.yml")
	initCmd := exec.Command(binstallerPath, "init", "--verbose", "--source=goreleaser", "--repo", repo, "-o", configPath)
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init binstaller config: %v", err)
	}

	// Generate installer script
	installerPath := filepath.Join(tempDir, binaryName+".binstaller.yml")
	genCmd := exec.Command(binstallerPath, "gen", "--config", configPath, "-o", installerPath)
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		t.Fatalf("Failed to generate installation script: %v", err)
	}

	// Create a temporary bin directory
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Run the installation script
	var stderr bytes.Buffer
	var installStdout bytes.Buffer
	installCmd := exec.Command("sh", installerPath, "-b", binDir, "-d")
	installCmd.Stderr = &stderr
	installCmd.Stdout = &installStdout
	if err := installCmd.Run(); err != nil {
		t.Fatalf("Failed to run installation script: %v\nStdout: %s\nStderr: %s", err, installStdout.String(), stderr.String())
	}

	// Check that the binary was installed
	binName := binaryName
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binaryPath := filepath.Join(binDir, binName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("%s binary was not installed at %s", binName, binaryPath)
	}

	// Check that the binary works
	var stdout bytes.Buffer
	stderr.Reset()
	versionCmd := exec.Command(binaryPath, versionFlag)
	versionCmd.Stdout = &stdout
	versionCmd.Stderr = &stderr
	if err := versionCmd.Run(); err != nil {
		t.Fatalf("Failed to run %s %s: %v", binaryName, versionFlag, err)
	}

	output := stdout.String()
	stderrOutput := stderr.String()
	if output == "" && stderrOutput == "" {
		t.Fatalf("%s %s returned empty output", binaryName, versionFlag)
	}

	t.Logf("Successfully installed and ran %s with %s flag", binaryName, versionFlag)
}

func TestReviewdogE2E(t *testing.T) {
	testInstallScript(t, "reviewdog/reviewdog", "reviewdog", "-version")
}

func TestGoreleaserE2E(t *testing.T) {
	testInstallScript(t, "goreleaser/goreleaser", "goreleaser", "--version")
}

func TestGhSetupE2E(t *testing.T) {
	testInstallScript(t, "k1LoW/gh-setup", "gh-setup", "--help")
}

func TestSigspyE2E(t *testing.T) {
	testInstallScript(t, "actionutils/sigspy", "sigspy", "--help")
}

func TestGolangciLintE2E(t *testing.T) {
	testInstallScript(t, "golangci/golangci-lint", "golangci-lint", "--version")
}
