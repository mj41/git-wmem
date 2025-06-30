package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestValidations_WorkdirPathRequirements tests workdir path validation rules
// Reference: docs/validations.md#workdir-path-requirements
func TestValidations_WorkdirPathRequirements(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo
	wmemDir := setupBasicWmemRepo(h)
	h.SetWorkDir(wmemDir)

	testCases := []struct {
		name        string
		path        string
		shouldError bool
		errorText   string
		description string
	}{
		{
			name:        "valid_relative_path",
			path:        "../my-projectA",
			shouldError: false,
			description: "Valid relative path starting with ../",
		},
		{
			name:        "valid_nested_relative_path",
			path:        "../other-projects/my-projectB",
			shouldError: false,
			description: "Valid nested relative path",
		},
		{
			name:        "valid_deep_relative_path",
			path:        "../../../workspace/external-project",
			shouldError: false,
			description: "Valid deep relative path",
		},
		{
			name:        "invalid_absolute_path",
			path:        "/absolute/path/to/project",
			shouldError: true,
			errorText:   "Absolute paths not allowed",
			description: "Invalid absolute path",
		},
		{
			name:        "invalid_excessive_traversal",
			path:        "../my-project/../../../etc",
			shouldError: true,
			errorText:   "path traversal",
			description: "Invalid excessive path traversal",
		},
		{
			name:        "invalid_local_subdir",
			path:        "./relative-subdir",
			shouldError: true,
			errorText:   "wmem-repo paths not allowed",
			description: "Invalid local subdirectory path",
		},
		{
			name:        "invalid_plain_relative",
			path:        "some-dir",
			shouldError: true,
			errorText:   "Must start with ../",
			description: "Invalid plain relative path",
		},
	}

	// Create a valid test project for successful cases
	validProject := filepath.Join(h.TempDir(), "my-projectA")
	h.MkdirAll(validProject)
	h.SetWorkDir(validProject)
	_, err := h.RunGit("init")
	h.AssertCommandSuccess("", err, "git init valid project")
	h.WriteFile("file.txt", "content")
	_, err = h.RunGit("add", "file.txt")
	h.AssertCommandSuccess("", err, "git add")
	_, err = h.RunGit("commit", "-m", "Initial commit")
	h.AssertCommandSuccess("", err, "git commit")

	h.SetWorkDir(wmemDir)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear workdir paths file
			h.WriteFile("md/commit-workdir-paths", "")

			// Only add the valid project for successful test cases
			if !tc.shouldError {
				h.AppendToFile("md/commit-workdir-paths", "../my-projectA")
			} else {
				h.AppendToFile("md/commit-workdir-paths", tc.path)
			}

			// Try to commit
			output, err := h.RunGitWmem("commit")

			if tc.shouldError {
				h.AssertCommandError(output, err, tc.errorText, tc.description)
			} else {
				h.AssertCommandSuccess(output, err, tc.description)
			}
		})
	}
}

// TestValidations_BranchNameRequirements tests branch naming patterns
// Reference: docs/validations.md#branch-name-requirements
func TestValidations_BranchNameRequirements(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo and test project
	wmemDir := setupBasicWmemRepo(h)

	// Create project with different branch names
	testBranches := []struct {
		branchName     string
		expectedWmemBr string
		description    string
	}{
		{
			branchName:     "main",
			expectedWmemBr: "wmem-br/main",
			description:    "Simple branch name",
		},
		{
			branchName:     "feat/X1",
			expectedWmemBr: "wmem-br/feat/X1",
			description:    "Feature branch with slash",
		},
		{
			branchName:     "dev-branch",
			expectedWmemBr: "wmem-br/dev-branch",
			description:    "Branch with hyphen",
		},
	}

	for _, tc := range testBranches {
		t.Run(tc.branchName, func(t *testing.T) {
			// Create fresh project for each test
			// Replace slashes in branch name for directory naming
			safeBranchName := strings.ReplaceAll(tc.branchName, "/", "-")
			testProjectDir := filepath.Join(h.TempDir(), "test-project-"+safeBranchName)
			h.MkdirAll(testProjectDir)
			h.SetWorkDir(testProjectDir)

			_, err := h.RunGit("init")
			h.AssertCommandSuccess("", err, "git init")

			h.WriteFile("file.txt", "content")
			_, err = h.RunGit("add", "file.txt")
			h.AssertCommandSuccess("", err, "git add")

			_, err = h.RunGit("commit", "-m", "Initial commit")
			h.AssertCommandSuccess("", err, "git commit")

			// Create the test branch
			if tc.branchName != "main" {
				_, err = h.RunGit("checkout", "-b", tc.branchName)
				h.AssertCommandSuccess("", err, "git checkout -b "+tc.branchName)
			}

			// Setup wmem to track this project
			h.SetWorkDir(wmemDir)
			relPath := "../" + filepath.Base(testProjectDir)
			h.WriteFile("md/commit-workdir-paths", relPath)

			// Run wmem commit
			output, err := h.RunGitWmem("commit")
			h.AssertCommandSuccess(output, err, "git-wmem-commit for "+tc.description)

			// Check that the bare repository has the correct wmem branch
			bareRepoDir := filepath.Join(wmemDir, "repos", filepath.Base(testProjectDir)+".git")
			h.SetWorkDir(bareRepoDir)

			output, err = h.RunGit("branch", "-a")
			h.AssertCommandSuccess(output, err, "git branch in bare repo")

			// Should contain the wmem-br prefixed branch in the git branch output
			if !strings.Contains(output, tc.expectedWmemBr) {
				h.t.Errorf("Expected branch %s not found in git branch output: %s", tc.expectedWmemBr, output)
			}
		})
	}
}

// TestValidations_WmemRepoDetection tests detection of wmem-repo subdirectories
// Reference: docs/validations.md#workdir-path-requirements
func TestValidations_WmemRepoDetection(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	// Setup wmem repo
	wmemDir := setupBasicWmemRepo(h)
	h.SetWorkDir(wmemDir)

	// Try to add wmem-repo itself as workdir
	h.WriteFile("md/commit-workdir-paths", ".")
	output, err := h.RunGitWmem("commit")
	h.AssertCommandError(output, err, "wmem-repo", "Cannot point to wmem-repo itself")

	// Try to add wmem-repo subdirectory as workdir
	h.MkdirAll("subdir")
	h.WriteFile("md/commit-workdir-paths", "./subdir")
	output, err = h.RunGitWmem("commit")
	h.AssertCommandError(output, err, "wmem-repo", "Cannot point to wmem-repo subdirs")
}
