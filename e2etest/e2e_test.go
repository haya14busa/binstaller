package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var goinstallerPath string

// TestMain builds the goinstaller binary once before running all tests
func TestMain(m *testing.M) {
	// Create a temporary directory for the goinstaller binary
	tempDir, err := os.MkdirTemp("", "goinstaller-test")
	if err != nil {
		panic("Failed to create temp directory: " + err.Error())
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			panic("Failed to remove temp directory: " + err.Error())
		}
	}()

	// Build the goinstaller tool to a temporary location
	execName := "goinstaller"
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}
	goinstallerPath = filepath.Join(tempDir, execName)
	cmd := exec.Command("go", "build", "-o", goinstallerPath)
	cmd.Dir = ".." // Go up one level to reach the root directory
	if err := cmd.Run(); err != nil {
		panic("Failed to build goinstaller: " + err.Error())
	}

	// Run the tests
	os.Exit(m.Run())
}

// testInstallScript tests that the goinstaller tool can generate a working
// installation script for the specified repository and that the script
// can successfully install the binary.
func testInstallScript(t *testing.T, repo, binaryName, versionFlag string) {
	// Create a temporary directory for all test artifacts
	tempDir := t.TempDir()

	// Generate the installation script for the repository
	installerPath := filepath.Join(tempDir, binaryName+"-install.sh")
	var stdout bytes.Buffer
	generateCmd := exec.Command(goinstallerPath, "--repo="+repo)
	generateCmd.Stdout = &stdout
	if err := generateCmd.Run(); err != nil {
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
	var stderr bytes.Buffer
	var installStdout bytes.Buffer
	var installCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		installCmd = exec.Command("powershell", "-Command", installerPath, "-b", binDir, "-d")
	} else {
		installCmd = exec.Command("sh", installerPath, "-b", binDir, "-d")
	}
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
	stdout.Reset()
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

// TestReviewdogE2E tests that the goinstaller tool can generate a working
// installation script for the reviewdog repository and that the script
// can successfully install the reviewdog binary.
func TestReviewdogE2E(t *testing.T) {
	testInstallScript(t, "reviewdog/reviewdog", "reviewdog", "-version")
}

// TestGoreleaserE2E tests that the goinstaller tool can generate a working
// installation script for the goreleaser repository, which uses "main"
// as its default branch, and that the script can successfully install
// the goreleaser binary.
func TestGoreleaserE2E(t *testing.T) {
	testInstallScript(t, "goreleaser/goreleaser", "goreleaser", "--version")
}

// TestGhSetupE2E tests that the goinstaller tool can generate a working
// installation script for the k1LoW/gh-setup repository and that the script
// can successfully install the gh-setup binary.
func TestGhSetupE2E(t *testing.T) {
	testInstallScript(t, "k1LoW/gh-setup", "gh-setup", "--help")
}

// TestSigspyE2E tests that the goinstaller tool can generate a working
// installation script for the actionutils/sigspy repository and that the script
// can successfully install the sigspy binary.
func TestSigspyE2E(t *testing.T) {
	testInstallScript(t, "actionutils/sigspy", "sigspy", "--help")
}

// TestGolangciLintE2E tests that the goinstaller tool can generate a working
// installation script for the golangci/golangci-lint repository and that the script
// can successfully install the golangci-lint binary.
func TestGolangciLintE2E(t *testing.T) {
	testInstallScript(t, "golangci/golangci-lint", "golangci-lint", "--version")
}
