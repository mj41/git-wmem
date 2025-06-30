package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestHelper provides common utilities for git-wmem e2e tests
type TestHelper struct {
	t       *testing.T
	tempDir string
	workDir string
}

// NewTestHelper creates a new test helper with temporary directory
// Reference: Implements temp file pattern from general requirements
func NewTestHelper(t *testing.T) *TestHelper {
	timestamp := time.Now().Format("060102-150405")
	randomSuffix := fmt.Sprintf("%x", time.Now().UnixNano()%0xFFFFFF)
	tmpSubDir := fmt.Sprintf("%s-%s", timestamp, randomSuffix)

	tempDir := filepath.Join("/tmp/git-wmem-e2e", tmpSubDir)
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	return &TestHelper{
		t:       t,
		tempDir: tempDir,
		workDir: tempDir,
	}
}

// Cleanup removes the temporary directory
func (h *TestHelper) Cleanup() {
	if h.tempDir != "" {
		os.RemoveAll(h.tempDir)
	}
}

// TempDir returns the temporary directory path
func (h *TestHelper) TempDir() string {
	return h.tempDir
}

// SetWorkDir changes the working directory for subsequent commands
func (h *TestHelper) SetWorkDir(dir string) {
	h.workDir = dir
}

// RunCommand executes a command in the current working directory
func (h *TestHelper) RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = h.workDir

	output, err := cmd.CombinedOutput()
	h.t.Logf("Command: %s %s", name, strings.Join(args, " "))
	h.t.Logf("Dir: %s", h.workDir)
	h.t.Logf("Output: %s", string(output))
	if err != nil {
		h.t.Logf("Error: %v", err)
	}

	return string(output), err
}

// RunGit executes a git command in the current working directory
func (h *TestHelper) RunGit(args ...string) (string, error) {
	return h.RunCommand("git", args...)
}

// RunGitWmem executes a git-wmem command in the current working directory
func (h *TestHelper) RunGitWmem(command string, args ...string) (string, error) {
	fullArgs := append([]string{command}, args...)
	return h.RunCommand("git-wmem", fullArgs...)
}

// AssertFileExists checks if a file exists
func (h *TestHelper) AssertFileExists(filePath string) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(h.workDir, filePath)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		h.t.Errorf("Expected file to exist: %s", filePath)
	}
}

// AssertFileContains checks if a file contains expected content
func (h *TestHelper) AssertFileContains(filePath, expectedContent string) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(h.workDir, filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		h.t.Errorf("Failed to read file %s: %v", filePath, err)
		return
	}

	if !strings.Contains(string(content), expectedContent) {
		h.t.Errorf("File %s does not contain expected content: %s\nActual content: %s",
			filePath, expectedContent, string(content))
	}
}

// AssertFileEquals checks if a file has exact expected content
func (h *TestHelper) AssertFileEquals(filePath, expectedContent string) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(h.workDir, filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		h.t.Errorf("Failed to read file %s: %v", filePath, err)
		return
	}

	if strings.TrimSpace(string(content)) != strings.TrimSpace(expectedContent) {
		h.t.Errorf("File %s content mismatch.\nExpected: %s\nActual: %s",
			filePath, expectedContent, string(content))
	}
}

// AssertDirExists checks if a directory exists
func (h *TestHelper) AssertDirExists(dirPath string) {
	if !filepath.IsAbs(dirPath) {
		dirPath = filepath.Join(h.workDir, dirPath)
	}

	if stat, err := os.Stat(dirPath); os.IsNotExist(err) || !stat.IsDir() {
		h.t.Errorf("Expected directory to exist: %s", dirPath)
	}
}

// WriteFile writes content to a file
func (h *TestHelper) WriteFile(filePath, content string) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(h.workDir, filePath)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		h.t.Fatalf("Failed to write file %s: %v", filePath, err)
	}
}

// AppendToFile appends content to a file
func (h *TestHelper) AppendToFile(filePath, content string) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(h.workDir, filePath)
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		h.t.Fatalf("Failed to open file %s: %v", filePath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content + "\n"); err != nil {
		h.t.Fatalf("Failed to append to file %s: %v", filePath, err)
	}
}

// MkdirAll creates directories
func (h *TestHelper) MkdirAll(dirPath string) {
	if !filepath.IsAbs(dirPath) {
		dirPath = filepath.Join(h.workDir, dirPath)
	}

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		h.t.Fatalf("Failed to create directory %s: %v", dirPath, err)
	}
}

// AssertCommandSuccess checks that a command executed successfully
func (h *TestHelper) AssertCommandSuccess(output string, err error, context string) {
	if err != nil {
		h.t.Errorf("%s failed: %v\nOutput: %s", context, err, output)
	}
}

// AssertOutputContains checks that command output contains expected text
func (h *TestHelper) AssertOutputContains(output, expectedContent string) {
	if !strings.Contains(output, expectedContent) {
		h.t.Errorf("Output does not contain expected content: %s\nActual output: %s",
			expectedContent, output)
	}
}

// AssertCommandError checks that a command failed with expected error
func (h *TestHelper) AssertCommandError(output string, err error, expectedError string, context string) {
	if err == nil {
		h.t.Errorf("%s should have failed but succeeded\nOutput: %s", context, output)
		return
	}

	if !strings.Contains(output, expectedError) && !strings.Contains(err.Error(), expectedError) {
		h.t.Errorf("%s failed but with unexpected error.\nExpected: %s\nActual error: %v\nOutput: %s",
			context, expectedError, err, output)
	}
}
