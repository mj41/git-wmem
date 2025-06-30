package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDataStructures_WmemUid tests wmem-uid format and uniqueness
// Reference: docs/data-structures.md#wmem-uid
func TestDataStructures_WmemUid(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test projects
	wmemDir := setupBasicWmemRepo(h)
	_, _ = setupTestProjects(h)

	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	// Make multiple commits and collect wmem-uids
	var wmemUids []string

	for i := 0; i < 3; i++ {
		output, err := h.RunGitWmem("commit")
		h.AssertCommandSuccess(output, err, "git-wmem-commit")

		// Get the commit message which should contain wmem-uid
		logOutput, err := h.RunGit("log", "--oneline", "-1")
		h.AssertCommandSuccess(logOutput, err, "git log")

		// Extract wmem-uid from commit message
		lines := strings.Split(strings.TrimSpace(logOutput), "\n")
		commitMsg := lines[0]

		// Look for wmem-uid pattern in commit message
		// Reference: docs/data-structures.md#wmem-uid format
		if strings.Contains(commitMsg, "wmem-") {
			// Extract the wmem-uid (should be in format wmem-YYMMDD-HHMMSS-abXY1234)
			parts := strings.Fields(commitMsg)
			for _, part := range parts {
				if strings.HasPrefix(part, "wmem-") && len(part) == 27 { // wmem- + 6 + 1 + 6 + 1 + 8 = 27
					wmemUids = append(wmemUids, part)
					break
				}
			}
		}
	}

	// Verify we collected wmem-uids
	if len(wmemUids) == 0 {
		t.Fatal("No wmem-uids found in commit messages")
	}

	// Verify wmem-uid format
	// Reference: docs/data-structures.md#wmem-uid format
	for i, uid := range wmemUids {
		// Should match pattern: wmem-YYMMDD-HHMMSS-abXY1234
		if !strings.HasPrefix(uid, "wmem-") {
			t.Errorf("wmem-uid %d should start with 'wmem-': %s", i, uid)
		}

		if len(uid) != 27 {
			t.Errorf("wmem-uid %d should be 27 characters long: %s (len=%d)", i, uid, len(uid))
		}

		parts := strings.Split(uid, "-")
		if len(parts) != 4 {
			t.Errorf("wmem-uid %d should have 4 parts separated by '-': %s", i, uid)
			continue
		}

		// Verify date part (YYMMDD)
		if len(parts[1]) != 6 {
			t.Errorf("wmem-uid %d date part should be 6 digits: %s", i, parts[1])
		}

		// Verify time part (HHMMSS)
		if len(parts[2]) != 6 {
			t.Errorf("wmem-uid %d time part should be 6 digits: %s", i, parts[2])
		}

		// Verify random part (8 characters [a-zA-Z0-9])
		if len(parts[3]) != 8 {
			t.Errorf("wmem-uid %d random part should be 8 characters: %s", i, parts[3])
		}
	}

	// Verify uniqueness (with multiple rapid commits, they should still be unique)
	uidSet := make(map[string]bool)
	for _, uid := range wmemUids {
		if uidSet[uid] {
			t.Errorf("Duplicate wmem-uid found: %s", uid)
		}
		uidSet[uid] = true
	}
}

// TestDataStructures_CommitInfo tests commit-info structure
// Reference: docs/data-structures.md#commit-info
func TestDataStructures_CommitInfo(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test projects
	wmemDir := setupBasicWmemRepo(h)
	_, _ = setupTestProjects(h)

	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	// Set commit message prefix
	// Reference: docs/data-structures.md#commit-msg-example
	h.WriteFile("md/commit/msg-prefix", "My git-wmem commit prefix")

	// Make commit
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "git-wmem-commit")

	// Get full commit message
	fullMsg, err := h.RunGit("log", "-1", "--pretty=format:%B")
	h.AssertCommandSuccess(fullMsg, err, "git log full message")

	// Verify commit message structure
	// Reference: docs/data-structures.md#commit-msg-example
	if !strings.Contains(fullMsg, "My git-wmem commit prefix") {
		t.Errorf("Expected msg-prefix in commit message: %s", fullMsg)
	}

	if !strings.Contains(fullMsg, "wmem-uid:") {
		t.Errorf("Expected wmem-uid field in commit message: %s", fullMsg)
	}

	// Verify author and committer information
	// Reference: docs/data-structures.md#commit-info
	authorOutput, err := h.RunGit("log", "-1", "--pretty=format:%an <%ae>")
	h.AssertCommandSuccess(authorOutput, err, "git log author")

	if !strings.Contains(authorOutput, "WMem Git <git-wmem@mj41.cz>") {
		t.Errorf("Expected default author in commit: %s", authorOutput)
	}

	committerOutput, err := h.RunGit("log", "-1", "--pretty=format:%cn <%ce>")
	h.AssertCommandSuccess(committerOutput, err, "git log committer")

	if !strings.Contains(committerOutput, "WMem Git <git-wmem@mj41.cz>") {
		t.Errorf("Expected default committer in commit: %s", committerOutput)
	}
}

// TestDataStructures_WorkdirMap tests workdir-map.json structure
// Reference: docs/data-structures.md#workdir-map
func TestDataStructures_WorkdirMap(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test projects
	wmemDir := setupBasicWmemRepo(h)
	_, _ = setupTestProjects(h)

	h.SetWorkDir(wmemDir)

	// Verify initial empty workdir-map
	h.AssertFileEquals("md-internal/workdir-map.json", "{}")

	// Add first set of workdirs
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	// Make commit
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Verify workdir-map.json was updated
	content, err := os.ReadFile(filepath.Join(wmemDir, "md-internal/workdir-map.json"))
	if err != nil {
		t.Fatalf("Failed to read workdir-map.json: %v", err)
	}

	var workdirMap map[string]string
	if err := json.Unmarshal(content, &workdirMap); err != nil {
		t.Fatalf("Failed to parse workdir-map.json: %v", err)
	}

	// Verify mappings
	// Reference: docs/data-structures.md#workdir-map example
	expectedMappings := map[string]string{
		"my-projectA": "../my-projectA",
		"my-projectB": "../my-projectB",
	}

	for name, path := range expectedMappings {
		if workdirMap[name] != path {
			t.Errorf("Expected mapping %s->%s, got %s->%s", name, path, name, workdirMap[name])
		}
	}

	// Test append-only nature by adding a duplicate path with different name
	// Reference: docs/data-structures.md#workdir-map append-only

	// Create another project with similar name to test collision handling
	projectA2 := filepath.Join(h.TempDir(), "my-projectA-2")
	h.MkdirAll(projectA2)
	h.SetWorkDir(projectA2)

	_, err = h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init projectA-2")

	h.WriteFile("file.txt", "content")
	_, err = h.RunGit("add", "file.txt")
	h.AssertCommandSuccess("", err, "git add")

	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit")

	// Add new workdir
	h.SetWorkDir(wmemDir)
	h.WriteFile("md/commit-workdir-paths", "../my-projectA-2")

	// Make another commit
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second git-wmem-commit")

	// Verify workdir-map.json preserved old entries and added new one
	content, err = os.ReadFile(filepath.Join(wmemDir, "md-internal/workdir-map.json"))
	if err != nil {
		t.Fatalf("Failed to read updated workdir-map.json: %v", err)
	}

	if err := json.Unmarshal(content, &workdirMap); err != nil {
		t.Fatalf("Failed to parse updated workdir-map.json: %v", err)
	}

	// Should still have old mappings
	if workdirMap["my-projectA"] != "../my-projectA" {
		t.Errorf("Old mapping should be preserved: my-projectA")
	}
	if workdirMap["my-projectB"] != "../my-projectB" {
		t.Errorf("Old mapping should be preserved: my-projectB")
	}

	// Should have new mapping
	if workdirMap["my-projectA-2"] != "../my-projectA-2" {
		t.Errorf("New mapping should be added: my-projectA-2")
	}

	// Verify total number of mappings
	if len(workdirMap) != 3 {
		t.Errorf("Expected 3 mappings, got %d: %v", len(workdirMap), workdirMap)
	}
}

// TestBareRepoIntegrity_UncommittedChanges tests the missing tree object bug
// This reproduces the exact issue: "fatal: unable to read tree"
func TestBareRepoIntegrity_UncommittedChanges(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo
	wmemDir := setupBasicWmemRepo(h)

	// Create test project
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

	// Setup wmem to track this project
	h.SetWorkDir(wmemDir)
	h.WriteFile("md/commit-workdir-paths", "../test-project")

	// First wmem commit (should work)
	output, err := h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Make UNCOMMITTED changes in test project (this triggers the bug)
	h.SetWorkDir(testProjectDir)
	h.WriteFile("file1.txt", "Modified content")
	h.WriteFile("file2.txt", "New uncommitted file")
	// NOTE: Not committing these changes - they remain uncommitted

	// Second wmem commit with uncommitted changes (this creates the missing tree object bug)
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "git-wmem-commit with uncommitted changes")

	// Now verify bare repo integrity - this should NOT fail
	bareRepoDir := filepath.Join(wmemDir, "repos", "test-project.git")
	h.SetWorkDir(bareRepoDir)

	// Test 1: git log should work without tree errors
	output, err = h.RunGit("log", "--oneline", "wmem-br/head")
	h.AssertCommandSuccess(output, err, "git log --oneline")

	// Test 2: git log --stat should work (this was failing before)
	output, err = h.RunGit("log", "--stat", "wmem-br/head")
	h.AssertCommandSuccess(output, err, "git log --stat")

	// Test 3: git fsck should pass
	output, err = h.RunGit("fsck")
	h.AssertCommandSuccess(output, err, "git fsck")

	// Test 4: Verify all tree objects exist
	output, err = h.RunGit("log", "--format=%H %T", "-3", "wmem-br/head")
	h.AssertCommandSuccess(output, err, "git log format")

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			treeHash := parts[1]
			output, err := h.RunGit("cat-file", "-t", treeHash)
			h.AssertCommandSuccess(output, err, "git cat-file -t "+treeHash)
			if !strings.Contains(output, "tree") {
				t.Errorf("Object %s should be a tree, got: %s", treeHash, output)
			}
		}
	}
}
