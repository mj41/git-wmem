package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupBasicWmemRepo creates a basic wmem repository for testing
// Reference: docs/use-cases/git-wmem-init/basic.md#main-scenario
func setupBasicWmemRepo(h *TestHelper) string {
	h.SetWorkDir(h.TempDir())
	output, err := h.RunGitWmem("init", "my-wmem1")
	h.AssertCommandSuccess(output, err, "git-wmem-init my-wmem1")

	wmemDir := filepath.Join(h.TempDir(), "my-wmem1")
	h.SetWorkDir(wmemDir)
	return wmemDir
}

// setupTestProjects creates test projects as described in use cases
// Reference: docs/use-cases/user-sh-cmds/wds-setup-basic.md#main-scenario
func setupTestProjects(h *TestHelper) (string, string) {
	workDir := h.TempDir()

	// Create project A
	projectA := filepath.Join(workDir, "my-projectA")
	h.MkdirAll(projectA)
	h.SetWorkDir(projectA)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init projectA")

	h.WriteFile("fileA.txt", "file A content")
	_, err = h.RunGit("add", "fileA.txt")
	h.AssertCommandSuccess("", err, "git add fileA.txt")

	_, err = h.RunGit("commit", "-m", "Initial commit in my-projectA")
	h.AssertCommandSuccess("", err, "git commit projectA")

	// Create project B
	projectB := filepath.Join(workDir, "my-projectB")
	h.MkdirAll(projectB)
	h.SetWorkDir(projectB)

	_, err = h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init projectB")

	h.WriteFile("fileB.txt", "file B content")
	_, err = h.RunGit("add", "fileB.txt")
	h.AssertCommandSuccess("", err, "git add fileB.txt")

	_, err = h.RunGit("commit", "-m", "Initial commit in my-projectB")
	h.AssertCommandSuccess("", err, "git commit projectB")

	return projectA, projectB
}

// TestGitWmemCommit_Basic tests basic git-wmem-commit functionality
// Reference: docs/use-cases/git-wmem-commit/basic.md#main-scenario
func TestGitWmemCommit_Basic(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test projects
	wmemDir := setupBasicWmemRepo(h)
	_, _ = setupTestProjects(h)

	// Setup workdir paths
	// Reference: docs/use-cases/user-sh-cmds/wds-setup-basic.md#main-scenario step 2
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	// Run git-wmem-commit
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "git-wmem-commit")

	// Verify workdir-map.json was updated
	// Reference: docs/data-structures.md#workdir-map
	workdirMapPath := "md-internal/workdir-map.json"
	content, err := os.ReadFile(filepath.Join(wmemDir, workdirMapPath))
	if err != nil {
		t.Fatalf("Failed to read workdir-map.json: %v", err)
	}

	var workdirMap map[string]string
	if err := json.Unmarshal(content, &workdirMap); err != nil {
		t.Fatalf("Failed to parse workdir-map.json: %v", err)
	}

	if workdirMap["my-projectA"] != "../my-projectA" {
		t.Errorf("Expected my-projectA mapping, got: %v", workdirMap)
	}
	if workdirMap["my-projectB"] != "../my-projectB" {
		t.Errorf("Expected my-projectB mapping, got: %v", workdirMap)
	}

	// Verify bare repositories were created
	// Reference: docs/use-cases/git-wmem-commit/basic.md#uc-git-wmem-commit-init-repos
	h.AssertDirExists("repos/my-projectA.git")
	h.AssertDirExists("repos/my-projectB.git")

	// Verify wmem-repo commit was created
	// Reference: docs/use-cases/git-wmem-commit/basic.md#uc-git-wmem-commit-commit-all
	output, err = h.RunGit("log", "--oneline", "-1")
	h.AssertCommandSuccess(output, err, "git log")

	// Should contain wmem-uid in commit message
	// Reference: docs/data-structures.md#commit-info
	if !strings.Contains(output, "wmem-") {
		t.Errorf("Expected wmem-uid in commit message, got: %s", output)
	}
}

// TestGitWmemCommit_WithFileChanges tests commit with actual file changes
// Reference: docs/use-cases/user-sh-cmds/wds-file-changes.md
func TestGitWmemCommit_WithFileChanges(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test projects
	wmemDir := setupBasicWmemRepo(h)
	projectA, projectB := setupTestProjects(h)

	// Setup workdir paths and commit first time
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "initial git-wmem-commit")

	// Make file changes as described in use case
	// Reference: docs/use-cases/user-sh-cmds/wds-file-changes.md step 1
	h.SetWorkDir(projectA)
	h.WriteFile("file-featX1.txt", "file file-featX1.txt: content A-X-pre-a, line 1")

	h.SetWorkDir(projectB)
	_, err = h.RunGit("checkout", "-b", "workH")
	h.AssertCommandSuccess("", err, "git checkout -b workH")

	h.MkdirAll("workH-dir")
	h.WriteFile("workH-dir/file-workH1.txt", "file workH-dir/file-workH1.txt: content B-W-pre-a, line 1")

	// Commit changes in wmem
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "git-wmem-commit with file changes")

	// Verify commit was created
	output, err = h.RunGit("log", "--oneline", "-2")
	h.AssertCommandSuccess(output, err, "git log after file changes")

	// Should have 2 commits now (initial + file changes)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 commits, got %d: %s", len(lines), output)
	}
}

// TestGitWmemCommit_WithGitCommands tests commit with git commands in workdirs
// Reference: docs/use-cases/user-sh-cmds/wds-git-cmds.md
func TestGitWmemCommit_WithGitCommands(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test projects
	wmemDir := setupBasicWmemRepo(h)
	projectA, projectB := setupTestProjects(h)

	// Setup workdir paths and commit first time
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "initial git-wmem-commit")

	// Make git commits as described in use case
	// Reference: docs/use-cases/user-sh-cmds/wds-git-cmds.md step 1
	h.SetWorkDir(projectA)
	_, err = h.RunGit("checkout", "-b", "feat/X1")
	h.AssertCommandSuccess("", err, "git checkout -b feat/X1")

	h.WriteFile("file-featX1.txt", "file file-featX1.txt: content A-X-a, line 1")
	_, err = h.RunGit("add", "file-featX1.txt")
	h.AssertCommandSuccess("", err, "git add file-featX1.txt")

	_, err = h.RunGit("commit", "-m", "Project my-projectA, feature X, commit A-X-a")
	h.AssertCommandSuccess("", err, "git commit A-X-a")

	h.AppendToFile("file-featX1.txt", "file file-featX1.txt: content A-X-b, line 2")
	_, err = h.RunGit("commit", "-a", "-m", "Project my-projectA, feature X, commit A-X-b")
	h.AssertCommandSuccess("", err, "git commit A-X-b")

	h.SetWorkDir(projectB)
	_, err = h.RunGit("checkout", "-b", "workH")
	h.AssertCommandSuccess("", err, "git checkout -b workH")

	h.MkdirAll("workH-dir")
	h.WriteFile("workH-dir/file-workH1.txt", "file workH-dir/file-workH1.txt: content B-W-a, line 1")
	_, err = h.RunGit("add", "workH-dir/file-workH1.txt")
	h.AssertCommandSuccess("", err, "git add workH-dir/file-workH1.txt")

	_, err = h.RunGit("commit", "-m", "Project my-projectB, feature W, commit B-W-a")
	h.AssertCommandSuccess("", err, "git commit B-W-a")

	// Set commit message prefix
	// Reference: docs/use-cases/user-sh-cmds/commit-msg-prefix-file.md
	h.SetWorkDir(wmemDir)
	h.WriteFile("md/commit/msg-prefix", "projA and projB features")

	// Commit changes in wmem
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "git-wmem-commit with git commands")

	// Verify commit message contains prefix
	output, err = h.RunGit("log", "--oneline", "-1")
	h.AssertCommandSuccess(output, err, "git log after git commands")

	if !strings.Contains(output, "projA and projB features") {
		t.Errorf("Expected commit message prefix in log output, got: %s", output)
	}
}

// TestGitWmemCommit_ErrorNoWorkdirs tests error when no workdirs configured
// Reference: docs/use-cases/git-wmem-commit/basic.md#alternatives 1b
func TestGitWmemCommit_ErrorNoWorkdirs(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo but no workdirs
	wmemDir := setupBasicWmemRepo(h)
	h.SetWorkDir(wmemDir)

	// Try to commit without workdirs
	output, err := h.RunGitWmem("commit")
	h.AssertCommandError(output, err, "No workdirs configured for commit", "git-wmem-commit without workdirs")
}

// TestGitWmemCommit_ErrorNotInWmemRepo tests error when not in wmem repo
// Reference: docs/use-cases/git-wmem-commit/basic.md preconditions
func TestGitWmemCommit_ErrorNotInWmemRepo(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Try to commit outside wmem repo
	h.SetWorkDir(h.TempDir())
	output, err := h.RunGitWmem("commit")
	h.AssertCommandError(output, err, ".git-wmem", "git-wmem-commit outside wmem repo")
}

// TestGitWmemCommit_SkipsSubdirectoriesWithGitRepos tests that subdirectories with .git are handled as gitlinks
// Reference: docs/use-cases/git-wmem-commit/basic.md step 7 detail - should work "like git add -A"
func TestGitWmemCommit_SkipsSubdirectoriesWithGitRepos(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)
	workDir := h.TempDir()

	// Create main project
	projectMain := filepath.Join(workDir, "main-project")
	h.MkdirAll(projectMain)
	h.SetWorkDir(projectMain)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init main-project")

	// Add main project files
	h.WriteFile("main-file.txt", "main project content")
	_, err = h.RunGit("add", "main-file.txt")
	h.AssertCommandSuccess("", err, "git add main-file.txt")
	_, err = h.RunGit("commit", "-m", "Initial commit in main-project")
	h.AssertCommandSuccess("", err, "git commit main-project")

	// Create subdirectory with its own git repository (should be ignored)
	subProjectWithGit := filepath.Join(projectMain, "nested-git-project")
	h.MkdirAll(subProjectWithGit)
	h.SetWorkDir(subProjectWithGit)

	_, err = h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init nested-git-project")

	h.WriteFile("nested-file.txt", "nested project content")
	_, err = h.RunGit("add", "nested-file.txt")
	h.AssertCommandSuccess("", err, "git add nested-file.txt")
	_, err = h.RunGit("commit", "-m", "Initial commit in nested-git-project")
	h.AssertCommandSuccess("", err, "git commit nested-git-project")

	// Create regular subdirectory (should be included)
	regularSubdir := filepath.Join(projectMain, "regular-subdir")
	h.MkdirAll(regularSubdir)
	h.WriteFile(filepath.Join(regularSubdir, "regular-file.txt"), "regular subdirectory content")

	// Create files in main project directory
	h.SetWorkDir(projectMain)
	h.WriteFile("additional-file.txt", "additional content")

	// Setup wmem repo for commit
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../main-project")

	// Perform commit
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "git-wmem-commit with nested git repos")

	// Verify the bare repository exists
	h.AssertDirExists("repos/main-project.git")
	// Check that the tree structure was created correctly by examining the committed files
	// We should have main-file.txt, additional-file.txt, regular-subdir/regular-file.txt
	// and nested-git-project as a gitlink (like git add -A does)

	repoPath := filepath.Join(wmemDir, "repos/main-project.git")
	h.SetWorkDir(repoPath)

	// List the tree contents of the HEAD commit to verify structure
	output, err = h.RunGit("ls-tree", "-r", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git ls-tree")

	// Verify expected files are present
	if !strings.Contains(output, "main-file.txt") {
		t.Errorf("Expected main-file.txt in tree, got: %s", output)
	}
	if !strings.Contains(output, "additional-file.txt") {
		t.Errorf("Expected additional-file.txt in tree, got: %s", output)
	}
	if !strings.Contains(output, "regular-subdir/regular-file.txt") {
		t.Errorf("Expected regular-subdir/regular-file.txt in tree, got: %s", output)
	}

	// Verify nested git project is present as gitlink (like git add -A does)
	if !strings.Contains(output, "nested-git-project") {
		t.Errorf("Expected nested-git-project as gitlink in tree (like git add -A), got: %s", output)
	}

	// Verify that nested git project files are NOT present individually
	if strings.Contains(output, "nested-git-project/nested-file.txt") {
		t.Errorf("Unexpected nested-git-project/nested-file.txt in tree (should be gitlink only), got: %s", output)
	}

	// Verify the nested-git-project is a gitlink (commit object, not blob)
	// We can check this by ensuring it doesn't have the "blob" type in ls-tree output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "nested-git-project") && !strings.Contains(line, "/") {
			// This should be the gitlink entry
			if strings.Contains(line, "blob") {
				t.Errorf("nested-git-project should be a commit (gitlink), not a blob, got: %s", line)
			}
			if !strings.Contains(line, "commit") {
				t.Errorf("nested-git-project should be a commit (gitlink), got: %s", line)
			}
		}
	}
}

// TestGitWmemCommit_RespectsGitignore tests that .gitignore rules are respected for nested git repos
// Reference: docs/use-cases/git-wmem-commit/basic.md step 7 detail - should work "like git add -A"
func TestGitWmemCommit_RespectsGitignore(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)
	workDir := h.TempDir()

	// Create main project
	projectMain := filepath.Join(workDir, "main-project")
	h.MkdirAll(projectMain)
	h.SetWorkDir(projectMain)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init main-project")

	// Add main project files
	h.WriteFile("main-file.txt", "main project content")
	_, err = h.RunGit("add", "main-file.txt")
	h.AssertCommandSuccess("", err, "git add main-file.txt")
	_, err = h.RunGit("commit", "-m", "Initial commit in main-project")
	h.AssertCommandSuccess("", err, "git commit main-project")

	// Create nested git repository
	nestedProjectWithGit := filepath.Join(projectMain, "nested-git-project")
	h.MkdirAll(nestedProjectWithGit)
	h.SetWorkDir(nestedProjectWithGit)

	_, err = h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init nested-git-project")

	h.WriteFile("nested-file.txt", "nested project content")
	_, err = h.RunGit("add", "nested-file.txt")
	h.AssertCommandSuccess("", err, "git add nested-file.txt")
	_, err = h.RunGit("commit", "-m", "Initial commit in nested-git-project")
	h.AssertCommandSuccess("", err, "git commit nested-git-project")

	// Create .gitignore file that ignores the nested git project
	h.SetWorkDir(projectMain)
	h.WriteFile(".gitignore", "nested-git-project/")

	// Create additional files
	h.WriteFile("additional-file.txt", "additional content")

	// Setup wmem repo for commit
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../main-project")

	// Perform commit
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "git-wmem-commit with gitignore")

	// Verify the bare repository exists
	h.AssertDirExists("repos/main-project.git")

	// Check that the ignored nested git repo is NOT included
	repoPath := filepath.Join(wmemDir, "repos/main-project.git")
	h.SetWorkDir(repoPath)

	output, err = h.RunGit("ls-tree", "-r", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git ls-tree")

	// Verify expected files are present
	if !strings.Contains(output, "main-file.txt") {
		t.Errorf("Expected main-file.txt in tree, got: %s", output)
	}
	if !strings.Contains(output, "additional-file.txt") {
		t.Errorf("Expected additional-file.txt in tree, got: %s", output)
	}
	if !strings.Contains(output, ".gitignore") {
		t.Errorf("Expected .gitignore in tree, got: %s", output)
	}

	// Verify nested git project is NOT present (ignored by .gitignore)
	if strings.Contains(output, "nested-git-project") {
		t.Errorf("Unexpected nested-git-project in tree (should be ignored by .gitignore), got: %s", output)
	}
}

// TestCommitWorkdir_FileSystemStateComparison tests that file system state is compared with wmem-tracked state
// This tests the new behavior where files deleted from filesystem are detected and committed
// Reference: docs/use-cases/git-wmem-commit/basic.md#uc-sync-workdir step 6
func TestCommitWorkdir_FileSystemStateComparison(t *testing.T) {
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
	h.WriteFile("file1.txt", "file 1 content")
	h.WriteFile("file2.txt", "file 2 content")
	h.WriteFile("file3.txt", "file 3 content")
	_, err = h.RunGit("add", ".")
	h.AssertCommandSuccess("", err, "git add initial files")
	_, err = h.RunGit("commit", "-m", "Initial commit with 3 files")
	h.AssertCommandSuccess("", err, "git commit initial files")

	// Setup wmem repo for commit
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../test-project")

	// First wmem commit - should capture all 3 files
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Verify all files are tracked in wmem-br/main
	repoPath := filepath.Join(wmemDir, "repos/test-project.git")
	h.SetWorkDir(repoPath)

	output, err = h.RunGit("ls-tree", "-r", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git ls-tree after first commit")

	if !strings.Contains(output, "file1.txt") {
		t.Errorf("Expected file1.txt in tree, got: %s", output)
	}
	if !strings.Contains(output, "file2.txt") {
		t.Errorf("Expected file2.txt in tree, got: %s", output)
	}
	if !strings.Contains(output, "file3.txt") {
		t.Errorf("Expected file3.txt in tree, got: %s", output)
	}

	// Delete file2.txt from filesystem (but NOT from git repo)
	// This simulates the scenario where user deletes a file that was previously wmem-committed
	h.SetWorkDir(projectPath)
	err = os.Remove(filepath.Join(projectPath, "file2.txt"))
	if err != nil {
		t.Fatalf("Failed to delete file2.txt: %v", err)
	}

	// Add a new file4.txt to filesystem
	h.WriteFile("file4.txt", "file 4 content")

	// Second wmem commit - should detect file2.txt deletion and file4.txt addition
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second git-wmem-commit with file deletion")

	// Verify the wmem-tracked state now reflects filesystem state
	h.SetWorkDir(repoPath)
	output, err = h.RunGit("ls-tree", "-r", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git ls-tree after second commit")

	// Should still have file1.txt and file3.txt
	if !strings.Contains(output, "file1.txt") {
		t.Errorf("Expected file1.txt in tree after deletion, got: %s", output)
	}
	if !strings.Contains(output, "file3.txt") {
		t.Errorf("Expected file3.txt in tree after deletion, got: %s", output)
	}

	// Should now have file4.txt
	if !strings.Contains(output, "file4.txt") {
		t.Errorf("Expected file4.txt in tree after addition, got: %s", output)
	}

	// Should NOT have file2.txt (was deleted from filesystem)
	if strings.Contains(output, "file2.txt") {
		t.Errorf("Unexpected file2.txt in tree after deletion, got: %s", output)
	}

	// Verify that workdir git repo still has file2.txt (we only deleted from filesystem, not from git)
	h.SetWorkDir(projectPath)
	output, err = h.RunGit("ls-files")
	h.AssertCommandSuccess(output, err, "git ls-files in workdir")

	// Workdir git should still track file2.txt because we only deleted it from filesystem
	if !strings.Contains(output, "file2.txt") {
		t.Errorf("Expected file2.txt to still be tracked in workdir git, got: %s", output)
	}

	// But file2.txt should show as deleted in git status
	output, err = h.RunGit("status", "--porcelain")
	h.AssertCommandSuccess(output, err, "git status in workdir")
	if !strings.Contains(output, "D file2.txt") {
		t.Errorf("Expected file2.txt to show as deleted in git status, got: %s", output)
	}
}

// TestCommitWorkdir_NoChangesDetection tests that no commit is created when filesystem matches wmem-tracked state
// Reference: docs/use-cases/git-wmem-commit/basic.md#alternatives 6b
func TestCommitWorkdir_NoChangesDetection(t *testing.T) {
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
	h.WriteFile("file1.txt", "file 1 content")
	_, err = h.RunGit("add", ".")
	h.AssertCommandSuccess("", err, "git add initial files")
	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit initial files")

	// Setup wmem repo for commit
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../test-project")

	// First wmem commit
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Second wmem commit with no changes - should skip workdir commit creation
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second git-wmem-commit with no changes")

	// Should contain message about no changes detected
	if !strings.Contains(output, "No modified files") {
		t.Errorf("Expected 'No modified files' message, got: %s", output)
	}

	// Verify that the workdir commit was skipped by checking the wmem-br/main branch in the bare repo
	repoPath := filepath.Join(wmemDir, "repos/test-project.git")
	h.SetWorkDir(repoPath)

	// Get the commit count on wmem-br/main - should still be 1 (no new workdir commits)
	output, err = h.RunGit("rev-list", "--count", "wmem-br/main")
	h.AssertCommandSuccess(output, err, "git rev-list count on wmem-br/main")
	wmemBrCommitCount := strings.TrimSpace(output)

	// Should still be 1 commit on wmem-br/main since no changes were detected
	if wmemBrCommitCount != "1" {
		t.Errorf("Expected 1 commit on wmem-br/main when no changes, got: %s", wmemBrCommitCount)
	}
}
