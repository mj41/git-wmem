package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPerformance_CheckModifiedFiles tests the performance improvement in checkModifiedFiles
// This test creates a repository with multiple files and measures performance
func TestPerformance_CheckModifiedFiles(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)
	workDir := h.TempDir()

	// Create test project with multiple files
	projectPath := filepath.Join(workDir, "perf-test-project")
	h.MkdirAll(projectPath)
	h.SetWorkDir(projectPath)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init")

	// Create 50 files (moderate size for testing)
	for i := 0; i < 50; i++ {
		filename := fmt.Sprintf("file_%03d.txt", i)
		content := fmt.Sprintf("content of file %d\nline 2\nline 3", i)
		h.WriteFile(filename, content)
	}

	// Create some nested directories with files
	h.MkdirAll("subdir1")
	h.MkdirAll("subdir2/nested")
	for i := 0; i < 20; i++ {
		filename := fmt.Sprintf("subdir1/nested_file_%03d.txt", i)
		h.WriteFile(filename, fmt.Sprintf("nested content %d", i))
	}
	for i := 0; i < 15; i++ {
		filename := fmt.Sprintf("subdir2/nested/deep_file_%03d.txt", i)
		h.WriteFile(filename, fmt.Sprintf("deep nested content %d", i))
	}

	_, err = h.RunGit("add", ".")
	h.AssertCommandSuccess("", err, "git add .")
	_, err = h.RunGit("commit", "-m", "Initial commit with 85 files")
	h.AssertCommandSuccess("", err, "git commit")

	// Setup wmem repo for commit
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../perf-test-project")

	// First wmem commit - measure time
	start := time.Now()
	output, err := h.RunGitWmem("commit")
	firstCommitTime := time.Since(start)
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	t.Logf("First commit with 85 files took: %v", firstCommitTime)

	// Second wmem commit with no changes - should be very fast due to optimization
	start = time.Now()
	output, err = h.RunGitWmem("commit")
	secondCommitTime := time.Since(start)
	h.AssertCommandSuccess(output, err, "second git-wmem-commit (no changes)")

	t.Logf("Second commit (no changes) took: %v", secondCommitTime)

	// The optimization should make the second commit significantly faster
	// because it detects no working directory changes quickly
	if !strings.Contains(output, "No modified files") {
		t.Errorf("Expected 'No modified files' message in output, got: %s", output)
	}

	// Make a small change and measure performance
	h.SetWorkDir(projectPath)
	h.WriteFile("new_file.txt", "new content")

	h.SetWorkDir(wmemDir)
	start = time.Now()
	output, err = h.RunGitWmem("commit")
	thirdCommitTime := time.Since(start)
	h.AssertCommandSuccess(output, err, "third git-wmem-commit (with change)")

	t.Logf("Third commit (with 1 new file) took: %v", thirdCommitTime)

	// Log performance summary
	t.Logf("Performance summary:")
	t.Logf("  Initial commit (85 files): %v", firstCommitTime)
	t.Logf("  No changes commit: %v", secondCommitTime)
	t.Logf("  Single file change: %v", thirdCommitTime)

	// Verify that the optimization worked - no changes should be much faster
	if secondCommitTime > firstCommitTime/2 {
		t.Logf("Warning: No-changes commit not significantly faster than initial commit")
		t.Logf("This might indicate the optimization could be improved further")
	} else {
		t.Logf("SUCCESS: No-changes commit is significantly faster (optimization working)")
	}
}

// TestPerformance_FileSystemStateComparison tests the filesystem state comparison performance
func TestPerformance_FileSystemStateComparison(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)
	workDir := h.TempDir()

	// Create test project
	projectPath := filepath.Join(workDir, "fs-comparison-test")
	h.MkdirAll(projectPath)
	h.SetWorkDir(projectPath)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init")

	// Create initial files
	for i := 0; i < 30; i++ {
		h.WriteFile(fmt.Sprintf("initial_file_%03d.txt", i), fmt.Sprintf("initial content %d", i))
	}

	_, err = h.RunGit("add", ".")
	h.AssertCommandSuccess("", err, "git add .")
	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit")

	// Setup wmem
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../fs-comparison-test")

	// First wmem commit
	start := time.Now()
	output, err := h.RunGitWmem("commit")
	baselineTime := time.Since(start)
	h.AssertCommandSuccess(output, err, "baseline commit")

	// Delete a file from filesystem (but not from git)
	h.SetWorkDir(projectPath)
	err = os.Remove(filepath.Join(projectPath, "initial_file_010.txt"))
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Add a new file to filesystem
	h.WriteFile("new_filesystem_file.txt", "new content")

	// Commit with filesystem changes - measure performance
	h.SetWorkDir(wmemDir)
	start = time.Now()
	output, err = h.RunGitWmem("commit")
	filesystemChangeTime := time.Since(start)
	h.AssertCommandSuccess(output, err, "commit with filesystem changes")

	t.Logf("Filesystem state comparison performance:")
	t.Logf("  Baseline commit: %v", baselineTime)
	t.Logf("  With FS changes: %v", filesystemChangeTime)

	// Verify the changes were detected and committed
	repoPath := filepath.Join(wmemDir, "repos/fs-comparison-test.git")
	h.SetWorkDir(repoPath)

	output, err = h.RunGit("ls-tree", "-r", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git ls-tree")

	// Should have new_filesystem_file.txt but not initial_file_010.txt
	if !strings.Contains(output, "new_filesystem_file.txt") {
		t.Errorf("Expected new_filesystem_file.txt in tree, got: %s", output)
	}
	if strings.Contains(output, "initial_file_010.txt") {
		t.Errorf("Unexpected initial_file_010.txt in tree (should be deleted), got: %s", output)
	}

	t.Logf("SUCCESS: Filesystem state changes correctly detected and committed")
}
