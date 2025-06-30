package internal

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Type definitions and cache instances have been moved to types.go

// workdirCheckResult holds the result of parallel workdir checking
type workdirCheckResult struct {
	WorkdirPath       string
	WorkdirName       string
	CurrentBranchName string
	HasModifiedFiles  bool
	Error             error
}

// CommitWmem performs the main git-wmem-commit operation
// Reference: docs/use-cases/git-wmem-commit/basic.md
func CommitWmem() error {
	// Check if we're in a wmem-repo
	if !isWmemRepo() {
		return fmt.Errorf("not in a wmem repository (missing .git-wmem file). Run this command from a wmem-repo directory.")
	}

	// Check if workdir paths are configured
	workdirPaths, err := readWorkdirPaths()
	if err != nil {
		return fmt.Errorf("failed to read workdir paths: %w", err)
	}

	if len(workdirPaths) == 0 {
		return fmt.Errorf("No workdirs configured for commit. Add paths to your workdirs in md/commit-workdir-paths file.")
	}

	// Perform init-repos operation
	if err := initRepos(workdirPaths); err != nil {
		return fmt.Errorf("failed to init repos: %w", err)
	}

	// Perform commit-all operation
	if err := commitAll(workdirPaths); err != nil {
		return fmt.Errorf("failed to commit all: %w", err)
	}

	return nil
}

// Repository operations have been moved to repo.go

// commitAll implements the commit-all sub-operation
// Reference: docs/use-cases/git-wmem-commit/basic.md#uc-git-wmem-commit-commit-all
func commitAll(workdirPaths []string) error {
	// Read commit info
	commitInfo, err := readCommitInfo()
	if err != nil {
		return fmt.Errorf("failed to read commit info: %w", err)
	}

	// Read workdir map
	workdirMap, err := readWorkdirMap()
	if err != nil {
		return fmt.Errorf("failed to read workdir map: %w", err)
	}

	// Phase 1: Run initial checks in parallel to determine which workdirs have changes
	// For single workdir, skip parallel overhead and run directly
	var checkResults []workdirCheckResult
	if len(workdirPaths) == 1 {
		fmt.Printf("Info: Processing single workdir %s\n", workdirPaths[0])
		result := checkWorkdirInParallel(workdirPaths[0], workdirMap, commitInfo)
		checkResults = []workdirCheckResult{result}
	} else {
		fmt.Printf("Info: Running parallel checks on %d workdir(s)\n", len(workdirPaths))
		checkResults = runParallelWorkdirChecks(workdirPaths, workdirMap, commitInfo)
	}

	// Phase 2: Process workdirs with changes sequentially to avoid race conditions
	var workdirResults []WorkdirCommitResult
	hasAnyChanges := false

	for _, checkResult := range checkResults {
		if checkResult.Error != nil {
			return fmt.Errorf("failed to check workdir %s: %w", checkResult.WorkdirPath, checkResult.Error)
		}

		if !checkResult.HasModifiedFiles {
			fmt.Printf("Info: No modified files in workdir %s, skipping commit creation\n", checkResult.WorkdirPath)
			workdirResults = append(workdirResults, WorkdirCommitResult{
				WorkdirName: checkResult.WorkdirName,
				BranchName:  checkResult.CurrentBranchName,
				CommitHash:  "", // No new commit created
				HasChanges:  false,
			})
			continue
		}

		// Process workdir with changes (steps 7-9 of UC: sync-workdir)
		result, err := commitWorkdirWithChanges(checkResult.WorkdirPath, checkResult.WorkdirName, checkResult.CurrentBranchName, commitInfo)
		if err != nil {
			return fmt.Errorf("failed to commit workdir %s: %w", checkResult.WorkdirPath, err)
		}
		workdirResults = append(workdirResults, result)

		// Track if any workdir has changes
		if result.HasChanges {
			hasAnyChanges = true
		}
	}

	// Only create wmem-repo commit if there are actual changes in at least one workdir
	// or if there are metadata changes in the wmem-repo itself
	if hasAnyChanges {
		if err := createWmemCommit(commitInfo, workdirResults); err != nil {
			return fmt.Errorf("failed to create wmem commit: %w", err)
		}
		fmt.Printf("Info: Created wmem-repo commit with changes from %d workdir(s)\n", countChangedWorkdirs(workdirResults))
	} else {
		// Check if there are metadata changes that should trigger a wmem-repo commit
		hasMetadataChanges, err := hasWmemRepoMetadataChanges()
		if err != nil {
			return fmt.Errorf("failed to check wmem-repo metadata changes: %w", err)
		}

		if hasMetadataChanges {
			if err := createWmemCommit(commitInfo, workdirResults); err != nil {
				return fmt.Errorf("failed to create wmem commit: %w", err)
			}
			fmt.Printf("Info: Created wmem-repo commit due to metadata changes (no workdir changes)\n")
		} else {
			fmt.Printf("Info: No changes detected in any workdir or metadata, skipping wmem-repo commit creation\n")
		}
	}

	// Print cache statistics at the end
	printCacheStats()

	return nil
}

// readCommitInfo reads commit information from md/commit/ files
func readCommitInfo() (*CommitInfo, error) {
	// Generate wmem-uid
	wmemUID, err := generateWmemUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate wmem-uid: %w", err)
	}

	// Read message prefix
	msgPrefix, err := os.ReadFile("md/commit/msg-prefix")
	if err != nil {
		return nil, fmt.Errorf("failed to read msg-prefix: %w", err)
	}

	// Read author
	author, err := os.ReadFile("md/commit/author")
	if err != nil {
		return nil, fmt.Errorf("failed to read author: %w", err)
	}

	// Read committer
	committer, err := os.ReadFile("md/commit/committer")
	if err != nil {
		return nil, fmt.Errorf("failed to read committer: %w", err)
	}

	// Build commit message
	message := strings.TrimSpace(string(msgPrefix))
	if message != "" {
		message += "\n\n"
	}
	message += fmt.Sprintf("wmem-uid: %s", wmemUID)

	return &CommitInfo{
		WmemUID:   wmemUID,
		Message:   message,
		Author:    strings.TrimSpace(string(author)),
		Committer: strings.TrimSpace(string(committer)),
	}, nil
}

// generateWmemUID generates a unique wmem-uid
// Reference: docs/data-structures.md#wmem-uid
func generateWmemUID() (string, error) {
	now := time.Now()

	// Format: wmem-YYMMDD-HHMMSS-abXY1234
	datePart := now.Format("060102")
	timePart := now.Format("150405")

	// Generate 8-character random string [a-zA-Z0-9]
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomPart := make([]byte, 8)
	randomBytes := make([]byte, 8)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	for i, b := range randomBytes {
		randomPart[i] = charset[b%byte(len(charset))]
	}

	return fmt.Sprintf("wmem-%s-%s-%s", datePart, timePart, string(randomPart)), nil
}

// runParallelWorkdirChecks runs initial checks (steps 1-6) on all workdirs in parallel
func runParallelWorkdirChecks(workdirPaths []string, workdirMap WorkdirMap, commitInfo *CommitInfo) []workdirCheckResult {
	results := make([]workdirCheckResult, len(workdirPaths))
	var wg sync.WaitGroup

	for i, workdirPath := range workdirPaths {
		wg.Add(1)
		go func(index int, path string) {
			defer wg.Done()
			results[index] = checkWorkdirInParallel(path, workdirMap, commitInfo)
		}(i, workdirPath)
	}

	wg.Wait()
	return results
}

// checkWorkdirInParallel performs steps 1-6 of UC: sync-workdir in parallel
func checkWorkdirInParallel(workdirPath string, workdirMap WorkdirMap, commitInfo *CommitInfo) workdirCheckResult {
	result := workdirCheckResult{
		WorkdirPath: workdirPath,
	}

	// Find workdir name
	workdirName, exists := FindWorkdirName(workdirPath, workdirMap)
	if !exists {
		result.Error = fmt.Errorf("workdir %s not found in workdir map", workdirPath)
		return result
	}
	result.WorkdirName = workdirName

	// Step 1: Get the current branch name of workdir-path
	currentBranchName, err := getCurrentBranchName(workdirPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to get current branch name: %w", err)
		return result
	}
	result.CurrentBranchName = currentBranchName

	// Step 2: Ensure wmem-br/<current-branch-name> branch exists in wmem-wd-repo
	err = ensureWmemBranchExists(workdirName, currentBranchName, workdirPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to ensure wmem-br branch exists: %w", err)
		return result
	}

	// Step 3: Ensure wmem-br/head branch exists and points to current wmem-br/<current-branch-name>
	err = ensureWmemHeadBranch(workdirName, currentBranchName)
	if err != nil {
		result.Error = fmt.Errorf("failed to ensure wmem-br/head branch: %w", err)
		return result
	}

	// Step 4: Fetch latest changes from wmem-wd remote repo
	err = fetchLatestChanges(workdirName)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch latest changes: %w", err)
		return result
	}

	// Step 5: Ensure that wmem-wd current-branch-name commit is already merged to wmem-wd-repo's wmem-br/<current-branch-name> branch
	_, err = ensureWorkdirCommitMerged(workdirPath, workdirName, currentBranchName, commitInfo)
	if err != nil {
		result.Error = fmt.Errorf("failed to ensure workdir commit merged: %w", err)
		return result
	}

	// Step 6: Check that there are modified files in the workdir-path (Alternative 6b)
	hasModifiedFiles, err := checkModifiedFiles(workdirPath, workdirName, currentBranchName)
	if err != nil {
		result.Error = fmt.Errorf("failed to check modified files: %w", err)
		return result
	}
	result.HasModifiedFiles = hasModifiedFiles

	return result
}

// commitWorkdirWithChanges performs steps 7-9 of UC: sync-workdir for workdirs with changes
func commitWorkdirWithChanges(workdirPath, workdirName, currentBranchName string, commitInfo *CommitInfo) (WorkdirCommitResult, error) {
	// Step 7: Add all files (like git add -A) in workdir-path to the index in wmem-wd-repo
	// Step 8: Create a new commit to wmem-br/<current-branch-name> branch
	newCommitHash, err := addFilesAndCommit(workdirPath, workdirName, currentBranchName, commitInfo)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to add files and commit: %w", err)
	}

	// Step 9: Update wmem-br/head to point to the new commit
	err = updateWmemHeadBranch(workdirName, newCommitHash)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to update wmem-br/head: %w", err)
	}

	fmt.Printf("Info: Successfully committed changes in workdir %s to wmem-br/%s\n", workdirPath, currentBranchName)
	return WorkdirCommitResult{
		WorkdirName: workdirName,
		BranchName:  currentBranchName,
		CommitHash:  newCommitHash.String(),
		HasChanges:  true,
	}, nil
}

// commitWorkdir implements UC: sync-workdir
// Reference: docs/use-cases/git-wmem-commit/basic.md#uc-sync-workdir
func commitWorkdir(workdirPath, workdirName string, commitInfo *CommitInfo) (WorkdirCommitResult, error) {
	// Step 1: Get the current branch name of workdir-path
	currentBranchName, err := getCurrentBranchName(workdirPath)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to get current branch name: %w", err)
	}

	// Step 2: Ensure wmem-br/<current-branch-name> branch exists in wmem-wd-repo
	err = ensureWmemBranchExists(workdirName, currentBranchName, workdirPath)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to ensure wmem-br branch exists: %w", err)
	}

	// Step 3: Ensure wmem-br/head branch exists and points to current wmem-br/<current-branch-name>
	err = ensureWmemHeadBranch(workdirName, currentBranchName)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to ensure wmem-br/head branch: %w", err)
	}

	// Step 4: Fetch latest changes from wmem-wd remote repo
	err = fetchLatestChanges(workdirName)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to fetch latest changes: %w", err)
	}

	// Step 5: Ensure that wmem-wd current-branch-name commit is already merged to wmem-wd-repo's wmem-br/<current-branch-name> branch
	_, err = ensureWorkdirCommitMerged(workdirPath, workdirName, currentBranchName, commitInfo)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to ensure workdir commit merged: %w", err)
	}

	// Step 6: Check that there are modified files in the workdir-path (Alternative 6b)
	hasModifiedFiles, err := checkModifiedFiles(workdirPath, workdirName, currentBranchName)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to check modified files: %w", err)
	}

	if !hasModifiedFiles {
		fmt.Printf("Info: No modified files in workdir %s, skipping commit creation\n", workdirPath)
		return WorkdirCommitResult{
			WorkdirName: workdirName,
			BranchName:  currentBranchName,
			CommitHash:  "", // No new commit created
			HasChanges:  false,
		}, nil
	}

	// Step 7: Add all files (like git add -A) in workdir-path to the index in wmem-wd-repo
	// Step 8: Create a new commit to wmem-br/<current-branch-name> branch
	newCommitHash, err := addFilesAndCommit(workdirPath, workdirName, currentBranchName, commitInfo)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to add files and commit: %w", err)
	}

	// Step 9: Update wmem-br/head to point to the new commit
	err = updateWmemHeadBranch(workdirName, newCommitHash)
	if err != nil {
		return WorkdirCommitResult{}, fmt.Errorf("failed to update wmem-br/head: %w", err)
	}

	fmt.Printf("Info: Successfully committed changes in workdir %s to wmem-br/%s\n", workdirPath, currentBranchName)
	return WorkdirCommitResult{
		WorkdirName: workdirName,
		BranchName:  currentBranchName,
		CommitHash:  newCommitHash.String(),
		HasChanges:  true,
	}, nil
}

// ensureBranchNameMatches implements step 1 of UC: sync-workdir
// Alternative 1b: Creates wmem-br/<current-branch-name> if it doesn't match pattern

// ensureWorkdirCommitMerged implements step 5 of UC: sync-workdir (Alternative 5b)
func ensureWorkdirCommitMerged(workdirPath, workdirName, currentBranchName string, commitInfo *CommitInfo) (bool, error) {
	absWorkdirPath, err := filepath.Abs(workdirPath)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute workdir path: %w", err)
	}

	workdirRepo, err := git.PlainOpen(absWorkdirPath)
	if err != nil {
		return false, fmt.Errorf("failed to open workdir repository: %w", err)
	}

	head, err := workdirRepo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get workdir HEAD: %w", err)
	}

	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to open bare repository: %w", err)
	}

	wmemBranchName := fmt.Sprintf("wmem-br/%s", currentBranchName)
	wmemBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName))

	wmemBranchHashRef, err := bareRepo.Reference(wmemBranchRef, true)
	if err != nil {
		return false, fmt.Errorf("failed to get wmem branch reference: %w", err)
	}

	// Check if workdir HEAD commit is already merged
	isAlreadyMerged, err := isCommitMerged(bareRepo, head.Hash(), wmemBranchHashRef.Hash())
	if err != nil {
		return false, fmt.Errorf("failed to check if commit is merged: %w", err)
	}

	if !isAlreadyMerged {
		// Alternative 5b: Create merge commit following ALG: wmem merge
		authorSig, committerSig, err := parseCommitSignatures(commitInfo)
		if err != nil {
			return false, fmt.Errorf("failed to parse commit signatures: %w", err)
		}

		newCommitHash, err := createWmemMergeCommit(bareRepo, wmemBranchHashRef.Hash(), head.Hash(), currentBranchName, commitInfo, authorSig, committerSig)
		if err != nil {
			return false, fmt.Errorf("failed to create merge commit: %w", err)
		}

		// Update wmem-br/<current-branch-name> to point to new merge commit
		newWmemBranchRef := plumbing.NewHashReference(wmemBranchRef, newCommitHash)
		err = bareRepo.Storer.SetReference(newWmemBranchRef)
		if err != nil {
			return false, fmt.Errorf("failed to update wmem branch: %w", err)
		}

		// Update wmem-br/head to point to new merge commit
		wmemHeadRef := plumbing.ReferenceName("refs/heads/wmem-br/head")
		newWmemHeadRef := plumbing.NewHashReference(wmemHeadRef, newCommitHash)
		err = bareRepo.Storer.SetReference(newWmemHeadRef)
		if err != nil {
			return false, fmt.Errorf("failed to update wmem-br/head: %w", err)
		}

		fmt.Printf("Info: Created merge commit for workdir %s into wmem-br/%s\n", workdirPath, currentBranchName)
	}

	return isAlreadyMerged, nil
}

// getTouchedFilesSinceMerge gets all files that have been touched since the last merge
// Implements optimization to only track files that have actually changed
// Implementation: docs/optimizations.md#touched-files-optimization
func getTouchedFilesSinceMerge(workdirPath string, lastMergeHash plumbing.Hash) ([]string, error) {
	// Open workdir repository
	workdirRepo, err := git.PlainOpen(workdirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workdir repository: %w", err)
	}

	// Get HEAD commit
	headRef, err := workdirRepo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	headCommit, err := workdirRepo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	// Get last merge commit
	lastMergeCommit, err := workdirRepo.CommitObject(lastMergeHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get last merge commit: %w", err)
	}

	// Get diff between trees
	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD tree: %w", err)
	}

	lastMergeTree, err := lastMergeCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get last merge tree: %w", err)
	}

	changes, err := lastMergeTree.Diff(headTree)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree diff: %w", err)
	}

	var files []string
	for _, change := range changes {
		if change.To.Name != "" {
			files = append(files, change.To.Name)
		}
		if change.From.Name != "" && change.From.Name != change.To.Name {
			// Handle renames - include both old and new names
			files = append(files, change.From.Name)
		}
	}

	return files, nil
}

// createTreeFromCurrentState creates a git tree from the current workdir state
// Optimized replacement for filesystem-based tree creation
func createTreeFromCurrentState(workdirPath string, targetRepo *git.Repository) (plumbing.Hash, error) {
	// Handle nested git repos correctly maintaining gitlink support
	// Use the filesystem-based approach to maintain gitlink handling
	absWorkdirPath, err := filepath.Abs(workdirPath)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get absolute workdir path: %w", err)
	}

	// Use the createTreeFromFilesystem which handles gitlinks correctly
	return createTreeFromFilesystem(targetRepo, absWorkdirPath)
}

// findLastMergeCommit finds the most recent merge commit in the branch history
// A merge commit is defined as a commit with exactly two parents
func findLastMergeCommit(repo *git.Repository, startHash plumbing.Hash) (plumbing.Hash, error) {
	commit, err := repo.CommitObject(startHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get start commit: %w", err)
	}

	// Walk through commit history looking for merge commits
	iter := object.NewCommitIterCTime(commit, nil, nil)
	defer iter.Close()

	for {
		c, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return plumbing.ZeroHash, fmt.Errorf("failed to iterate commits: %w", err)
		}

		// Check if this commit has exactly two parents (merge commit)
		if c.NumParents() == 2 {
			return c.Hash, nil
		}
	}

	return plumbing.ZeroHash, fmt.Errorf("no merge commit found in branch history")
}

// hasWorkingDirectoryChanges checks if workdir has any unstaged or staged changes
func hasWorkingDirectoryChanges(workdirPath string) (bool, error) {
	workdirRepo, err := git.PlainOpen(workdirPath)
	if err != nil {
		return false, fmt.Errorf("failed to open workdir repository: %w", err)
	}

	worktree, err := workdirRepo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	return !status.IsClean(), nil
}

// isHeadUnchangedSinceLastWmemCommit checks if the current HEAD of workdir
// is the same as what was last processed in the wmem branch
func isHeadUnchangedSinceLastWmemCommit(workdirPath, workdirName, currentBranchName string) (bool, error) {
	// Get current HEAD of workdir
	workdirRepo, err := git.PlainOpen(workdirPath)
	if err != nil {
		return false, fmt.Errorf("failed to open workdir repository: %w", err)
	}

	headRef, err := workdirRepo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD: %w", err)
	}
	currentHead := headRef.Hash()

	// Get the last commit in wmem-br/<current-branch-name>
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to open bare repository: %w", err)
	}

	wmemBranchName := fmt.Sprintf("wmem-br/%s", currentBranchName)
	wmemBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName))

	wmemBranchHashRef, err := bareRepo.Reference(wmemBranchRef, true)
	if err != nil {
		// If wmem branch doesn't exist yet, HEAD has definitely changed
		return false, nil
	}

	// Get the wmem commit and find the original workdir commit it was based on
	wmemCommit, err := bareRepo.CommitObject(wmemBranchHashRef.Hash())
	if err != nil {
		return false, fmt.Errorf("failed to get wmem commit: %w", err)
	}

	// Check if wmem commit message contains the current HEAD hash
	// Format: "Commit from workdir: <original-hash>"
	expectedMessage := fmt.Sprintf("Commit from workdir: %s", currentHead.String())
	if wmemCommit.Message == expectedMessage {
		return true, nil // Same HEAD as last processed
	}

	// Alternative: Check if current HEAD is in the wmem commit parents
	// This handles the case where the wmem commit was based on current HEAD
	for _, parent := range wmemCommit.ParentHashes {
		if parent == currentHead {
			return true, nil
		}
	}

	return false, nil // HEAD has moved since last wmem commit
}

// checkModifiedFiles implements step 6 of UC: sync-workdir
// Compares the current filesystem state in workdir with wmem-repo's wmem-br/<current-branch-name> branch
// Uses multi-level optimization strategy - see docs/optimizations.md#multi-level-architecture
func checkModifiedFiles(workdirPath, workdirName, currentBranchName string) (bool, error) {
	fmt.Printf("Debug: checkModifiedFiles called for workdir %s\n", workdirPath)

	// Timestamp-based early exit optimization - see docs/optimizations.md#timestamp-check
	startTimestamp := time.Now()
	hasRecentChanges, err := hasFilesNewerThanLastWmemCommit(workdirPath, workdirName, currentBranchName)
	if err == nil && !hasRecentChanges {
		fmt.Printf("Debug: No files newer than last wmem commit - ultra-fast early exit for %s (took %v)\n", workdirPath, time.Since(startTimestamp))
		return false, nil // Early exit: No files modified since last commit
	}
	if err != nil {
		fmt.Printf("Debug: Timestamp check failed, falling back to git status check: %v\n", err)
	}
	fmt.Printf("Debug: Timestamp check took %v for %s\n", time.Since(startTimestamp), workdirPath)

	// Quick check for working directory changes
	hasCurrentChanges, err := hasWorkingDirectoryChanges(workdirPath)
	if err != nil {
		return false, fmt.Errorf("failed to check working directory changes: %w", err)
	}

	fmt.Printf("Debug: hasWorkingDirectoryChanges=%v for %s\n", hasCurrentChanges, workdirPath)

	// Early exit if no working directory changes and no new commits
	if !hasCurrentChanges {
		fmt.Printf("Debug: No working dir changes detected for %s\n", workdirPath)

		// Additional check: verify HEAD hasn't moved since last wmem commit
		headUnchanged, err := isHeadUnchangedSinceLastWmemCommit(workdirPath, workdirName, currentBranchName)
		if err != nil {
			fmt.Printf("Debug: Failed to check HEAD status, proceeding with full check: %v\n", err)
			// Fall through to full check on error
		} else if headUnchanged {
			fmt.Printf("Debug: HEAD unchanged and no working dir changes - early exit for %s\n", workdirPath)
			return false, nil // EARLY EXIT: Nothing changed since last commit
		} else {
			fmt.Printf("Debug: HEAD moved but no working dir changes - need to check for new commits in %s\n", workdirPath)
		}
	} else {
		fmt.Printf("Debug: Has working dir changes, proceeding with full check for %s\n", workdirPath)
	}

	// Fall back to full tree comparison if early exit conditions not met
	absWorkdirPath, err := filepath.Abs(workdirPath)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute workdir path: %w", err)
	}

	// Open wmem-repo's bare repository
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to open bare repository: %w", err)
	}

	// Get wmem-br/<current-branch-name> branch
	wmemBranchName := fmt.Sprintf("wmem-br/%s", currentBranchName)
	wmemBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName))

	wmemBranchHashRef, err := bareRepo.Reference(wmemBranchRef, true)
	if err != nil {
		return false, fmt.Errorf("failed to get wmem branch reference: %w", err)
	}

	// Get the tree from wmem-br/<current-branch-name> commit
	wmemCommit, err := bareRepo.CommitObject(wmemBranchHashRef.Hash())
	if err != nil {
		return false, fmt.Errorf("failed to get wmem commit: %w", err)
	}

	// Create tree from current filesystem state
	// Find the last merge commit to identify touched files
	workdirRepo, err := git.PlainOpen(workdirPath)
	if err != nil {
		return false, fmt.Errorf("failed to open workdir repository: %w", err)
	}

	headRef, err := workdirRepo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD: %w", err)
	}

	lastMergeHash, err := findLastMergeCommit(workdirRepo, headRef.Hash())
	if err != nil {
		// If no merge commit found, use full tree creation
		currentTreeHash, err := createTreeFromFilesystem(bareRepo, absWorkdirPath)
		if err != nil {
			return false, fmt.Errorf("failed to create tree from filesystem: %w", err)
		}
		return currentTreeHash != wmemCommit.TreeHash, nil
	}

	// Get HEAD SHA1 for caching
	headSHA1 := headRef.Hash().String()
	lastMergeSHA1 := lastMergeHash.String()

	fmt.Printf("Debug: Getting touched files for %s (HEAD: %s, LastMerge: %s)\n", workdirPath, headSHA1[:8], lastMergeSHA1[:8])
	startTouched := time.Now()

	// Try to get touched files from cache first
	touchedFiles, cacheHit := globalCommitCache.getTouchedFilesCached(workdirPath, headSHA1, lastMergeSHA1)
	if cacheHit {
		fmt.Printf("Debug: CACHE HIT for touched files - %d files (took %v) for %s\n", len(touchedFiles), time.Since(startTouched), workdirPath)
	} else {
		// Cache miss - compute touched files and cache the result
		fmt.Printf("Debug: CACHE MISS for touched files - computing...\n")
		touchedFiles, err = getTouchedFilesSinceMerge(workdirPath, lastMergeHash)
		if err != nil {
			return false, fmt.Errorf("failed to get touched files: %w", err)
		}

		// Cache the result for future calls
		globalCommitCache.cacheTouchedFiles(workdirPath, headSHA1, lastMergeSHA1, touchedFiles)
		fmt.Printf("Debug: CACHED touched files result - %d files (took %v) for %s\n", len(touchedFiles), time.Since(startTouched), workdirPath)
	}

	// If no files are touched, we can skip the expensive tree creation
	if len(touchedFiles) == 0 {
		return false, nil
	}

	// Only create tree from touched files with caching
	// Implementation: docs/optimizations.md#touched-files-optimization
	fmt.Printf("Debug: Processing %d touched files for %s\n", len(touchedFiles), workdirPath)
	startTree := time.Now()

	// Try to get tree hash from cache first
	currentTreeHash, treeCacheHit := globalCommitCache.getTreeHashCached(workdirPath, headSHA1, touchedFiles)
	if treeCacheHit {
		fmt.Printf("Debug: CACHE HIT for tree hash (took %v) for %s\n", time.Since(startTree), workdirPath)
	} else {
		// Cache miss - compute tree hash and cache the result
		fmt.Printf("Debug: CACHE MISS for tree hash - computing...\n")
		currentTreeHash, err = createTreeFromTouchedFiles(bareRepo, absWorkdirPath, touchedFiles, wmemCommit.TreeHash)
		if err != nil {
			return false, fmt.Errorf("failed to create tree from touched files: %w", err)
		}

		// Cache the result for future calls
		globalCommitCache.cacheTreeHash(workdirPath, headSHA1, touchedFiles, currentTreeHash)
		fmt.Printf("Debug: CACHED tree hash result (took %v) for %s\n", time.Since(startTree), workdirPath)
	}

	// Compare tree hashes - if they're different, there are modifications
	return currentTreeHash != wmemCommit.TreeHash, nil
}

// isBrokenSymlink detects broken symbolic links
func isBrokenSymlink(path string) bool {
	lstat, err := os.Lstat(path)
	if err != nil || lstat.Mode()&os.ModeSymlink == 0 {
		return false // Not a symlink or doesn't exist
	}

	// Check if the symlink target exists
	_, err = os.Stat(path)
	return os.IsNotExist(err) // True if target doesn't exist (broken symlink)
}

// addFilesAndCommit implements steps 7-8 of UC: sync-workdir
func addFilesAndCommit(workdirPath, workdirName, currentBranchName string, commitInfo *CommitInfo) (plumbing.Hash, error) {
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to open bare repository: %w", err)
	}

	wmemBranchName := fmt.Sprintf("wmem-br/%s", currentBranchName)
	wmemBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName))

	wmemBranchHashRef, err := bareRepo.Reference(wmemBranchRef, true)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get wmem branch reference: %w", err)
	}

	// Parse commit signatures
	authorSig, committerSig, err := parseCommitSignatures(commitInfo)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to parse commit signatures: %w", err)
	}

	// Create regular commit with all changes from workdir
	newCommitHash, err := createRegularCommit(bareRepo, wmemBranchHashRef.Hash(), commitInfo, authorSig, committerSig, workdirPath)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to create regular commit: %w", err)
	}

	// Update wmem-br/<current-branch-name> to point to new commit
	newWmemBranchRef := plumbing.NewHashReference(wmemBranchRef, newCommitHash)
	err = bareRepo.Storer.SetReference(newWmemBranchRef)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to update wmem branch: %w", err)
	}

	return newCommitHash, nil
}

// updateWmemHeadBranch implements step 9 of UC: sync-workdir
func updateWmemHeadBranch(workdirName string, newCommitHash plumbing.Hash) error {
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open bare repository: %w", err)
	}

	wmemHeadRef := plumbing.ReferenceName("refs/heads/wmem-br/head")
	newWmemHeadRef := plumbing.NewHashReference(wmemHeadRef, newCommitHash)
	err = bareRepo.Storer.SetReference(newWmemHeadRef)
	if err != nil {
		return fmt.Errorf("failed to update wmem-br/head: %w", err)
	}

	return nil
}

// isCommitMerged checks if a commit is already merged into a target branch
func isCommitMerged(repo *git.Repository, commitHash, targetHash plumbing.Hash) (bool, error) {
	// If the commit hashes are the same, it's already merged
	if commitHash == targetHash {
		return true, nil
	}

	// Get commit objects
	targetCommit, err := repo.CommitObject(targetHash)
	if err != nil {
		return false, fmt.Errorf("failed to get target commit: %w", err)
	}

	// Check if commitHash is an ancestor of targetHash
	commitToCheck, err := repo.CommitObject(commitHash)
	if err != nil {
		return false, fmt.Errorf("failed to get commit to check: %w", err)
	}

	// Check if commitToCheck is an ancestor of targetCommit (i.e., commitHash is reachable from targetHash)
	isAncestor, err := commitToCheck.IsAncestor(targetCommit)
	if err != nil {
		return false, fmt.Errorf("failed to check ancestry: %w", err)
	}

	return isAncestor, nil
}

// parseCommitSignatures parses author and committer signatures from commit info
func parseCommitSignatures(commitInfo *CommitInfo) (*object.Signature, *object.Signature, error) {
	authorSig, err := parseSignature(commitInfo.Author)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse author: %w", err)
	}

	committerSig, err := parseSignature(commitInfo.Committer)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse committer: %w", err)
	}

	return authorSig, committerSig, nil
}

// createRegularCommit creates a regular commit when HEAD is already merged and there are uncommitted changes
// This implements steps 7-8 of UC: sync-workdir with READ-ONLY access to workdir
// Uses optimized tree creation from current repository state
func createRegularCommit(repo *git.Repository, wmemBranchHash plumbing.Hash, commitInfo *CommitInfo, author, committer *object.Signature, workdirPath string) (plumbing.Hash, error) {
	// Build tree directly from current state (READ-ONLY approach)
	rootTreeHash, err := createTreeFromCurrentState(workdirPath, repo)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to create tree from current state: %w", err)
	}

	// Step 8: Create new commit to wmem-br/<current-branch-name> branch based on commit-info
	commit := &object.Commit{
		Message:      commitInfo.Message,
		TreeHash:     rootTreeHash,                    // Tree built from filesystem
		ParentHashes: []plumbing.Hash{wmemBranchHash}, // wmem-br branch as parent
		Author:       *author,
		Committer:    *committer,
	}

	// Encode and store the commit in bare repository
	obj := repo.Storer.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to encode commit: %w", err)
	}

	commitHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to store commit: %w", err)
	}

	return commitHash, nil
}

// createWmemMergeCommit creates a merge commit following ALG: wmem merge
// Reference: docs/use-cases/git-wmem-commit/basic.md#alg-wmem-merge
// This implements Alternative 5b when the workdir HEAD is not yet merged into wmem-br branch
func createWmemMergeCommit(repo *git.Repository, wmemBranchHash, workdirCommitHash plumbing.Hash, currentBranchName string, commitInfo *CommitInfo, author, committer *object.Signature) (plumbing.Hash, error) {
	// Get the tree SHA-1 from workdir HEAD commit (accepting workdir's branch tree hash)
	// This implements the "accepting workdir's branch tree hash" part of ALG: wmem merge
	workdirCommit, err := repo.CommitObject(workdirCommitHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get workdir commit: %w", err)
	}

	// Create merge commit message that explains the merge strategy
	mergeMessage := fmt.Sprintf("Merge workdir '%s' into 'wmem-br/%s' accepting workdir's branch tree hash\n\n%s",
		currentBranchName, currentBranchName, commitInfo.Message)

	// Create merge commit object with workdir's tree and both parents
	// Parent order: wmem-br parent first (main line), then workdir parent (merged branch)
	// Tree: workdir's tree (no conflicts - we accept workdir's version)
	mergeCommit := &object.Commit{
		Message:      mergeMessage,
		TreeHash:     workdirCommit.TreeHash,                             // Accept workdir's tree (no conflicts)
		ParentHashes: []plumbing.Hash{wmemBranchHash, workdirCommitHash}, // wmem-br parent first, then workdir parent
		Author:       *author,
		Committer:    *committer,
	}

	// Encode and store the merge commit
	obj := repo.Storer.NewEncodedObject()
	if err := mergeCommit.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to encode merge commit: %w", err)
	}

	commitHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to store merge commit: %w", err)
	}

	return commitHash, nil
}

// createWmemCommit creates the wmem repository commit
func createWmemCommit(commitInfo *CommitInfo, workdirResults []WorkdirCommitResult) error {
	// Generate wmem-repo commit message according to spec
	wmemCommitMessage := generateWmemRepoCommitMessage(commitInfo, workdirResults)

	// Open wmem repository
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open wmem repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all files (metadata might have changed)
	// Explicitly add metadata directories to ensure they're tracked
	metadataPaths := []string{
		"md/",
		"md-internal/",
		".",
	}

	for _, path := range metadataPaths {
		if _, err := os.Stat(path); err == nil {
			_, err = worktree.Add(path)
			if err != nil {
				return fmt.Errorf("failed to add metadata path %s: %w", path, err)
			}
		}
	}

	// Add all remaining files to catch any other changes
	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	// Debug: Check what files are staged for commit
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	var stagedFiles []string
	for filePath, fileStatus := range status {
		if fileStatus.Staging != ' ' { // ' ' means unmodified in staging area
			stagedFiles = append(stagedFiles, filePath)
		}
	}

	if len(stagedFiles) > 0 {
		fmt.Printf("Debug: Staging %d files for wmem-repo commit: %v\n", len(stagedFiles), stagedFiles)
	} else {
		fmt.Printf("Debug: No files staged for wmem-repo commit\n")
	}

	// Parse author and committer
	authorSig, err := parseSignature(commitInfo.Author)
	if err != nil {
		return fmt.Errorf("failed to parse author: %w", err)
	}

	committerSig, err := parseSignature(commitInfo.Committer)
	if err != nil {
		return fmt.Errorf("failed to parse committer: %w", err)
	}

	// Create commit (allow empty commits for wmem operations)
	_, err = worktree.Commit(wmemCommitMessage, &git.CommitOptions{
		Author:            authorSig,
		Committer:         committerSig,
		AllowEmptyCommits: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create wmem commit: %w", err)
	}

	return nil
}

// generateWmemRepoCommitMessage creates the wmem-repo commit message according to spec
// Reference: docs/data-structures.md#commit-msg
func generateWmemRepoCommitMessage(commitInfo *CommitInfo, workdirResults []WorkdirCommitResult) string {
	// Start with msg-prefix and wmem-uid (from original commitInfo.Message)
	message := commitInfo.Message

	// Add wmem-repo specific msg-body
	message += "\n\nMeta wmem-commit of workdir commits"
	hasAnyWorkdirChanges := false
	for _, result := range workdirResults {
		if result.HasChanges {
			// Truncate commit hash to 12 characters for readability
			shortHash := result.CommitHash
			if len(shortHash) > 12 {
				shortHash = shortHash[:12]
			}
			message += fmt.Sprintf("\n- `%s` `%s` `%s`", result.WorkdirName, result.BranchName, shortHash)
			hasAnyWorkdirChanges = true
		}
		// Skip workdirs with no changes - they won't appear in the commit message
	}

	// If no workdirs had changes, indicate this was a metadata-only commit
	if !hasAnyWorkdirChanges {
		message += "\n(No workdir changes - metadata only)"
	}

	return message
}

// countChangedWorkdirs counts how many workdirs had changes
func countChangedWorkdirs(results []WorkdirCommitResult) int {
	count := 0
	for _, result := range results {
		if result.HasChanges {
			count++
		}
	}
	return count
}

// hasWmemRepoMetadataChanges checks if there are uncommitted changes in wmem-repo metadata
func hasWmemRepoMetadataChanges() (bool, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return false, fmt.Errorf("failed to open wmem repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get wmem-repo status: %w", err)
	}

	// Check if there are any changes (staged or unstaged)
	return !status.IsClean(), nil
}

// parseSignature parses a git signature string
func parseSignature(sigStr string) (*object.Signature, error) {
	// Expected format: "Name <email>"
	parts := strings.Split(sigStr, " <")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid signature format: %s", sigStr)
	}

	name := parts[0]
	email := strings.TrimSuffix(parts[1], ">")

	return &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}, nil
}

// copyTreeObjects recursively copies a tree and all its referenced objects (subtrees and blobs)
// from the source repository to the destination repository
func copyTreeObjects(srcRepo, dstRepo *git.Repository, treeHash plumbing.Hash) error {
	// Get the tree object from source repository
	srcTree, err := srcRepo.TreeObject(treeHash)
	if err != nil {
		return fmt.Errorf("failed to get tree object from source: %w", err)
	}

	// Check if tree already exists in destination repository
	_, err = dstRepo.TreeObject(treeHash)
	if err == nil {
		// Tree already exists, no need to copy
		return nil
	}

	// Copy all referenced objects first (depth-first)
	for _, entry := range srcTree.Entries {
		switch entry.Mode {
		case filemode.Dir:
			// Recursively copy subtree
			err = copyTreeObjects(srcRepo, dstRepo, entry.Hash)
			if err != nil {
				return fmt.Errorf("failed to copy subtree %s: %w", entry.Hash, err)
			}
		case filemode.Regular, filemode.Executable:
			// Copy blob object
			err = copyBlobObject(srcRepo, dstRepo, entry.Hash)
			if err != nil {
				return fmt.Errorf("failed to copy blob %s: %w", entry.Hash, err)
			}
		}
	}

	// Now copy the tree object itself
	return copyObject(srcRepo, dstRepo, treeHash)
}

// copyBlobObject copies a blob object from source to destination repository
func copyBlobObject(srcRepo, dstRepo *git.Repository, blobHash plumbing.Hash) error {
	// Check if blob already exists in destination repository
	_, err := dstRepo.BlobObject(blobHash)
	if err == nil {
		// Blob already exists, no need to copy
		return nil
	}

	// Copy the blob object
	return copyObject(srcRepo, dstRepo, blobHash)
}

// copyObject copies any git object from source to destination repository
func copyObject(srcRepo, dstRepo *git.Repository, objectHash plumbing.Hash) error {
	// Get the encoded object from source repository
	srcObj, err := srcRepo.Storer.EncodedObject(plumbing.AnyObject, objectHash)
	if err != nil {
		return fmt.Errorf("failed to get object from source: %w", err)
	}

	// Check if object already exists in destination
	_, err = dstRepo.Storer.EncodedObject(plumbing.AnyObject, objectHash)
	if err == nil {
		// Object already exists, no need to copy
		return nil
	}

	// Copy the object to destination repository
	_, err = dstRepo.Storer.SetEncodedObject(srcObj)
	if err != nil {
		return fmt.Errorf("failed to store object in destination: %w", err)
	}

	return nil
}

// createTreeFromTouchedFiles creates a git tree from only the specified touched files
// Only processes files that have actually changed for better performance
// Implementation: docs/optimizations.md#touched-files-optimization
func createTreeFromTouchedFiles(repo *git.Repository, dirPath string, touchedFiles []string, baseTreeHash plumbing.Hash) (plumbing.Hash, error) {
	// Get base tree to start with
	baseTree, err := repo.TreeObject(baseTreeHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get base tree: %w", err)
	}

	// Create a map of base tree entries with full path support
	baseEntries := make(map[string]object.TreeEntry)
	err = baseTree.Files().ForEach(func(file *object.File) error {
		baseEntries[file.Name] = object.TreeEntry{
			Name: filepath.Base(file.Name),
			Mode: file.Mode,
			Hash: file.Hash,
		}
		return nil
	})
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to enumerate base tree files: %w", err)
	}

	// Track which files we need to update
	updatedFiles := make(map[string]bool)
	for _, filename := range touchedFiles {
		updatedFiles[filename] = true
	}

	// Update entries for touched files
	for _, filename := range touchedFiles {
		filePath := filepath.Join(dirPath, filename)

		// Check if file exists in filesystem
		fileInfo, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			// File was deleted, remove from entries
			delete(baseEntries, filename)
			continue
		} else if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("failed to stat file %s: %w", filePath, err)
		}

		// Handle directories (should not happen in touched files, but defensive programming)
		if fileInfo.IsDir() {
			continue
		}

		// Check if this is a symbolic link
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			// Handle symlinks by reading the target
			target, err := os.Readlink(filePath)
			if err != nil {
				return plumbing.ZeroHash, fmt.Errorf("failed to read symlink %s: %w", filePath, err)
			}

			// Create blob from symlink target
			blob := repo.Storer.NewEncodedObject()
			blob.SetType(plumbing.BlobObject)
			writer, err := blob.Writer()
			if err != nil {
				return plumbing.ZeroHash, fmt.Errorf("failed to create blob writer for symlink: %w", err)
			}

			_, err = writer.Write([]byte(target))
			if err != nil {
				writer.Close()
				return plumbing.ZeroHash, fmt.Errorf("failed to write symlink content: %w", err)
			}
			writer.Close()

			blobHash, err := repo.Storer.SetEncodedObject(blob)
			if err != nil {
				return plumbing.ZeroHash, fmt.Errorf("failed to store symlink blob: %w", err)
			}

			baseEntries[filename] = object.TreeEntry{
				Name: filepath.Base(filename),
				Mode: filemode.Symlink,
				Hash: blobHash,
			}
			continue
		}

		// Read regular file content and create blob
		content, err := os.ReadFile(filePath)
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		blob := repo.Storer.NewEncodedObject()
		blob.SetType(plumbing.BlobObject)
		writer, err := blob.Writer()
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("failed to create blob writer: %w", err)
		}

		_, err = writer.Write(content)
		if err != nil {
			writer.Close()
			return plumbing.ZeroHash, fmt.Errorf("failed to write blob content: %w", err)
		}
		writer.Close()

		blobHash, err := repo.Storer.SetEncodedObject(blob)
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("failed to store blob: %w", err)
		}

		// Determine file mode
		mode := filemode.Regular
		if fileInfo.Mode()&0111 != 0 {
			mode = filemode.Executable
		}

		baseEntries[filename] = object.TreeEntry{
			Name: filepath.Base(filename),
			Mode: mode,
			Hash: blobHash,
		}
	}

	// Create tree from updated entries
	var treeEntries []object.TreeEntry
	for _, entry := range baseEntries {
		treeEntries = append(treeEntries, entry)
	}

	// Sort entries by name (required for git tree objects)
	sort.Slice(treeEntries, func(i, j int) bool {
		return treeEntries[i].Name < treeEntries[j].Name
	})

	tree := &object.Tree{Entries: treeEntries}
	treeObj := repo.Storer.NewEncodedObject()
	err = tree.Encode(treeObj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to encode tree: %w", err)
	}

	return repo.Storer.SetEncodedObject(treeObj)
}

// createTreeFromFilesystem creates a git tree object from the filesystem directory structure
// This is a READ-ONLY approach that doesn't modify the working directory or its repo
func createTreeFromFilesystem(repo *git.Repository, dirPath string) (plumbing.Hash, error) {
	// Read directory entries
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	var treeEntries []object.TreeEntry

	// Process each entry in the directory
	for _, entry := range entries {
		// Skip .git directory specifically (like git add -A does), but include other dotfiles
		if entry.Name() == ".git" {
			continue
		}

		entryPath := filepath.Join(dirPath, entry.Name())

		// Check if this path is ignored by gitignore rules (like git add -A does)
		// We need to check relative to the workdir root, not the current subdirectory
		isIgnored, err := isPathIgnored(dirPath, entry.Name())
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("failed to check gitignore for %s: %w", entryPath, err)
		}
		if isIgnored {
			// Skip ignored files/directories entirely (like git add -A does)
			continue
		}

		if entry.IsDir() {
			// Check if this subdirectory contains a .git directory (indicates it's a git repository)
			// Reference: docs/use-cases/git-wmem-commit/basic.md step 7 detail
			gitPath := filepath.Join(entryPath, ".git")
			if _, err := os.Stat(gitPath); err == nil {
				// Handle nested git repository as gitlink (like git add -A does)
				// Get the HEAD commit hash from the nested repository
				nestedRepo, err := git.PlainOpen(entryPath)
				if err != nil {
					return plumbing.ZeroHash, fmt.Errorf("failed to open nested git repository %s: %w", entryPath, err)
				}

				head, err := nestedRepo.Head()
				if err != nil {
					return plumbing.ZeroHash, fmt.Errorf("failed to get HEAD of nested git repository %s: %w", entryPath, err)
				}

				// Add gitlink entry to tree (mode 160000 like git add -A does)
				treeEntries = append(treeEntries, object.TreeEntry{
					Name: entry.Name(),
					Mode: filemode.Submodule, // 160000 - gitlink mode
					Hash: head.Hash(),
				})
				continue
			}

			// Recursively create subtree for regular directories
			subTreeHash, err := createTreeFromFilesystem(repo, entryPath)
			if err != nil {
				return plumbing.ZeroHash, fmt.Errorf("failed to create subtree for %s: %w", entryPath, err)
			}

			// Add directory entry to tree
			treeEntries = append(treeEntries, object.TreeEntry{
				Name: entry.Name(),
				Mode: filemode.Dir,
				Hash: subTreeHash,
			})
		} else {
			// Check for broken symlinks before creating blob
			if isBrokenSymlink(entryPath) {
				// Skip broken symlinks (like git add -A does)
				continue
			}

			// Create blob for file
			blobHash, err := createBlobFromFile(repo, entryPath)
			if err != nil {
				return plumbing.ZeroHash, fmt.Errorf("failed to create blob for %s: %w", entryPath, err)
			}

			// Determine file mode
			info, err := entry.Info()
			if err != nil {
				return plumbing.ZeroHash, fmt.Errorf("failed to get file info for %s: %w", entryPath, err)
			}

			mode := filemode.Regular
			if info.Mode()&0111 != 0 {
				mode = filemode.Executable
			}

			// Add file entry to tree
			treeEntries = append(treeEntries, object.TreeEntry{
				Name: entry.Name(),
				Mode: mode,
				Hash: blobHash,
			})
		}
	}

	// Sort entries by name using go-git's native sorting (ensures Git compatibility)
	sort.Sort(object.TreeEntrySorter(treeEntries))

	// Create the tree object
	tree := &object.Tree{Entries: treeEntries}

	// Encode and store the tree object
	obj := repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.TreeObject)

	if err := tree.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to encode tree object: %w", err)
	}

	treeHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to store tree object: %w", err)
	}

	return treeHash, nil
}

// createBlobFromFile creates a git blob object from a file
func createBlobFromFile(repo *git.Repository, filePath string) (plumbing.Hash, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Create blob with the file content
	blob := &object.Blob{}
	blob.Size = int64(len(content))

	// Create encoded object for the blob
	obj := repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(content)))

	// Write content to the blob object
	writer, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get blob writer: %w", err)
	}

	_, err = writer.Write(content)
	if err != nil {
		writer.Close()
		return plumbing.ZeroHash, fmt.Errorf("failed to write blob content: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to close blob writer: %w", err)
	}

	// Store the blob object
	blobHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to store blob object: %w", err)
	}

	return blobHash, nil
}

// isPathIgnored checks if a file/directory path should be ignored according to .gitignore rules
// This mimics git add -A behavior by respecting gitignore patterns
func isPathIgnored(dirPath, entryName string) (bool, error) {
	// Find the root of the git repository to locate .gitignore files
	gitRoot, err := findGitRoot(dirPath)
	if err != nil {
		// If we can't find git root, don't ignore anything
		return false, nil
	}

	// Get relative path from git root
	relPath, err := filepath.Rel(gitRoot, filepath.Join(dirPath, entryName))
	if err != nil {
		return false, fmt.Errorf("failed to get relative path: %w", err)
	}

	// Check for .gitignore files from git root down to current directory
	return checkGitignorePatterns(gitRoot, relPath)
}

// findGitRoot finds the root directory of the git repository
func findGitRoot(startPath string) (string, error) {
	currentPath := startPath
	for {
		gitPath := filepath.Join(currentPath, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return currentPath, nil
		}

		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			// Reached filesystem root
			return "", fmt.Errorf("not in a git repository")
		}
		currentPath = parentPath
	}
}

// checkGitignorePatterns checks if a path matches any gitignore patterns
func checkGitignorePatterns(gitRoot, relPath string) (bool, error) {
	// For now, implement basic gitignore checking
	// This could be enhanced with a full gitignore library later

	gitignorePath := filepath.Join(gitRoot, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No .gitignore file, nothing is ignored
			return false, nil
		}
		return false, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Basic pattern matching - this is simplified
		// Remove trailing slash for directory patterns
		pattern := strings.TrimSuffix(line, "/")

		// Simple exact match or prefix match for directories
		if pattern == relPath || (strings.HasSuffix(line, "/") && strings.HasPrefix(relPath, pattern+"/")) {
			return true, nil
		}
	}

	return false, nil
}
