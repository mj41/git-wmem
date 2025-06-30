package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// isWmemRepo checks if current directory is a wmem repository
func isWmemRepo() bool {
	_, err := os.Stat(".git-wmem")
	return err == nil
}

// initRepos implements the init-repos sub-operation
// Reference: docs/use-cases/git-wmem-commit/basic.md#uc-git-wmem-commit-init-repos
func initRepos(workdirPaths []string) error {
	// Read existing workdir map
	workdirMap, err := readWorkdirMap()
	if err != nil {
		return fmt.Errorf("failed to read workdir map: %w", err)
	}

	for _, workdirPath := range workdirPaths {
		// Validate the workdir path
		if err := validateWorkdirPath(workdirPath); err != nil {
			return fmt.Errorf("invalid workdir path %s: %w", workdirPath, err)
		}

		// Check if workdir is already in the map
		if _, exists := FindWorkdirName(workdirPath, workdirMap); exists {
			continue // Already initialized
		}

		// Generate workdir name
		workdirName := generateWorkdirName(workdirPath, workdirMap)

		// Create bare repository
		if err := createBareRepo(workdirName, workdirPath); err != nil {
			return fmt.Errorf("failed to create bare repo for %s: %w", workdirPath, err)
		}

		// Update workdir map (name -> path mapping)
		// Normalize path to ensure consistent handling of trailing slashes
		workdirMap[workdirName] = filepath.Clean(workdirPath)
	}

	// Save updated workdir map
	if err := saveWorkdirMap(workdirMap); err != nil {
		return fmt.Errorf("failed to save workdir map: %w", err)
	}

	return nil
}

// createBareRepo creates a bare repository for the workdir
func createBareRepo(workdirName, workdirPath string) error {
	repoPath := filepath.Join("repos", workdirName+".git")

	// Create bare repository
	_, err := git.PlainInit(repoPath, true)
	if err != nil {
		return fmt.Errorf("failed to create bare repository: %w", err)
	}

	// Open the bare repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open bare repository: %w", err)
	}

	// Add remote pointing to workdir
	absWorkdirPath, err := filepath.Abs(workdirPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute workdir path: %w", err)
	}

	remoteConfig := &config.RemoteConfig{
		Name: "wmem-wd",
		URLs: []string{absWorkdirPath},
	}

	_, err = repo.CreateRemote(remoteConfig)
	if err != nil {
		return fmt.Errorf("failed to create remote: %w", err)
	}

	// Fetch from workdir
	remote, err := repo.Remote("wmem-wd")
	if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	err = remote.Fetch(&git.FetchOptions{})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch from workdir: %w", err)
	}

	// Get current branch from workdir and create wmem-br/ branch
	if err := createWmemBranch(repo, absWorkdirPath); err != nil {
		return fmt.Errorf("failed to create wmem branch: %w", err)
	}

	return nil
}

// createWmemBranch creates wmem-br/<branch> from workdir's current branch
func createWmemBranch(repo *git.Repository, workdirPath string) error {
	// Open workdir repository to get current branch
	workdirRepo, err := git.PlainOpen(workdirPath)
	if err != nil {
		return fmt.Errorf("failed to open workdir repository: %w", err)
	}

	// Get current branch name
	head, err := workdirRepo.Head()
	if err != nil {
		return fmt.Errorf("failed to get workdir HEAD: %w", err)
	}

	branchName := head.Name().Short()
	wmemBranchName := fmt.Sprintf("wmem-br/%s", branchName)

	// Create wmem branch pointing to the same commit
	wmemBranchRef := plumbing.NewHashReference(
		plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName)),
		head.Hash(),
	)

	err = repo.Storer.SetReference(wmemBranchRef)
	if err != nil {
		return fmt.Errorf("failed to create wmem branch: %w", err)
	}

	return nil
}

// ensureWmemBranchExists implements step 2 of UC: sync-workdir (Alternative 2b)
func ensureWmemBranchExists(workdirName, currentBranchName, workdirPath string) error {
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open bare repository: %w", err)
	}

	wmemBranchName := fmt.Sprintf("wmem-br/%s", currentBranchName)
	wmemBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName))

	// Check if wmem-br/<current-branch-name> branch exists
	_, err = bareRepo.Reference(wmemBranchRef, true)
	if err != nil {
		// Alternative 2b: Create new branch pointing to current workdir HEAD commit
		absWorkdirPath, err := filepath.Abs(workdirPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute workdir path: %w", err)
		}

		workdirRepo, err := git.PlainOpen(absWorkdirPath)
		if err != nil {
			return fmt.Errorf("failed to open workdir repository: %w", err)
		}

		head, err := workdirRepo.Head()
		if err != nil {
			return fmt.Errorf("failed to get workdir HEAD: %w", err)
		}

		// Create new wmem branch
		newWmemBranchRef := plumbing.NewHashReference(wmemBranchRef, head.Hash())
		err = bareRepo.Storer.SetReference(newWmemBranchRef)
		if err != nil {
			return fmt.Errorf("failed to create wmem branch: %w", err)
		}
	}

	return nil
}

// ensureWmemHeadBranch implements step 3 of UC: sync-workdir
func ensureWmemHeadBranch(workdirName, currentBranchName string) error {
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open bare repository: %w", err)
	}

	wmemBranchName := fmt.Sprintf("wmem-br/%s", currentBranchName)
	wmemBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName))

	// Get wmem-br/<current-branch-name> reference
	wmemBranchHashRef, err := bareRepo.Reference(wmemBranchRef, true)
	if err != nil {
		return fmt.Errorf("failed to get wmem branch reference: %w", err)
	}

	// Set HEAD to point to wmem-br/<current-branch-name>
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, wmemBranchRef)
	err = bareRepo.Storer.SetReference(headRef)
	if err != nil {
		return fmt.Errorf("failed to set HEAD to wmem branch: %w", err)
	}

	fmt.Printf("Debug: Set HEAD to wmem-br/%s (%s)\n", currentBranchName, wmemBranchHashRef.Hash().String()[:12])
	return nil
}

// fetchLatestChanges implements step 4 of UC: sync-workdir
func fetchLatestChanges(workdirName string) error {
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open bare repository: %w", err)
	}

	remote, err := bareRepo.Remote("wmem-wd")
	if err != nil {
		return fmt.Errorf("failed to get workdir remote: %w", err)
	}

	err = remote.Fetch(&git.FetchOptions{})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch latest changes: %w", err)
	}

	return nil
}
