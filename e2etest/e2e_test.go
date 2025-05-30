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
func testInstallScript(t *testing.T, repo, binaryName, versionFlag, sha string) {
	// Create a temporary directory for all test artifacts
	tempDir := t.TempDir()

	// Init binstaller config
	configPath := filepath.Join(tempDir, binaryName+".binstaller.yml")
	initCmd := exec.Command(binstallerPath, "init", "--verbose", "--source=goreleaser", "--repo", repo, "-o", configPath, "--sha", sha)
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init binstaller config: %v", err)
	}

	// Check that the config file content
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	// Log the config file content
	t.Logf("Config file content:\n%s", configContent)

	// Generate installer script
	installerPath := filepath.Join(tempDir, binaryName+".binstaller.sh")
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
	testInstallScript(t, "reviewdog/reviewdog", "reviewdog", "-version", "7e05fa3e78ba7f2be4999ca2d35b00a3fd92a783")
}

func TestGoreleaserE2E(t *testing.T) {
	testInstallScript(t, "goreleaser/goreleaser", "goreleaser", "--version", "79c76c229d50ca45ef77afa1745df0a0e438d237")
}

func TestGhSetupE2E(t *testing.T) {
	testInstallScript(t, "k1LoW/gh-setup", "gh-setup", "--help", "a2359e4bcda8af5d7e16e1b3fb0eeec1be267e63")
}

func TestSigspyE2E(t *testing.T) {
	testInstallScript(t, "actionutils/sigspy", "sigspy", "--help", "3e1c6f32072cd4b8309d00bd31f498903f71c422")
}

func TestGolangciLintE2E(t *testing.T) {
	testInstallScript(t, "golangci/golangci-lint", "golangci-lint", "--version", "6d2a94be6b20f1c06e95d79479c6fdc34a69c45f")
}
