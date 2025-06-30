package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitCompatibility tests that repositories created by git-wmem are fully compatible with native Git
func TestGitCompatibility(t *testing.T) {
	// Get the current working directory to find the binaries
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Go up one directory to the project root to find binaries
	projectRoot := filepath.Dir(cwd)
	gitWmemInit := filepath.Join(projectRoot, "bin", "git-wmem-init")
	gitWmemCommit := filepath.Join(projectRoot, "bin", "git-wmem-commit")

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "git-wmem-compatibility-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test setup: Create a git-wmem repository and a source project
	wmemDir := filepath.Join(tempDir, "my-wmem")
	projectDir := filepath.Join(tempDir, "my-project")

	// Step 1: Create source git repository
	t.Logf("Creating source git repository at %s", projectDir)
	createTestGitRepo(t, projectDir)

	// Step 2: Initialize git-wmem repository
	t.Logf("Initializing git-wmem repository at %s", wmemDir)
	runCommand(t, tempDir, gitWmemInit, "my-wmem")

	// Step 3: Configure git-wmem to track the project
	workdirPathsFile := filepath.Join(wmemDir, "md", "commit-workdir-paths")
	relProjectPath, err := filepath.Rel(wmemDir, projectDir)
	if err != nil {
		t.Fatalf("Failed to get relative path: %v", err)
	}
	err = ioutil.WriteFile(workdirPathsFile, []byte(relProjectPath), 0644)
	if err != nil {
		t.Fatalf("Failed to write workdir paths: %v", err)
	}

	// Step 4: Configure commit info
	setupCommitInfo(t, wmemDir)

	// Step 5: Create git-wmem commit
	t.Logf("Creating git-wmem commit")
	runCommand(t, wmemDir, gitWmemCommit)

	// Step 6: Verify Git binary compatibility
	t.Logf("Verifying Git binary compatibility")
	verifyGitCompatibility(t, wmemDir, "my-project")
}

// createTestGitRepo creates a test git repository with some files
func createTestGitRepo(t *testing.T, projectDir string) {
	err := os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Initialize git repository
	runCommand(t, projectDir, "git", "init")
	runCommand(t, projectDir, "git", "config", "user.name", "Test User")
	runCommand(t, projectDir, "git", "config", "user.email", "test@example.com")

	// Create test files with various names to test sorting
	testFiles := map[string]string{
		"README.md":     "# Test Project\n",
		"LICENSE":       "MIT License\n",
		"Makefile":      "all:\n\techo hello\n",
		"src/main.go":   "package main\n\nfunc main() {}\n",
		"docs/api.md":   "# API Documentation\n",
		"a-file.txt":    "Content A\n",
		"a/subfile.txt": "Sub content\n",
		"b.txt":         "Content B\n",
		"b/other.txt":   "Other content\n",
	}

	for filename, content := range testFiles {
		fullPath := filepath.Join(projectDir, filename)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		err = ioutil.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	// Add and commit files
	runCommand(t, projectDir, "git", "add", ".")
	runCommand(t, projectDir, "git", "commit", "-m", "Initial commit")

	// Create and checkout a feature branch
	runCommand(t, projectDir, "git", "checkout", "-b", "feature/test")

	// Make some changes on the feature branch
	changeFile := filepath.Join(projectDir, "feature.txt")
	err = ioutil.WriteFile(changeFile, []byte("Feature content\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write feature file: %v", err)
	}

	runCommand(t, projectDir, "git", "add", "feature.txt")
	runCommand(t, projectDir, "git", "commit", "-m", "Add feature")
}

// setupCommitInfo configures the commit information for git-wmem
func setupCommitInfo(t *testing.T, wmemDir string) {
	commitDir := filepath.Join(wmemDir, "md", "commit")
	err := os.MkdirAll(commitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create commit dir: %v", err)
	}

	files := map[string]string{
		"msg-prefix": "git-wmem compatibility test\n\n",
		"author":     "Test User <test@example.com>",
		"committer":  "Test User <test@example.com>",
	}

	for filename, content := range files {
		path := filepath.Join(commitDir, filename)
		err := ioutil.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}
}

// verifyGitCompatibility runs native Git commands on git-wmem created repositories
// to verify they are fully compatible
func verifyGitCompatibility(t *testing.T, wmemDir, projectName string) {
	repoPath := filepath.Join(wmemDir, "repos", projectName+".git")

	// Test 1: git fsck - verify repository integrity
	t.Run("git_fsck", func(t *testing.T) {
		output := runCommand(t, repoPath, "git", "fsck", "--full")
		t.Logf("git fsck output: %s", output)
		// If git fsck fails, runCommand will fail the test
	})

	// Test 2: git log - verify commit history is readable
	t.Run("git_log", func(t *testing.T) {
		output := runCommand(t, repoPath, "git", "log", "--oneline", "--all")
		t.Logf("git log output:\n%s", output)

		// Verify that we have commits
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) < 1 {
			t.Errorf("Expected at least 1 commit, got %d lines", len(lines))
		}
	})

	// Test 3: git show - verify tree objects are readable
	t.Run("git_show_trees", func(t *testing.T) {
		// Get all commit hashes
		output := runCommand(t, repoPath, "git", "rev-list", "--all")
		commits := strings.Fields(strings.TrimSpace(output))

		for _, commit := range commits {
			// Show tree for each commit
			treeOutput := runCommand(t, repoPath, "git", "show", "--name-only", commit)
			t.Logf("Tree for commit %s:\n%s", commit[:7], treeOutput)
		}
	})

	// Test 4: git ls-tree - verify tree sorting is correct
	t.Run("git_ls_tree_sorting", func(t *testing.T) {
		// Get the latest commit from wmem-br/head branch
		latestCommit := strings.TrimSpace(runCommand(t, repoPath, "git", "rev-parse", "wmem-br/head"))

		// List tree entries
		output := runCommand(t, repoPath, "git", "ls-tree", "-r", latestCommit)
		t.Logf("Tree entries for latest commit:\n%s", output)

		// Parse and verify sorting
		lines := strings.Split(strings.TrimSpace(output), "\n")
		var filenames []string
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			// git ls-tree format: mode type hash filename
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				filename := strings.Join(parts[3:], " ")
				filenames = append(filenames, filename)
			}
		}

		// Verify files are sorted in Git order
		for i := 1; i < len(filenames); i++ {
			if strings.Compare(filenames[i-1], filenames[i]) > 0 {
				t.Errorf("Files not sorted correctly: %s should come after %s",
					filenames[i-1], filenames[i])
			}
		}

		t.Logf("Verified %d files are correctly sorted", len(filenames))
	})

	// Test 5: git clone - verify repository can be cloned
	t.Run("git_clone", func(t *testing.T) {
		cloneDir := filepath.Join(filepath.Dir(wmemDir), "cloned-repo")
		defer os.RemoveAll(cloneDir)

		runCommand(t, filepath.Dir(wmemDir), "git", "clone", repoPath, "cloned-repo")

		// Verify clone worked
		clonedFiles, err := ioutil.ReadDir(cloneDir)
		if err != nil {
			t.Fatalf("Failed to read cloned directory: %v", err)
		}

		if len(clonedFiles) == 0 {
			t.Errorf("Cloned repository is empty")
		}

		t.Logf("Successfully cloned repository with %d entries", len(clonedFiles))
	})
}

// runCommand executes a command and returns the output, failing the test on error
func runCommand(t *testing.T, dir string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %s %v\nDir: %s\nOutput: %s\nError: %v",
			name, args, dir, string(output), err)
	}

	return string(output)
}

func TestMain(m *testing.M) {
	// Ensure git is available
	_, err := exec.LookPath("git")
	if err != nil {
		fmt.Fprintf(os.Stderr, "git is not available in PATH: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}
