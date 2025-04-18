package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReproducibleBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "goinstaller-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("failed to remove temp dir: %v", err)
		}
	}()

	// Test cases
	testCases := []struct {
		name            string
		args            []string
		expectSourceIn  bool
		expectVersionIn bool
	}{
		{
			name:            "Default",
			args:            []string{"--repo=k1LoW/gh-setup"},
			expectSourceIn:  true,
			expectVersionIn: true,
		},
		{
			name:            "WithCommit",
			args:            []string{"--repo=k1LoW/gh-setup", "--commit=2edb1d01b11f04c78bf3a0d023aefc65e9fb81e5"},
			expectSourceIn:  true,
			expectVersionIn: true,
		},
		{
			name:            "SkipSourceInfo",
			args:            []string{"--repo=k1LoW/gh-setup", "--skip-source-info"},
			expectSourceIn:  false,
			expectVersionIn: false,
		},
		{
			name:            "WithCommitAndSkipSourceInfo",
			args:            []string{"--repo=k1LoW/gh-setup", "--commit=2edb1d01b11f04c78bf3a0d023aefc65e9fb81e5", "--skip-source-info"},
			expectSourceIn:  false,
			expectVersionIn: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate the script
			outputFile := filepath.Join(tempDir, tc.name+".sh")
			args := append(tc.args, "--output="+outputFile)
			cmd := exec.Command(goinstallerPath, args...)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to run command: %v, stderr: %s", err, stderr.String())
			}

			// Read the generated script
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}
			contentStr := string(content)

			// Check if source information is included
			sourceInfoIncluded := strings.Contains(contentStr, "goinstaller (") && strings.Contains(contentStr, "from ")
			if tc.expectSourceIn && !sourceInfoIncluded {
				t.Errorf("expected source information to be included, but it wasn't")
			} else if !tc.expectSourceIn && sourceInfoIncluded {
				t.Errorf("expected source information to be excluded, but it was included")
			}

			// Check if version information is included
			versionInfoIncluded := strings.Contains(contentStr, "goinstaller (")
			if tc.expectVersionIn && !versionInfoIncluded {
				t.Errorf("expected version information to be included, but it wasn't")
			} else if !tc.expectVersionIn && versionInfoIncluded {
				t.Errorf("expected version information to be excluded, but it was included")
			}

			// Check for reproducibility by generating the script again
			outputFile2 := filepath.Join(tempDir, tc.name+"_2.sh")
			args2 := append(tc.args, "--output="+outputFile2)
			cmd2 := exec.Command(goinstallerPath, args2...)
			var stderr2 bytes.Buffer
			cmd2.Stderr = &stderr2
			if err := cmd2.Run(); err != nil {
				t.Fatalf("failed to run command: %v, stderr: %s", err, stderr2.String())
			}

			// Read the second generated script
			content2, err := os.ReadFile(outputFile2)
			if err != nil {
				t.Fatalf("failed to read second output file: %v", err)
			}

			// Compare the two scripts
			if !bytes.Equal(content, content2) {
				t.Errorf("scripts are not identical, not reproducible")
			}
		})
	}
}

func TestReproducibleBuildsWithDifferentCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "goinstaller-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("failed to remove temp dir: %v", err)
		}
	}()

	// Generate scripts with different commits
	commits := []string{
		"2edb1d01b11f04c78bf3a0d023aefc65e9fb81e5",
		"v1.11.0", // Using a tag as a commit reference
	}

	scripts := make([][]byte, len(commits))
	for i, commit := range commits {
		// Generate the script
		outputFile := filepath.Join(tempDir, "commit_"+commit+".sh")
		args := []string{"--repo=k1LoW/gh-setup", "--commit=" + commit, "--output=" + outputFile}
		cmd := exec.Command(goinstallerPath, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to run command: %v, stderr: %s", err, stderr.String())
		}

		// Read the generated script
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}
		scripts[i] = content
	}

	// Compare the scripts - they should be different because they're from different commits
	if bytes.Equal(scripts[0], scripts[1]) {
		t.Errorf("scripts from different commits are identical, expected them to be different")
	}

	// Now generate scripts with different commits but skip source info
	scriptsNoSource := make([][]byte, len(commits))
	for i, commit := range commits {
		// Generate the script
		outputFile := filepath.Join(tempDir, "commit_"+commit+"_no_source.sh")
		args := []string{"--repo=k1LoW/gh-setup", "--commit=" + commit, "--skip-source-info", "--output=" + outputFile}
		cmd := exec.Command(goinstallerPath, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to run command: %v, stderr: %s", err, stderr.String())
		}

		// Read the generated script
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}
		scriptsNoSource[i] = content
	}

	// If the scripts are identical with --skip-source-info, it means the only difference was the source info
	// This is the expected behavior if the repository hasn't changed its release structure
	if !bytes.Equal(scriptsNoSource[0], scriptsNoSource[1]) {
		t.Logf("scripts from different commits with --skip-source-info are different")
		// This is not necessarily an error, as the repository structure might have changed
		// between commits, but it's worth logging
	}
}
