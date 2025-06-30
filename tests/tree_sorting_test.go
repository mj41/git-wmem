package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestTreeSortingCompatibility specifically tests that git-wmem creates trees
// that match Git's native tree sorting exactly
func TestTreeSortingCompatibility(t *testing.T) {
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
	tempDir, err := ioutil.TempDir("", "git-wmem-tree-sort-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test setup: Create a git-wmem repository and a source project
	wmemDir := filepath.Join(tempDir, "tree-test-wmem")
	projectDir := filepath.Join(tempDir, "tree-test-project")

	// Step 1: Create a source git repository with challenging tree sorting scenarios
	t.Logf("Creating source git repository with complex tree structure at %s", projectDir)
	createComplexTestRepo(t, projectDir)

	// Step 2: Initialize git-wmem repository
	t.Logf("Initializing git-wmem repository at %s", wmemDir)
	runCommandForTreeTest(t, tempDir, gitWmemInit, "tree-test-wmem")

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
	setupCommitInfoForTreeTest(t, wmemDir)

	// Step 5: Create git-wmem commit
	t.Logf("Creating git-wmem commit")
	runCommandForTreeTest(t, wmemDir, gitWmemCommit)

	// Step 6: Compare tree sorting between git-wmem and native git
	compareTreeSorting(t, wmemDir, projectDir, "tree-test-project")
}

// createComplexTestRepo creates a test repository with files that test edge cases in tree sorting
func createComplexTestRepo(t *testing.T, projectDir string) {
	err := os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Initialize git repository
	runCommandForTreeTest(t, projectDir, "git", "init")
	runCommandForTreeTest(t, projectDir, "git", "config", "user.name", "Tree Test User")
	runCommandForTreeTest(t, projectDir, "git", "config", "user.email", "tree-test@example.com")

	// Create test files that specifically test Git's tree sorting behavior
	// These are the same patterns that could cause issues with custom sorting
	testFiles := map[string]string{
		// Basic files
		"README.md": "# Tree Test Project\n",
		"LICENSE":   "MIT License\n",
		"Makefile":  "all:\n\techo test\n",

		// Files vs directories with same prefix - the critical test cases
		"a-file.txt":    "Content in a-file.txt\n",
		"a/info.txt":    "Content in a directory\n",
		"a.txt":         "Content in a.txt\n",
		"aa/nested.txt": "Content in aa directory\n",

		"b.txt":      "Content in b.txt\n",
		"b/data.txt": "Content in b directory\n",
		"bb.txt":     "Content in bb.txt\n",

		"cmd-extra.txt": "Content in cmd-extra.txt\n",
		"cmd/main.go":   "package main\n",
		"cmd.txt":       "Content in cmd.txt\n",

		// Edge cases
		"z-last.txt":   "Should be last\n",
		"00-first.txt": "Should be first numerically\n",
		"~weird.txt":   "Weird character file\n",

		// Nested structure
		"deep/nested/file.txt": "Deep nesting\n",
		"deep/another.txt":     "Another deep file\n",
		"deep.txt":             "File with same name as directory\n",

		// Special characters and case sensitivity
		"Case-Sensitive.txt":     "Case test\n",
		"case-sensitive.txt":     "lowercase case test\n",
		"special-chars_file.txt": "Special chars test\n",
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
	runCommandForTreeTest(t, projectDir, "git", "add", ".")
	runCommandForTreeTest(t, projectDir, "git", "commit", "-m", "Tree sorting test commit")
}

// setupCommitInfoForTreeTest configures the commit information for git-wmem
func setupCommitInfoForTreeTest(t *testing.T, wmemDir string) {
	commitDir := filepath.Join(wmemDir, "md", "commit")
	err := os.MkdirAll(commitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create commit dir: %v", err)
	}

	files := map[string]string{
		"msg-prefix": "Tree sorting compatibility test\n\n",
		"author":     "Tree Test User <tree-test@example.com>",
		"committer":  "Tree Test User <tree-test@example.com>",
	}

	for filename, content := range files {
		path := filepath.Join(commitDir, filename)
		err := ioutil.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}
}

// compareTreeSorting compares tree sorting between git-wmem created repository and native git
func compareTreeSorting(t *testing.T, wmemDir, projectDir, projectName string) {
	gitWmemRepoPath := filepath.Join(wmemDir, "repos", projectName+".git")

	// Get tree entries from git-wmem created repository
	wmemCommit := strings.TrimSpace(runCommandForTreeTest(t, gitWmemRepoPath, "git", "rev-parse", "wmem-br/main"))
	wmemTreeOutput := runCommandForTreeTest(t, gitWmemRepoPath, "git", "ls-tree", "-r", wmemCommit)
	wmemFiles := parseTreeOutput(wmemTreeOutput)

	// Get tree entries from native git repository
	nativeCommit := strings.TrimSpace(runCommandForTreeTest(t, projectDir, "git", "rev-parse", "HEAD"))
	nativeTreeOutput := runCommandForTreeTest(t, projectDir, "git", "ls-tree", "-r", nativeCommit)
	nativeFiles := parseTreeOutput(nativeTreeOutput)

	t.Logf("Comparing tree sorting:")
	t.Logf("git-wmem files (%d): %v", len(wmemFiles), wmemFiles)
	t.Logf("native git files (%d): %v", len(nativeFiles), nativeFiles)

	// Compare that both have the same files in the same order
	if len(wmemFiles) != len(nativeFiles) {
		t.Errorf("Different number of files: git-wmem has %d, native git has %d",
			len(wmemFiles), len(nativeFiles))
	}

	for i := 0; i < len(wmemFiles) && i < len(nativeFiles); i++ {
		if wmemFiles[i] != nativeFiles[i] {
			t.Errorf("File order mismatch at position %d: git-wmem has %q, native git has %q",
				i, wmemFiles[i], nativeFiles[i])
		}
	}

	// Additional test: verify go-git's TreeEntrySorter matches the actual order
	t.Run("verify_go_git_sorter_matches", func(t *testing.T) {
		var entries []object.TreeEntry
		for _, filename := range nativeFiles {
			entries = append(entries, object.TreeEntry{
				Name: filename,
				Mode: filemode.Regular, // Simplified for this test
			})
		}

		// Create a shuffled copy
		shuffledEntries := make([]object.TreeEntry, len(entries))
		copy(shuffledEntries, entries)

		// Shuffle the entries (simple reverse)
		for i, j := 0, len(shuffledEntries)-1; i < j; i, j = i+1, j-1 {
			shuffledEntries[i], shuffledEntries[j] = shuffledEntries[j], shuffledEntries[i]
		}

		// Sort using go-git's TreeEntrySorter
		sort.Sort(object.TreeEntrySorter(shuffledEntries))

		// Verify the order matches
		for i, entry := range shuffledEntries {
			if i >= len(nativeFiles) {
				t.Errorf("go-git sorter produced more entries than expected")
				break
			}
			if entry.Name != nativeFiles[i] {
				t.Errorf("go-git sorter mismatch at position %d: got %q, want %q",
					i, entry.Name, nativeFiles[i])
			}
		}

		t.Logf("✓ go-git's TreeEntrySorter produces the same order as native Git")
	})

	if len(wmemFiles) == len(nativeFiles) {
		allMatch := true
		for i, file := range wmemFiles {
			if file != nativeFiles[i] {
				allMatch = false
				break
			}
		}
		if allMatch {
			t.Logf("✓ git-wmem tree sorting perfectly matches native Git")
		}
	}
}

// parseTreeOutput parses git ls-tree output and returns just the filenames in order
func parseTreeOutput(output string) []string {
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
	return filenames
}

// runCommandForTreeTest executes a command and returns the output, failing the test on error
func runCommandForTreeTest(t *testing.T, dir string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %s %v\nDir: %s\nOutput: %s\nError: %v",
			name, args, dir, string(output), err)
	}

	return string(output)
}
