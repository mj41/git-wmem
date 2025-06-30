package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestBasicDevelopmentWorkflow tests the complete basic development workflow
// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow
func TestBasicDevelopmentWorkflow(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	workDir := h.TempDir()

	// Step 1: User starts git-wmem-init basic
	// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow step 1
	// Reference: docs/use-cases/git-wmem-init/basic.md#main-scenario
	h.SetWorkDir(workDir)
	output, err := h.RunGitWmem("init", "my-wmem1")
	h.AssertCommandSuccess(output, err, "git-wmem-init my-wmem1")

	wmemDir := filepath.Join(workDir, "my-wmem1")
	h.SetWorkDir(wmemDir)

	// Verify initial commit was created
	output, err = h.RunGit("log", "--oneline")
	h.AssertCommandSuccess(output, err, "git log after init")
	if !strings.Contains(output, "Initialize git-wmem repository `my-wmem1`") {
		t.Errorf("Expected initial commit message, got: %s", output)
	}

	// Step 2: User runs wds-setup-basic
	// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow step 2
	// Reference: docs/use-cases/user-sh-cmds/wds-setup-basic.md#main-scenario

	// Create project A
	projectA := filepath.Join(workDir, "my-projectA")
	h.MkdirAll(projectA)
	h.SetWorkDir(projectA)

	_, err = h.RunGit("init")
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

	// Add workdirs to wmem
	h.SetWorkDir(wmemDir)
	h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
	h.AppendToFile("md/commit-workdir-paths", "../my-projectB")

	// Step 3: User runs git-wmem-commit basic (second commit)
	// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow step 3
	// Reference: docs/use-cases/git-wmem-commit/basic.md#main-scenario
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "first git-wmem-commit")

	// Verify bare repositories were created
	h.AssertDirExists("repos/my-projectA.git")
	h.AssertDirExists("repos/my-projectB.git")

	// Verify wmem commit was created
	output, err = h.RunGit("log", "--oneline")
	h.AssertCommandSuccess(output, err, "git log after first wmem commit")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 commits after first wmem-commit, got %d: %s", len(lines), output)
	}

	// Step 4: User runs wds-file-changes
	// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow step 4
	// Reference: docs/use-cases/user-sh-cmds/wds-file-changes.md
	h.SetWorkDir(projectA)
	h.WriteFile("file-featX1.txt", "file file-featX1.txt: content A-X-pre-a, line 1")

	h.SetWorkDir(projectB)
	_, err = h.RunGit("checkout", "-b", "workH")
	h.AssertCommandSuccess("", err, "git checkout -b workH")

	h.MkdirAll("workH-dir")
	h.WriteFile("workH-dir/file-workH1.txt", "file workH-dir/file-workH1.txt: content B-W-pre-a, line 1")

	// Step 5: User runs git-wmem-commit basic (third commit)
	// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow step 5
	h.SetWorkDir(wmemDir)
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "second git-wmem-commit")

	// Verify another wmem commit was created
	output, err = h.RunGit("log", "--oneline")
	h.AssertCommandSuccess(output, err, "git log after second wmem commit")
	lines = strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 commits after second wmem-commit, got %d: %s", len(lines), output)
	}

	// Step 6: User runs wds-git-cmds
	// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow step 6
	// Reference: docs/use-cases/user-sh-cmds/wds-git-cmds.md
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
	// workH branch should already exist from step 4
	_, err = h.RunGit("checkout", "workH")
	h.AssertCommandSuccess("", err, "git checkout workH")

	h.WriteFile("workH-dir/file-workH1.txt", "file workH-dir/file-workH1.txt: content B-W-a, line 1")
	_, err = h.RunGit("add", "workH-dir/file-workH1.txt")
	h.AssertCommandSuccess("", err, "git add workH-dir/file-workH1.txt")

	_, err = h.RunGit("commit", "-m", "Project my-projectB, feature W, commit B-W-a")
	h.AssertCommandSuccess("", err, "git commit B-W-a")

	// Step 7: User runs commit-msg-prefix-file
	// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow step 7
	// Reference: docs/use-cases/user-sh-cmds/commit-msg-prefix-file.md
	h.SetWorkDir(wmemDir)
	h.WriteFile("md/commit/msg-prefix", "projA and projB features")

	// Step 8: User runs git-wmem-commit basic (fourth commit)
	// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow step 8
	output, err = h.RunGitWmem("commit")
	h.AssertCommandSuccess(output, err, "third git-wmem-commit")

	// Verify commit message contains prefix
	output, err = h.RunGit("log", "--oneline", "-1")
	h.AssertCommandSuccess(output, err, "git log latest commit")
	if !strings.Contains(output, "projA and projB features") {
		t.Errorf("Expected commit message prefix in latest commit, got: %s", output)
	}

	// Step 9: User runs git-wmem-log basic
	// Reference: docs/use-cases/use-cases.md#uc-basic-development-workflow step 9
	// Reference: docs/use-cases/git-wmem-log/basic.md#main-scenario
	output, err = h.RunGitWmem("log")
	h.AssertCommandSuccess(output, err, "git-wmem-log")

	// Verify log shows wmem-uid and workdir information
	if !strings.Contains(output, "wmem-") {
		t.Errorf("Expected wmem-uid in log output: %s", output)
	}
	if !strings.Contains(output, "my-projectA:") {
		t.Errorf("Expected my-projectA in log output: %s", output)
	}
	if !strings.Contains(output, "my-projectB:") {
		t.Errorf("Expected my-projectB in log output: %s", output)
	}
	if !strings.Contains(output, "projA and projB features") {
		t.Errorf("Expected commit message prefix in log output: %s", output)
	}

	// Verify final state: should have 4 total commits
	output, err = h.RunGit("log", "--oneline")
	h.AssertCommandSuccess(output, err, "git log final state")
	lines = strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 total commits in final state, got %d: %s", len(lines), output)
	}
}
