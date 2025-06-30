package e2e

import (
	"strings"
	"testing"
)

// TestGitWmemLog_Basic tests basic git-wmem-log functionality
// Reference: file:///home/mj/work-stai/git-wmem/docs/use-cases/git-wmem-log/basic.md#main-scenario
// See also: docs/use-cases/git-wmem-log/basic.md
func TestGitWmemLog_Basic(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test projects
	wmemDir := setupBasicWmemRepo(h)
	_, _ = setupTestProjects(h)

	// Setup workdir paths and make initial commit
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	// Make first wmem commit
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Set commit message prefix for next commit
	// Reference: docs/use-cases/user-sh-cmds/commit-msg-prefix-file.md
	h.WriteFile("md/commit/msg-prefix", "projA and projB features")

	// Make second wmem commit
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second git-wmem-commit")

	// Test git-wmem-log
	// Reference: docs/use-cases/git-wmem-log/basic.md step 1-2
	output, err = h.RunGitWmem("log")
	h.AssertCommandSuccess(output, err, "git-wmem-log")

	// Verify output format
	// Reference: docs/use-cases/git-wmem-log/basic.md#example-output-format
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should show commits in reverse chronological order
	var wmemCommits []string
	var workdirLines []string

	for _, line := range lines {
		if line == "" {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(line), "wmem-") {
			wmemCommits = append(wmemCommits, strings.TrimSpace(line))
		} else if strings.HasPrefix(line, "  ") {
			workdirLines = append(workdirLines, strings.TrimSpace(line))
		}
	}

	// Should have at least 2 wmem commits (initial + one with prefix)
	if len(wmemCommits) < 2 {
		t.Errorf("Expected at least 2 wmem commits, got %d: %v", len(wmemCommits), wmemCommits)
	}

	// Latest commit should contain the prefix
	if !strings.Contains(wmemCommits[0], "projA and projB features") {
		t.Errorf("Expected latest commit to contain prefix, got: %s", wmemCommits[0])
	}

	// Should show workdir information
	// Reference: docs/use-cases/git-wmem-log/basic.md step 2
	foundProjectA := false
	foundProjectB := false

	for _, line := range workdirLines {
		if strings.Contains(line, "my-projectA:") {
			foundProjectA = true
		}
		if strings.Contains(line, "my-projectB:") {
			foundProjectB = true
		}
	}

	if !foundProjectA {
		t.Errorf("Expected to find my-projectA in workdir lines: %v", workdirLines)
	}
	if !foundProjectB {
		t.Errorf("Expected to find my-projectB in workdir lines: %v", workdirLines)
	}
}

// TestGitWmemLog_WithMultipleCommits tests log with multiple commits
// Reference: docs/use-cases/git-wmem-log/basic.md#example-output-format
func TestGitWmemLog_WithMultipleCommits(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test projects
	wmemDir := setupBasicWmemRepo(h)
	projectA, projectB := setupTestProjects(h)

	// Setup workdir paths
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	// First wmem commit
	h.WriteFile("md/commit/msg-prefix", "initial setup")
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first wmem commit")

	// Make changes in projects
	h.SetWorkDir(projectA)
	h.WriteFile("newfileA.txt", "new content A")
	_, err = h.RunGit("add", "newfileA.txt")
	h.AssertCommandSuccess("", err, "git add newfileA.txt")
	_, err = h.RunGit("commit", "-m", "Add newfileA")
	h.AssertCommandSuccess("", err, "git commit newfileA")

	h.SetWorkDir(projectB)
	h.WriteFile("newfileB.txt", "new content B")
	_, err = h.RunGit("add", "newfileB.txt")
	h.AssertCommandSuccess("", err, "git add newfileB.txt")
	_, err = h.RunGit("commit", "-m", "Add newfileB")
	h.AssertCommandSuccess("", err, "git commit newfileB")

	// Second wmem commit
	h.SetWorkDir(wmemDir)
	h.WriteFile("md/commit/msg-prefix", "added new files")
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second wmem commit")

	// Test log output
	output, err = h.RunGitWmem("log")
	h.AssertCommandSuccess(output, err, "git-wmem-log with multiple commits")

	// Verify both commit messages appear
	if !strings.Contains(output, "added new files") {
		t.Errorf("Expected 'added new files' in log output: %s", output)
	}
	if !strings.Contains(output, "initial setup") {
		t.Errorf("Expected 'initial setup' in log output: %s", output)
	}

	// Verify wmem-uid format in output
	// Reference: docs/data-structures.md#wmem-uid
	if !strings.Contains(output, "wmem-") {
		t.Errorf("Expected wmem-uid format in log output: %s", output)
	}
}

// TestGitWmemLog_ErrorNotInWmemRepo tests error when not in wmem repo
// Reference: docs/use-cases/git-wmem-log/basic.md preconditions
func TestGitWmemLog_ErrorNotInWmemRepo(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Try to run log outside wmem repo
	h.SetWorkDir(h.TempDir())
	output, err := h.RunGitWmem("log")
	h.AssertCommandError(output, err, ".git-wmem", "git-wmem-log outside wmem repo")
}
