package internal

import (
	"encoding/json"
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

// readWorkdirPaths reads the workdir paths from md/commit-workdir-paths
func readWorkdirPaths() ([]string, error) {
	content, err := os.ReadFile("md/commit-workdir-paths")
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var paths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Don't normalize path during reading - validation should check original format
			paths = append(paths, line)
		}
	}

	return paths, nil
}

// validateWorkdirPath validates a workdir path according to the rules
// Reference: docs/validations.md#workdir-path-requirements
func validateWorkdirPath(workdirPath string) error {
	// Check for absolute paths first
	if filepath.IsAbs(workdirPath) {
		return fmt.Errorf("Absolute paths not allowed")
	}

	// Check for wmem-repo itself or subdirectories (. or ./*)
	if workdirPath == "." || strings.HasPrefix(workdirPath, "./") {
		return fmt.Errorf("wmem-repo paths not allowed")
	}

	// Must start with ../
	if !strings.HasPrefix(workdirPath, "../") {
		return fmt.Errorf("Must start with ../")
	}

	// Check for invalid patterns (path traversal after non-.. segments)
	// Split path into segments and ensure .. only appears at the beginning
	segments := strings.Split(workdirPath, "/")
	foundNonDotDot := false
	for _, segment := range segments {
		if segment == ".." {
			if foundNonDotDot {
				return fmt.Errorf("path traversal not allowed")
			}
		} else if segment != "" {
			foundNonDotDot = true
		}
	}

	// Convert to absolute path for checking
	absPath, err := filepath.Abs(workdirPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if directory exists and is readable
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("workdir path not accessible: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("workdir path is not a directory")
	}

	// Check if it's a git repository
	gitDir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return fmt.Errorf("workdir is not a git repository")
	}

	// Check that it's not pointing to wmem-repo itself or its subdirs
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if strings.HasPrefix(absPath, currentDir) {
		return fmt.Errorf("wmem-repo paths not allowed")
	}

	return nil
}

// FindWorkdirName searches for a workdir name by path in the map
func FindWorkdirName(workdirPath string, workdirMap WorkdirMap) (string, bool) {
	// Normalize the input path to handle trailing slashes consistently
	normalizedWorkdirPath := filepath.Clean(workdirPath)

	for name, path := range workdirMap {
		// Normalize the stored path for comparison
		normalizedStoredPath := filepath.Clean(path)
		if normalizedStoredPath == normalizedWorkdirPath {
			return name, true
		}
	}
	return "", false
}

// generateWorkdirName generates a unique workdir name from path
func generateWorkdirName(workdirPath string, existingMap WorkdirMap) string {
	baseName := filepath.Base(workdirPath)

	// Check if base name is already used
	for existingName := range existingMap {
		if existingName == baseName {
			// Find a unique name with suffix
			counter := 2
			for {
				candidate := fmt.Sprintf("%s-%d", baseName, counter)
				if _, exists := existingMap[candidate]; !exists {
					return candidate
				}
				counter++
			}
		}
	}

	return baseName
}

// readWorkdirMap reads the workdir map from md-internal/workdir-map.json
func readWorkdirMap() (WorkdirMap, error) {
	content, err := os.ReadFile("md-internal/workdir-map.json")
	if err != nil {
		return nil, err
	}

	var workdirMap WorkdirMap
	err = json.Unmarshal(content, &workdirMap)
	if err != nil {
		return nil, err
	}

	return workdirMap, nil
}

// saveWorkdirMap saves the workdir map to md-internal/workdir-map.json
func saveWorkdirMap(workdirMap WorkdirMap) error {
	content, err := json.MarshalIndent(workdirMap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("md-internal/workdir-map.json", content, 0644)
}

// getCurrentBranchName implements step 1 of UC: sync-workdir
func getCurrentBranchName(workdirPath string) (string, error) {
	absWorkdirPath, err := filepath.Abs(workdirPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute workdir path: %w", err)
	}

	workdirRepo, err := git.PlainOpen(absWorkdirPath)
	if err != nil {
		return "", fmt.Errorf("failed to open workdir repository (Error case 1z.1): %w", err)
	}

	head, err := workdirRepo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get workdir HEAD: %w", err)
	}

	return head.Name().Short(), nil
}

// getCurrentHeadSHA1 gets the current HEAD SHA1 from a workdir
func getCurrentHeadSHA1(workdirPath string) (string, error) {
	repo, err := git.PlainOpen(workdirPath)
	if err != nil {
		return "", fmt.Errorf("failed to open workdir repository: %w", err)
	}

	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	return headRef.Hash().String(), nil
}

// getFileListInDirectory gets all files in a directory (recursively, excluding .git)
func getFileListInDirectory(dirPath string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if d.Name() == ".git" && d.IsDir() {
			return filepath.SkipDir
		}

		// Only include files, not directories
		if !d.IsDir() {
			// Get relative path from directory root
			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

// getTrackedFilesFromWmemTree gets the list of files tracked in the wmem tree
func getTrackedFilesFromWmemTree(workdirName, currentBranchName string) ([]string, error) {
	startTotal := time.Now()

	// Open repo once
	startRepoOpen := time.Now()
	repoPath := filepath.Join("repos", workdirName+".git")
	bareRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open bare repository: %w", err)
	}
	fmt.Printf("Debug: git.PlainOpen took %v for %s\n", time.Since(startRepoOpen), workdirName)

	startBranchRef := time.Now()
	wmemBranchName := fmt.Sprintf("wmem-br/%s", currentBranchName)
	wmemBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", wmemBranchName))

	wmemBranchHashRef, err := bareRepo.Reference(wmemBranchRef, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get wmem branch reference: %w", err)
	}
	fmt.Printf("Debug: bareRepo.Reference took %v for %s\n", time.Since(startBranchRef), wmemBranchName)

	currentCommitHash := wmemBranchHashRef.Hash().String()

	// Check cache
	cacheKey := fmt.Sprintf("%s:%s", workdirName, currentBranchName)
	globalCommitCache.mu.RLock()
	cachedEntry, hasCached := globalCommitCache.wmemTreeCache[cacheKey]
	globalCommitCache.mu.RUnlock()

	if hasCached && cachedEntry.commitHash == currentCommitHash {
		fmt.Printf("Debug: wmem tree cache HIT for %s (took %v, %d files)\n", workdirName, time.Since(startTotal), len(cachedEntry.fileList))
		return cachedEntry.fileList, nil
	}

	if hasCached {
		fmt.Printf("Debug: wmem tree cache MISS - commit hash changed for %s (was %s, now %s)\n", workdirName, cachedEntry.commitHash[:8], currentCommitHash[:8])
	} else {
		fmt.Printf("Debug: wmem tree cache MISS - no cached entry for %s\n", workdirName)
	}

	startCommitObject := time.Now()
	wmemCommit, err := bareRepo.CommitObject(wmemBranchHashRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get wmem commit: %w", err)
	}
	fmt.Printf("Debug: bareRepo.CommitObject took %v for %s\n", time.Since(startCommitObject), wmemBranchHashRef.Hash().String()[:8])

	startTreeObject := time.Now()
	wmemTree, err := wmemCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get wmem tree: %w", err)
	}
	fmt.Printf("Debug: wmemCommit.Tree took %v for %s\n", time.Since(startTreeObject), workdirName)

	startTreeIteration := time.Now()
	var files []string
	err = wmemTree.Files().ForEach(func(file *object.File) error {
		files = append(files, file.Name)
		return nil
	})
	fmt.Printf("Debug: wmemTree.Files().ForEach took %v for %s (%d files)\n", time.Since(startTreeIteration), workdirName, len(files))
	if err != nil {
		return nil, fmt.Errorf("failed to iterate wmem tree files: %w", err)
	}

	// Cache the result
	startCacheUpdate := time.Now()
	globalCommitCache.mu.Lock()
	globalCommitCache.wmemTreeCache[cacheKey] = wmemTreeCacheEntry{
		workdirName: workdirName,
		branchName:  currentBranchName,
		commitHash:  currentCommitHash,
		fileList:    files,
		cacheTime:   time.Now(),
	}
	globalCommitCache.mu.Unlock()
	fmt.Printf("Debug: wmem tree cache update took %v for %s\n", time.Since(startCacheUpdate), workdirName)

	fmt.Printf("Debug: getTrackedFilesFromWmemTree total took %v for %s\n", time.Since(startTotal), workdirName)
	return files, nil
}
