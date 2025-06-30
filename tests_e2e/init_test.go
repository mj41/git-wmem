package e2e

import (
	"path/filepath"
	"testing"
)

// TestGitWmemInit_Basic tests the basic git-wmem-init functionality
// Reference: docs/use-cases/git-wmem-init/basic.md#main-scenario
func TestGitWmemInit_Basic(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Step 1: User runs git-wmem-init my-wmem1
	// Reference: docs/use-cases/git-wmem-init/basic.md#main-scenario step 1
	h.SetWorkDir(h.TempDir())
	output, err := h.RunGitWmem("init", "my-wmem1")
	h.AssertCommandSuccess(output, err, "git-wmem-init my-wmem1")

	// Step 2: Verify directory structure was created
	// Reference: docs/use-cases/git-wmem-init/basic.md#main-scenario step 3
	wmemDir := filepath.Join(h.TempDir(), "my-wmem1")
	h.SetWorkDir(wmemDir)

	// Check marker file
	h.AssertFileExists(".git-wmem")

	// Check git repository
	h.AssertDirExists(".git")

	// Check gitignore file
	h.AssertFileExists(".gitignore")
	h.AssertFileContains(".gitignore", "repos/")

	// Check metadata directories
	h.AssertDirExists("md")
	h.AssertDirExists("md-internal")
	h.AssertDirExists("repos")

	// Check metadata files
	h.AssertFileExists("md/commit-workdir-paths")
	h.AssertFileExists("md/commit/msg-prefix")
	h.AssertFileExists("md/commit/author")
	h.AssertFileExists("md/commit/committer")
	h.AssertFileExists("md-internal/workdir-map.json")

	// Verify default author and committer
	// Reference: docs/use-cases/git-wmem-init/basic.md#main-scenario step 3
	h.AssertFileEquals("md/commit/author", "WMem Git <git-wmem@mj41.cz>")
	h.AssertFileEquals("md/commit/committer", "WMem Git <git-wmem@mj41.cz>")

	// Verify workdir-map.json is empty object
	h.AssertFileEquals("md-internal/workdir-map.json", "{}")

	// Step 4: Verify initial commit was created
	// Reference: docs/use-cases/git-wmem-init/basic.md#main-scenario step 4
	output, err = h.RunGit("log", "--oneline", "-1")
	h.AssertCommandSuccess(output, err, "git log")
	h.AssertOutputContains(output, "Initialize git-wmem repository `my-wmem1`")
}

// TestGitWmemInit_CurrentDirectory tests git-wmem-init with current directory
// Reference: docs/use-cases/git-wmem-init/basic.md#alternatives 1b
func TestGitWmemInit_CurrentDirectory(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Create and enter directory
	wmemDir := filepath.Join(h.TempDir(), "my-wmem1")
	h.MkdirAll(wmemDir)
	h.SetWorkDir(wmemDir)

	// Run git-wmem-init .
	output, err := h.RunGitWmem("init", ".")
	h.AssertCommandSuccess(output, err, "git-wmem-init .")

	// Verify structure was created
	h.AssertFileExists(".git-wmem")
	h.AssertDirExists(".git")
	h.AssertFileExists(".gitignore")
}

// TestGitWmemInit_ErrorDirectoryNotEmpty tests error when directory is not empty
// Reference: docs/use-cases/git-wmem-init/basic.md#alternatives 2b
func TestGitWmemInit_ErrorDirectoryNotEmpty(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Create directory with content
	wmemDir := filepath.Join(h.TempDir(), "my-wmem1")
	h.MkdirAll(wmemDir)
	h.WriteFile(filepath.Join(wmemDir, "existing-file.txt"), "content")

	// Try to initialize wmem repo in non-empty directory
	h.SetWorkDir(h.TempDir())
	output, err := h.RunGitWmem("init", "my-wmem1")
	h.AssertCommandError(output, err, "Directory is not empty", "git-wmem-init on non-empty directory")
}

// TestGitWmemInit_ErrorDirectoryExists tests error when directory already exists and is not empty
// Reference: docs/use-cases/git-wmem-init/basic.md#alternatives 2b
func TestGitWmemInit_ErrorDirectoryExists(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Create directory with content
	wmemDir := filepath.Join(h.TempDir(), "my-wmem1")
	h.MkdirAll(wmemDir)
	h.WriteFile(filepath.Join(wmemDir, "existing-file.txt"), "content")

	// Try to initialize wmem repo in existing non-empty directory
	h.SetWorkDir(h.TempDir())
	output, err := h.RunGitWmem("init", "my-wmem1")
	h.AssertCommandError(output, err, "Please specify an empty directory", "git-wmem-init on existing non-empty directory")
}
