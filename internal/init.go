package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// InitWmemRepo initializes a new wmem repository
// Reference: docs/use-cases/git-wmem-init/basic.md#main-scenario
func InitWmemRepo(targetDir string) error {
	// Check if directory exists and if it should be created
	if targetDir == "." {
		// Current directory case - check if empty
		entries, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("Directory is not empty. Please specify an empty directory to initialize wmem-repo.")
		}
	} else {
		// New directory case - check if it exists
		if _, err := os.Stat(targetDir); err == nil {
			// Directory exists, check if empty
			entries, err := os.ReadDir(targetDir)
			if err != nil {
				return fmt.Errorf("failed to read directory %s: %w", targetDir, err)
			}
			if len(entries) > 0 {
				return fmt.Errorf("Directory is not empty. Please specify an empty directory to initialize wmem-repo.")
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check directory %s: %w", targetDir, err)
		} else {
			// Directory doesn't exist, create it
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
			}
		}
	}

	// Change to target directory for operations
	var workDir string
	if targetDir == "." {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		workDir = wd
	} else {
		abs, err := filepath.Abs(targetDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		workDir = abs
		if err := os.Chdir(workDir); err != nil {
			return fmt.Errorf("failed to change to directory %s: %w", workDir, err)
		}
	}

	// Create the directory structure
	if err := createWmemStructure(); err != nil {
		return fmt.Errorf("failed to create wmem structure: %w", err)
	}

	// Initialize git repository
	repo, err := git.PlainInit(workDir, false)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Create initial commit
	if err := createInitialCommit(repo, filepath.Base(workDir)); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	return nil
}

// createWmemStructure creates the directory structure for wmem repository
func createWmemStructure() error {
	// Create .git-wmem marker file
	if err := os.WriteFile(".git-wmem", []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to create .git-wmem file: %w", err)
	}

	// Create .gitignore
	gitignoreContent := "repos/\n"
	if err := os.WriteFile(".gitignore", []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Create directories
	dirs := []string{"md", "md/commit", "md-internal", "repos"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create metadata files
	files := map[string]string{
		"md/commit-workdir-paths":      "",
		"md/commit/msg-prefix":         "",
		"md/commit/author":             "WMem Git <git-wmem@mj41.cz>",
		"md/commit/committer":          "WMem Git <git-wmem@mj41.cz>",
		"md-internal/workdir-map.json": "{}",
	}

	for filePath, content := range files {
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
	}

	return nil
}

// createInitialCommit creates the initial commit in the wmem repository
func createInitialCommit(repo *git.Repository, repoName string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all files
	if err := worktree.AddGlob("."); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	// Create commit
	commitMsg := fmt.Sprintf("Initialize git-wmem repository `%s`", repoName)

	commit, err := worktree.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "WMem Git",
			Email: "git-wmem@mj41.cz",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Set main branch as default
	headRef, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	mainRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/main"), commit)
	if err := repo.Storer.SetReference(mainRef); err != nil {
		return fmt.Errorf("failed to set main branch: %w", err)
	}

	// Update HEAD to point to main
	headRef = plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/main"))
	if err := repo.Storer.SetReference(headRef); err != nil {
		return fmt.Errorf("failed to update HEAD: %w", err)
	}

	return nil
}
