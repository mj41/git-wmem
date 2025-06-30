package e2e

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestWmemUidConsistency_BetweenRepos tests that the same wmem-uid is used
// for both wmem-repo and wmem-wd-repo commits that belong together
// Reference: docs/data-structures.md#wmem-uid
func TestWmemUidConsistency_BetweenRepos(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test projects
	wmemDir := setupBasicWmemRepo(h)
	projectADir, projectBDir := setupTestProjects(h)

	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	// Set a specific commit message prefix for easy identification
	h.WriteFile("md/commit/msg-prefix", "Test wmem-uid consistency")

	// Make some changes in both projects to ensure commits will be created
	h.SetWorkDir(projectADir)
	h.WriteFile("test-file-a.txt", "content for project A")
	_, err := h.RunGit("add", "test-file-a.txt")
	h.AssertCommandSuccess("", err, "git add projectA")
	_, err = h.RunGit("commit", "-m", "Add test file A")
	h.AssertCommandSuccess("", err, "git commit projectA")

	h.SetWorkDir(projectBDir)
	h.WriteFile("test-file-b.txt", "content for project B")
	_, err = h.RunGit("add", "test-file-b.txt")
	h.AssertCommandSuccess("", err, "git add projectB")
	_, err = h.RunGit("commit", "-m", "Add test file B")
	h.AssertCommandSuccess("", err, "git commit projectB")

	// Make wmem commit
	h.SetWorkDir(wmemDir)
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "git-wmem-commit")

	// Extract wmem-uid from wmem-repo commit
	wmemRepoCommitMsg, err := h.RunGit("log", "-1", "--pretty=format:%B")
	h.AssertCommandSuccess(wmemRepoCommitMsg, err, "git log wmem-repo")

	wmemRepoUID := extractWmemUID(t, wmemRepoCommitMsg, "wmem-repo")

	// Check wmem-wd-repo commits for projectA
	bareRepoADir := filepath.Join(wmemDir, "repos", "my-projectA.git")
	h.SetWorkDir(bareRepoADir)

	projectACommitMsg, err := h.RunGit("log", "-1", "--pretty=format:%B", "wmem-br/main")
	h.AssertCommandSuccess(projectACommitMsg, err, "git log projectA wmem-br/main")

	projectAUID := extractWmemUID(t, projectACommitMsg, "wmem-wd-repo projectA")

	// Check wmem-wd-repo commits for projectB
	bareRepoBDir := filepath.Join(wmemDir, "repos", "my-projectB.git")
	h.SetWorkDir(bareRepoBDir)

	projectBCommitMsg, err := h.RunGit("log", "-1", "--pretty=format:%B", "wmem-br/main")
	h.AssertCommandSuccess(projectBCommitMsg, err, "git log projectB wmem-br/main")

	projectBUID := extractWmemUID(t, projectBCommitMsg, "wmem-wd-repo projectB")

	// Verify all wmem-uids are the same
	if wmemRepoUID != projectAUID {
		t.Errorf("wmem-uid mismatch between wmem-repo (%s) and projectA wmem-wd-repo (%s)",
			wmemRepoUID, projectAUID)
	}

	if wmemRepoUID != projectBUID {
		t.Errorf("wmem-uid mismatch between wmem-repo (%s) and projectB wmem-wd-repo (%s)",
			wmemRepoUID, projectBUID)
	}

	if projectAUID != projectBUID {
		t.Errorf("wmem-uid mismatch between projectA (%s) and projectB (%s) wmem-wd-repos",
			projectAUID, projectBUID)
	}

	t.Logf("✅ All commits use the same wmem-uid: %s", wmemRepoUID)
}

// TestWmemUidConsistency_MergeCommits tests wmem-uid consistency when merge commits are created
// This covers the ALG: wmem merge scenario (Alternative 5b)
func TestWmemUidConsistency_MergeCommits(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)
	projectDir := filepath.Join(h.TempDir(), "test-project")
	h.MkdirAll(projectDir)
	h.SetWorkDir(projectDir)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init")

	h.WriteFile("initial.txt", "initial content")
	_, err = h.RunGit("add", "initial.txt")
	h.AssertCommandSuccess("", err, "git add")

	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit")

	// Setup wmem to track this project
	h.SetWorkDir(wmemDir)
	h.WriteFile("md/commit-workdir-paths", "../test-project")
	h.WriteFile("md/commit/msg-prefix", "Test merge consistency")

	// First wmem commit (establishes wmem-br/main)
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Create new commits in workdir that will require merging
	h.SetWorkDir(projectDir)
	h.WriteFile("new-file.txt", "new content")
	_, err = h.RunGit("add", "new-file.txt")
	h.AssertCommandSuccess("", err, "git add new file")

	_, err = h.RunGit("commit", "-m", "Add new file")
	h.AssertCommandSuccess("", err, "git commit new file")

	h.WriteFile("another-file.txt", "another content")
	_, err = h.RunGit("add", "another-file.txt")
	h.AssertCommandSuccess("", err, "git add another file")

	_, err = h.RunGit("commit", "-m", "Add another file")
	h.AssertCommandSuccess("", err, "git commit another file")

	// Second wmem commit (should create merge commits)
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second git-wmem-commit with merges")

	// Extract wmem-uid from wmem-repo commit
	wmemRepoCommitMsg, err := h.RunGit("log", "-1", "--pretty=format:%B")
	h.AssertCommandSuccess(wmemRepoCommitMsg, err, "git log wmem-repo")

	wmemRepoUID := extractWmemUID(t, wmemRepoCommitMsg, "wmem-repo")

	// Check wmem-wd-repo commit (should be a merge commit)
	bareRepoDir := filepath.Join(wmemDir, "repos", "test-project.git")
	h.SetWorkDir(bareRepoDir)

	// Get the latest commit on wmem-br/main (should be merge commit)
	mergeCommitMsg, err := h.RunGit("log", "-1", "--pretty=format:%B", "wmem-br/main")
	h.AssertCommandSuccess(mergeCommitMsg, err, "git log merge commit")

	mergeUID := extractWmemUID(t, mergeCommitMsg, "wmem-wd-repo merge commit")

	// Verify merge commit contains the same wmem-uid
	if wmemRepoUID != mergeUID {
		t.Errorf("wmem-uid mismatch between wmem-repo (%s) and merge commit (%s)",
			wmemRepoUID, mergeUID)
	}

	// Verify this is indeed a merge commit (has 2 parents)
	parentCount, err := h.RunGit("rev-list", "--count", "--parents", "-1", "wmem-br/main")
	h.AssertCommandSuccess(parentCount, err, "git rev-list parents")

	// The output should contain the commit hash + 2 parent hashes (3 total hashes)
	hashes := strings.Fields(strings.TrimSpace(parentCount))
	if len(hashes) < 3 {
		t.Errorf("Expected merge commit to have 2 parents, got %d hashes: %v", len(hashes)-1, hashes)
	}

	// Verify merge commit message mentions the merge strategy
	if !strings.Contains(mergeCommitMsg, "Merge workdir") {
		t.Errorf("Expected merge commit message to mention merge strategy, got: %s", mergeCommitMsg)
	}

	if !strings.Contains(mergeCommitMsg, "accepting workdir's branch tree hash") {
		t.Errorf("Expected merge commit message to mention ALG: wmem merge strategy, got: %s", mergeCommitMsg)
	}

	t.Logf("✅ Merge commit uses the same wmem-uid: %s", wmemRepoUID)
}

// TestWmemUidConsistency_UncommittedChanges tests wmem-uid consistency when committing uncommitted changes
// This covers the regular commit scenario (not merge)
func TestWmemUidConsistency_UncommittedChanges(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)
	projectDir := filepath.Join(h.TempDir(), "test-project")
	h.MkdirAll(projectDir)
	h.SetWorkDir(projectDir)

	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init")

	h.WriteFile("initial.txt", "initial content")
	_, err = h.RunGit("add", "initial.txt")
	h.AssertCommandSuccess("", err, "git add")

	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit")

	// Setup wmem to track this project
	h.SetWorkDir(wmemDir)
	h.WriteFile("md/commit-workdir-paths", "../test-project")
	h.WriteFile("md/commit/msg-prefix", "Test uncommitted consistency")

	// First wmem commit
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Make UNCOMMITTED changes in workdir
	h.SetWorkDir(projectDir)
	h.WriteFile("initial.txt", "modified content")
	h.WriteFile("uncommitted.txt", "this file is not committed")
	// NOTE: NOT committing these changes

	// Second wmem commit (should create regular commit from uncommitted changes)
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "git-wmem-commit with uncommitted changes")

	// Extract wmem-uid from wmem-repo commit
	wmemRepoCommitMsg, err := h.RunGit("log", "-1", "--pretty=format:%B")
	h.AssertCommandSuccess(wmemRepoCommitMsg, err, "git log wmem-repo")

	wmemRepoUID := extractWmemUID(t, wmemRepoCommitMsg, "wmem-repo")

	// Check wmem-wd-repo commit (should be regular commit, not merge)
	bareRepoDir := filepath.Join(wmemDir, "repos", "test-project.git")
	h.SetWorkDir(bareRepoDir)

	regularCommitMsg, err := h.RunGit("log", "-1", "--pretty=format:%B", "wmem-br/main")
	h.AssertCommandSuccess(regularCommitMsg, err, "git log regular commit")

	regularUID := extractWmemUID(t, regularCommitMsg, "wmem-wd-repo regular commit")

	// Verify regular commit contains the same wmem-uid
	if wmemRepoUID != regularUID {
		t.Errorf("wmem-uid mismatch between wmem-repo (%s) and regular commit (%s)",
			wmemRepoUID, regularUID)
	}

	// Verify this is NOT a merge commit (has 1 parent)
	parentCount, err := h.RunGit("rev-list", "--count", "--parents", "-1", "wmem-br/main")
	h.AssertCommandSuccess(parentCount, err, "git rev-list parents")

	// The output should contain the commit hash + 1 parent hash (2 total hashes)
	hashes := strings.Fields(strings.TrimSpace(parentCount))
	if len(hashes) != 2 {
		t.Errorf("Expected regular commit to have 1 parent, got %d hashes: %v", len(hashes)-1, hashes)
	}

	// Verify regular commit message does NOT mention merge
	if strings.Contains(regularCommitMsg, "Merge workdir") {
		t.Errorf("Expected regular commit, but got merge commit message: %s", regularCommitMsg)
	}

	t.Logf("✅ Regular commit uses the same wmem-uid: %s", wmemRepoUID)
}

// extractWmemUID extracts wmem-uid from a commit message
func extractWmemUID(t *testing.T, commitMsg, context string) string {
	// Look for wmem-uid: wmem-YYMMDD-HHMMSS-abXY1234 pattern
	re := regexp.MustCompile(`wmem-uid:\s*(wmem-\d{6}-\d{6}-[a-zA-Z0-9]{8})`)
	matches := re.FindStringSubmatch(commitMsg)
	if len(matches) < 2 {
		t.Fatalf("No wmem-uid found in %s commit message: %s", context, commitMsg)
	}
	return matches[1]
}
