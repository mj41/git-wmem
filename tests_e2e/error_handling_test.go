package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestMissingTreeObject_UncommittedChanges tests the specific issue where
// tree objects are missing from bare repository when committing uncommitted changes
// Reference: https://github.com/mj41/git-wmem/issues/tree-object-bug
func TestMissingTreeObject_UncommittedChanges(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)

	// Create a test project with initial commit
	testProjectDir := filepath.Join(h.TempDir(), "test-project")
	h.MkdirAll(testProjectDir)
	h.SetWorkDir(testProjectDir)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init")

	h.WriteFile("file1.txt", "Initial content")
	_, err = h.RunGit("add", "file1.txt")
	h.AssertCommandSuccess("", err, "git add")

	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit")

	// Configure wmem to track this project
	h.SetWorkDir(wmemDir)
	h.WriteFile("md/commit-workdir-paths", "../test-project")

	// First wmem commit (this should work fine)
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Make UNCOMMITTED changes in the test project
	h.SetWorkDir(testProjectDir)
	h.WriteFile("file1.txt", "Modified content")
	h.WriteFile("file2.txt", "New uncommitted file")
	// NOTE: These changes are NOT committed to git

	// Go back to wmem repo and make second commit
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second git-wmem-commit with uncommitted changes")

	// Verify that the bare repository is not corrupted
	bareRepoDir := filepath.Join(wmemDir, "repos", "test-project.git")
	h.SetWorkDir(bareRepoDir)

	// This should work without "unable to read tree" error
	output, err = h.RunGit("log", "--stat", "wmem-br/head")
	h.AssertCommandSuccess(output, err, "git log --stat on wmem-br/head")

	// Verify repository integrity
	output, err = h.RunGit("fsck")
	h.AssertCommandSuccess(output, err, "git fsck")

	// The output should not contain "missing tree" or similar errors
	if strings.Contains(output, "missing tree") {
		t.Errorf("Repository integrity check failed - missing tree objects detected")
	}
	if strings.Contains(output, "broken link") {
		t.Errorf("Repository integrity check failed - broken links detected")
	}
}

// TestBareRepoIntegrity_AfterCommit tests that commits don't leave repos corrupted
func TestBareRepoIntegrity_AfterCommit(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and multiple test projects
	wmemDir := setupBasicWmemRepo(h)
	projectA, projectB := setupTestProjects(h)

	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	// Make multiple commits with various changes
	scenarios := []struct {
		name        string
		setupFunc   func()
		description string
	}{
		{
			name: "committed_changes",
			setupFunc: func() {
				// Make committed changes
				h.SetWorkDir(projectA)
				h.WriteFile("committed.txt", "committed change")
				h.RunGit("add", "committed.txt")
				h.RunGit("commit", "-m", "Committed change")
			},
			description: "Committed changes scenario",
		},
		{
			name: "uncommitted_changes",
			setupFunc: func() {
				// Make uncommitted changes
				h.SetWorkDir(projectB)
				h.WriteFile("uncommitted.txt", "uncommitted change")
				// Don't commit these changes
			},
			description: "Uncommitted changes scenario",
		},
		{
			name: "branch_changes",
			setupFunc: func() {
				// Create new branch with changes
				h.SetWorkDir(projectA)
				h.RunGit("checkout", "-b", "feature/test")
				h.WriteFile("feature.txt", "feature content")
				h.RunGit("add", "feature.txt")
				h.RunGit("commit", "-m", "Feature commit")
			},
			description: "Branch changes scenario",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Setup the scenario
			scenario.setupFunc()

			// Make wmem commit
			h.SetWorkDir(wmemDir)
			output, err := h.RunGitWmem("commit")
			h.AssertCommandSuccess(output, err, "git-wmem-commit for "+scenario.description)

			// Verify integrity of both bare repositories
			for _, projectName := range []string{"my-projectA", "my-projectB"} {
				bareRepoDir := filepath.Join(wmemDir, "repos", projectName+".git")
				h.SetWorkDir(bareRepoDir)

				// Check git log works
				output, err := h.RunGit("log", "--oneline", "wmem-br/head")
				h.AssertCommandSuccess(output, err, "git log for "+projectName)

				// Check repository integrity
				output, err = h.RunGit("fsck")
				h.AssertCommandSuccess(output, err, "git fsck for "+projectName)

				// Verify no corruption
				if strings.Contains(output, "missing") || strings.Contains(output, "broken") {
					t.Errorf("Repository %s has integrity issues after %s: %s", projectName, scenario.description, output)
				}
			}
		})
	}
}
