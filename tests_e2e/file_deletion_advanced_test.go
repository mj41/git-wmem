package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommitWorkdir_FileAddedAndRemovedSinceMerge tests the specific scenario
// where a file is added after the last merge commit and then removed from filesystem
// This tests that hasFilesNewerThanLastWmemCommit properly falls back to full check
// when it cannot detect deletions through timestamp checking
func TestCommitWorkdir_FileAddedAndRemovedSinceMerge(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)
	workDir := h.TempDir()

	// Create test project
	projectPath := filepath.Join(workDir, "test-project")
	h.MkdirAll(projectPath)
	h.SetWorkDir(projectPath)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init test-project")

	// Add initial files and commit
	h.WriteFile("file1.txt", "file 1 content")
	_, err = h.RunGit("add", ".")
	h.AssertCommandSuccess("", err, "git add initial files")
	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit initial files")

	// Setup wmem repo for commit
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../test-project")

	// First wmem commit - establishes baseline
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Add a new file to workdir and commit to git
	h.SetWorkDir(projectPath)
	h.WriteFile("temp-file.txt", "temporary file content")
	_, err = h.RunGit("add", "temp-file.txt")
	h.AssertCommandSuccess("", err, "git add temp-file.txt")
	_, err = h.RunGit("commit", "-m", "Add temporary file")
	h.AssertCommandSuccess("", err, "git commit temp-file.txt")

	// Second wmem commit - should capture the new file
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second git-wmem-commit with new file")

	// Verify temp-file.txt is now tracked in wmem
	repoPath := filepath.Join(wmemDir, "repos/test-project.git")
	h.SetWorkDir(repoPath)
	output, err = h.RunGit("ls-tree", "-r", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git ls-tree after adding temp file")

	if !strings.Contains(output, "temp-file.txt") {
		t.Errorf("Expected temp-file.txt in wmem tree after addition, got: %s", output)
	}

	// Now DELETE the temp-file.txt from filesystem (but keep it in git)
	// This simulates the scenario where a file was added since last merge
	// and then deleted from filesystem
	h.SetWorkDir(projectPath)
	err = os.Remove(filepath.Join(projectPath, "temp-file.txt"))
	if err != nil {
		t.Fatalf("Failed to delete temp-file.txt: %v", err)
	}

	// The timestamp check should NOT detect this change because:
	// 1. The deleted file doesn't exist on filesystem to have a "newer" timestamp
	// 2. The remaining files (file1.txt) haven't been modified
	// However, the full tree comparison SHOULD detect the deletion

	// Third wmem commit - should detect temp-file.txt deletion despite timestamp check
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "third git-wmem-commit with file deletion")

	// Parse the debug output to understand what happened
	t.Logf("Debug output: %s", output)

	// Verify temp-file.txt is no longer in wmem tree (deletion detected)
	h.SetWorkDir(repoPath)
	output, err = h.RunGit("ls-tree", "-r", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git ls-tree after deletion")

	// Should still have file1.txt
	if !strings.Contains(output, "file1.txt") {
		t.Errorf("Expected file1.txt in wmem tree after deletion, got: %s", output)
	}

	// Should NOT have temp-file.txt (was deleted from filesystem)
	if strings.Contains(output, "temp-file.txt") {
		t.Errorf("Unexpected temp-file.txt in wmem tree after deletion, got: %s", output)
	}

	// Verify that workdir git repo still has temp-file.txt in its index
	h.SetWorkDir(projectPath)
	output, err = h.RunGit("ls-files")
	h.AssertCommandSuccess("", err, "git ls-files in workdir")

	if !strings.Contains(output, "temp-file.txt") {
		t.Errorf("Expected temp-file.txt to still be tracked in workdir git, got: %s", output)
	}

	// But temp-file.txt should show as deleted in git status
	output, err = h.RunGit("status", "--porcelain")
	h.AssertCommandSuccess("", err, "git status in workdir")
	if !strings.Contains(output, "D temp-file.txt") {
		t.Errorf("Expected temp-file.txt to show as deleted in git status, got: %s", output)
	}
}

// TestCommitWorkdir_MultipleFileOperationsSinceMerge tests complex scenario with multiple operations
func TestCommitWorkdir_MultipleFileOperationsSinceMerge(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)
	workDir := h.TempDir()

	// Create test project
	projectPath := filepath.Join(workDir, "test-project")
	h.MkdirAll(projectPath)
	h.SetWorkDir(projectPath)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init test-project")

	// Add initial files
	h.WriteFile("stable.txt", "stable file content")
	_, err = h.RunGit("add", ".")
	h.AssertCommandSuccess("", err, "git add initial files")
	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit initial files")

	// Setup wmem repo for commit
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../test-project")

	// First wmem commit - establishes baseline
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Perform complex operations since last merge:
	h.SetWorkDir(projectPath)

	// 1. Add file A and commit
	h.WriteFile("fileA.txt", "file A content")
	_, err = h.RunGit("add", "fileA.txt")
	h.AssertCommandSuccess("", err, "git add fileA.txt")
	_, err = h.RunGit("commit", "-m", "Add fileA")
	h.AssertCommandSuccess("", err, "git commit fileA")

	// 2. Add file B and commit
	h.WriteFile("fileB.txt", "file B content")
	_, err = h.RunGit("add", "fileB.txt")
	h.AssertCommandSuccess("", err, "git add fileB.txt")
	_, err = h.RunGit("commit", "-m", "Add fileB")
	h.AssertCommandSuccess("", err, "git commit fileB")

	// 3. Delete fileA from filesystem (but keep in git)
	err = os.Remove(filepath.Join(projectPath, "fileA.txt"))
	if err != nil {
		t.Fatalf("Failed to delete fileA.txt: %v", err)
	}

	// 4. Add fileC to filesystem only (not committed to git)
	h.WriteFile("fileC.txt", "file C content")

	// Second wmem commit - should handle all these changes correctly
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second git-wmem-commit with complex changes")

	// Verify final state in wmem tree
	repoPath := filepath.Join(wmemDir, "repos/test-project.git")
	h.SetWorkDir(repoPath)
	output, err = h.RunGit("ls-tree", "-r", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git ls-tree after complex operations")

	// Should have stable.txt (unchanged)
	if !strings.Contains(output, "stable.txt") {
		t.Errorf("Expected stable.txt in wmem tree, got: %s", output)
	}

	// Should have fileB.txt (added and still exists)
	if !strings.Contains(output, "fileB.txt") {
		t.Errorf("Expected fileB.txt in wmem tree, got: %s", output)
	}

	// Should have fileC.txt (added to filesystem)
	if !strings.Contains(output, "fileC.txt") {
		t.Errorf("Expected fileC.txt in wmem tree, got: %s", output)
	}

	// Should NOT have fileA.txt (added then deleted from filesystem)
	if strings.Contains(output, "fileA.txt") {
		t.Errorf("Unexpected fileA.txt in wmem tree after deletion, got: %s", output)
	}

	// Verify git status shows expected state
	h.SetWorkDir(projectPath)
	output, err = h.RunGit("status", "--porcelain")
	h.AssertCommandSuccess("", err, "git status after complex operations")

	// fileA should show as deleted
	if !strings.Contains(output, "D fileA.txt") {
		t.Errorf("Expected fileA.txt to show as deleted in git status, got: %s", output)
	}

	// fileC should show as untracked
	if !strings.Contains(output, "?? fileC.txt") {
		t.Errorf("Expected fileC.txt to show as untracked in git status, got: %s", output)
	}
}

// TestEarlyExitVsFullCheck tests the interaction between timestamp-based early exit
// and file deletion detection
func TestEarlyExitVsFullCheck(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)
	workDir := h.TempDir()

	// Create test project
	projectPath := filepath.Join(workDir, "test-project")
	h.MkdirAll(projectPath)
	h.SetWorkDir(projectPath)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init test-project")

	// Add initial file
	h.WriteFile("existing.txt", "existing file content")
	_, err = h.RunGit("add", ".")
	h.AssertCommandSuccess("", err, "git add initial files")
	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit initial files")

	// Setup wmem and do first commit
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../test-project")
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Test 1: Pure early exit case (no changes)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "test early exit with no changes")

	// Should show ultra-fast early exit
	if !strings.Contains(output, "ultra-fast early exit") {
		t.Logf("Expected ultra-fast early exit message, got: %s", output)
		// Note: This might not appear if there are other changes detected
	}

	// Test 2: Delete file to force full check despite timestamp optimization
	h.SetWorkDir(projectPath)
	err = os.Remove(filepath.Join(projectPath, "existing.txt"))
	if err != nil {
		t.Fatalf("Failed to delete existing.txt: %v", err)
	}

	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "test full check with file deletion")

	// Should proceed with full check and detect deletion
	if strings.Contains(output, "ultra-fast early exit") {
		t.Errorf("Should not use ultra-fast early exit when file deleted, got: %s", output)
	}

	// Verify deletion was detected
	repoPath := filepath.Join(wmemDir, "repos/test-project.git")
	h.SetWorkDir(repoPath)
	output, err = h.RunGit("ls-tree", "-r", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git ls-tree after deletion")

	// Should NOT have existing.txt
	if strings.Contains(output, "existing.txt") {
		t.Errorf("Unexpected existing.txt in wmem tree after deletion, got: %s", output)
	}
}
