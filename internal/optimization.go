package internal

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// hasFilesNewerThanLastWmemCommit performs timestamp-based change detection
// Returns true if any files in workdir are newer than the last wmem commit OR if any tracked files are missing
// Implementation: docs/optimizations.md#timestamp-check
func hasFilesNewerThanLastWmemCommit(workdirPath, workdirName, currentBranchName string) (bool, error) {
	startTotal := time.Now()

	// Get timestamp of last wmem commit
	startCommitTime := time.Now()
	lastCommitTime, err := getLastWmemCommitTime(workdirName, currentBranchName)
	if err != nil {
		// If we can't get last commit time, assume changes exist
		return true, err
	}
	fmt.Printf("Debug: getLastWmemCommitTime took %v for %s\n", time.Since(startCommitTime), workdirPath)

	// Quick filesystem scan for files newer than last commit
	startNewerFiles := time.Now()
	hasNewerFiles, err := hasFilesNewerThan(workdirPath, lastCommitTime)
	if err != nil {
		return true, err
	}
	fmt.Printf("Debug: hasFilesNewerThan took %v for %s\n", time.Since(startNewerFiles), workdirPath)

	if hasNewerFiles {
		return true, nil
	}

	// Additional check: Verify that all previously tracked files still exist
	// This catches file deletions that the timestamp check cannot detect
	startDeletion := time.Now()
	hasMissingFiles, err := hasFilesDeletedSinceLastWmemCommit(workdirPath, workdirName, currentBranchName)
	if err != nil {
		// On error, assume changes exist
		return true, err
	}
	fmt.Printf("Debug: hasFilesDeletedSinceLastWmemCommit took %v for %s\n", time.Since(startDeletion), workdirPath)

	fmt.Printf("Debug: Total hasFilesNewerThanLastWmemCommit took %v for %s\n", time.Since(startTotal), workdirPath)

	return hasMissingFiles, nil
}

// getLastWmemCommitTime gets the timestamp of the last wmem commit
func getLastWmemCommitTime(workdirName, currentBranchName string) (time.Time, error) {
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to open bare repository: %w", err)
	}

	wmemBranchName := fmt.Sprintf("wmem-br/%s", currentBranchName)
	wmemBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName))

	wmemBranchHashRef, err := bareRepo.Reference(wmemBranchRef, true)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get wmem branch reference: %w", err)
	}

	wmemCommit, err := bareRepo.CommitObject(wmemBranchHashRef.Hash())
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get wmem commit: %w", err)
	}

	return wmemCommit.Committer.When, nil
}

// hasFilesNewerThan checks if any files in directory are newer than given time
// This is a pure filesystem operation, very fast
func hasFilesNewerThan(dirPath string, since time.Time) (bool, error) {
	found := false

	// Use filepath.WalkDir for efficient traversal
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if d.Name() == ".git" && d.IsDir() {
			return filepath.SkipDir
		}

		// Skip if this is a directory (we only care about file modification times)
		if d.IsDir() {
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			return err
		}

		// Check if file is newer than last commit
		// Add small buffer (1 second) to account for filesystem timestamp precision
		if info.ModTime().After(since.Add(-1 * time.Second)) {
			found = true
			return filepath.SkipAll // Stop walking, we found a newer file
		}

		return nil
	})

	if err != nil {
		return true, err // On error, assume changes exist
	}

	return found, nil
}

// hasFilesDeletedSinceLastWmemCommit checks if any files tracked in the last wmem commit are missing from filesystem
// Uses directory modification times for efficient deletion detection
// Implementation: docs/optimizations.md#directory-deletion-detection
func hasFilesDeletedSinceLastWmemCommit(workdirPath, workdirName, currentBranchName string) (bool, error) {
	// Enhanced deletion detection using directory modification times
	startDirectoryMtime := time.Now()
	hasDeleted, err := hasFilesDeletedUsingDirectoryMtime(workdirPath, workdirName, currentBranchName)
	if err == nil {
		fmt.Printf("Debug: hasFilesDeletedUsingDirectoryMtime took %v for %s\n", time.Since(startDirectoryMtime), workdirPath)
		return hasDeleted, nil
	}
	fmt.Printf("Debug: hasFilesDeletedUsingDirectoryMtime failed (took %v), falling back to tree walk: %v\n", time.Since(startDirectoryMtime), err)

	// Use tree-walking approach if directory optimization fails
	startTreeWalk := time.Now()
	result, treeErr := hasFilesDeletedUsingTreeWalk(workdirPath, workdirName, currentBranchName)
	fmt.Printf("Debug: hasFilesDeletedUsingTreeWalk took %v for %s\n", time.Since(startTreeWalk), workdirPath)
	return result, treeErr
}

// hasFilesDeletedUsingTreeWalk uses tree-walking for deletion detection
func hasFilesDeletedUsingTreeWalk(workdirPath, workdirName, currentBranchName string) (bool, error) {
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		// If bare repo doesn't exist yet, no files to check for deletion
		return false, nil
	}

	wmemBranchName := fmt.Sprintf("wmem-br/%s", currentBranchName)
	wmemBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName))

	wmemBranchHashRef, err := bareRepo.Reference(wmemBranchRef, true)
	if err != nil {
		// If wmem branch doesn't exist yet, no files to check for deletion
		return false, nil
	}

	wmemCommit, err := bareRepo.CommitObject(wmemBranchHashRef.Hash())
	if err != nil {
		return false, fmt.Errorf("failed to get wmem commit: %w", err)
	}

	wmemTree, err := wmemCommit.Tree()
	if err != nil {
		return false, fmt.Errorf("failed to get wmem tree: %w", err)
	}

	// OPTIMIZATION: Instead of checking ALL files, use early termination
	// Check files one by one and return immediately if we find a missing file
	missingFound := false
	filesChecked := 0
	err = wmemTree.Files().ForEach(func(file *object.File) error {
		filesChecked++
		// Progress indicator for large repositories
		if filesChecked%100 == 0 {
			fmt.Printf("Debug: Checked %d files for deletions in %s\n", filesChecked, workdirPath)
		}

		filePath := filepath.Join(workdirPath, file.Name)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("Debug: Found deleted file: %s (after checking %d files)\n", file.Name, filesChecked)
			missingFound = true
			return fmt.Errorf("file deleted") // Use error to break the loop early
		}
		return nil
	})

	fmt.Printf("Debug: Checked %d total files for deletions in %s\n", filesChecked, workdirPath)

	// If we hit the "file deleted" error, that means we found a missing file
	if err != nil && strings.Contains(err.Error(), "file deleted") {
		return true, nil
	}

	// If we got a different error, return it
	if err != nil {
		return false, err
	}

	return missingFound, nil
}
