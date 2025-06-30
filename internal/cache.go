package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

// Simple file-based cache for directory mtimes
func readLastMtimeFromFile(cacheFile string) (time.Time, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return time.Time{}, err
	}

	var mtime time.Time
	err = json.Unmarshal(data, &mtime)
	return mtime, err
}

func writeLastMtimeToFile(cacheFile string, mtime time.Time) error {
	data, err := json.Marshal(mtime)
	if err != nil {
		return err
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// findWmemRepoRoot finds the wmem repository root by looking for .git-wmem file
func findWmemRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		wmemFile := filepath.Join(dir, ".git-wmem")
		if _, err := os.Stat(wmemFile); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a wmem repository (no .git-wmem file found)")
		}
		dir = parent
	}
}

// getCacheFilePath returns the cache file path within the wmem repo cache directory
func getCacheFilePath(workdirPath string) (string, error) {
	wmemRoot, err := findWmemRepoRoot()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(wmemRoot, "cache")
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("git-wmem-cache-%s.json", filepath.Base(workdirPath)))
	return cacheFile, nil
}

// getTouchedFilesCached gets touched files with SHA1-based caching
// Cache implementation: docs/optimizations.md#touched-files-cache
func (cc *CommitCache) getTouchedFilesCached(workdirPath string, headSHA1 string, lastMergeSHA1 string) ([]string, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	cacheKey := workdirPath
	entry, exists := cc.touchedFilesCache[cacheKey]

	if !exists {
		return nil, false
	}

	// Check if cache entry is valid (same HEAD and merge commit)
	if entry.headSHA1 == headSHA1 && entry.lastMergeSHA1 == lastMergeSHA1 {
		// Cache hit! Return cached touched files
		return entry.touchedFiles, true
	}

	return nil, false
}

// cacheTouchedFiles stores touched files result in cache
func (cc *CommitCache) cacheTouchedFiles(workdirPath string, headSHA1 string, lastMergeSHA1 string, touchedFiles []string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cacheKey := workdirPath
	cc.touchedFilesCache[cacheKey] = touchedFilesCacheEntry{
		headSHA1:      headSHA1,
		lastMergeSHA1: lastMergeSHA1,
		touchedFiles:  touchedFiles,
		cacheTime:     time.Now(),
	}
}

// getTreeHashCached gets tree hash with SHA1-based caching
// Cache implementation: docs/optimizations.md#tree-hash-cache
func (cc *CommitCache) getTreeHashCached(workdirPath string, headSHA1 string, touchedFiles []string) (plumbing.Hash, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	cacheKey := workdirPath
	entry, exists := cc.treeHashCache[cacheKey]

	if !exists {
		return plumbing.ZeroHash, false
	}

	// Check if cache entry is valid (same HEAD and same touched files)
	if entry.headSHA1 == headSHA1 && slicesEqual(entry.touchedFiles, touchedFiles) {
		return entry.treeHash, true
	}

	return plumbing.ZeroHash, false
}

// cacheTreeHash stores tree hash result in cache
func (cc *CommitCache) cacheTreeHash(workdirPath string, headSHA1 string, touchedFiles []string, treeHash plumbing.Hash) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cacheKey := workdirPath
	cc.treeHashCache[cacheKey] = treeHashCacheEntry{
		headSHA1:     headSHA1,
		touchedFiles: make([]string, len(touchedFiles)),
		treeHash:     treeHash,
		cacheTime:    time.Now(),
	}
	copy(cc.treeHashCache[cacheKey].touchedFiles, touchedFiles)
}

// clearCache clears all cache entries (useful for testing or memory management)
func (cc *CommitCache) clearCache() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.touchedFilesCache = make(map[string]touchedFilesCacheEntry)
	cc.treeHashCache = make(map[string]treeHashCacheEntry)
	cc.directoryStateCache = make(map[string]directoryStateCacheEntry)
	cc.fileListCache = make(map[string]fileListCacheEntry)
}

// getCacheStats returns cache statistics for debugging
func (cc *CommitCache) getCacheStats() (touchedFilesEntries, treeHashEntries, dirStateEntries, fileListEntries, wmemTreeEntries int) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return len(cc.touchedFilesCache), len(cc.treeHashCache), len(cc.directoryStateCache), len(cc.fileListCache), len(cc.wmemTreeCache)
}

// printCacheStats prints cache statistics
// Statistics documentation: docs/optimizations.md#cache-statistics
func printCacheStats() {
	touchedCount, treeCount, dirStateCount, fileListCount, wmemTreeCount := globalCommitCache.getCacheStats()
	if touchedCount > 0 || treeCount > 0 || dirStateCount > 0 || fileListCount > 0 || wmemTreeCount > 0 {
		fmt.Printf("Debug: Cache stats - TouchedFiles: %d, TreeHash: %d, DirState: %d, FileList: %d, WmemTree: %d entries\n",
			touchedCount, treeCount, dirStateCount, fileListCount, wmemTreeCount)
	}
}

// hasFilesDeletedUsingDirectoryMtime uses directory modification times to efficiently detect file deletions
// Based on Git's untracked cache optimization strategy
// Implementation: docs/optimizations.md#directory-deletion-detection
func hasFilesDeletedUsingDirectoryMtime(workdirPath, workdirName, currentBranchName string) (bool, error) {
	startTotal := time.Now()

	// Get current directory state
	startDirStat := time.Now()
	dirStat, err := os.Stat(workdirPath)
	if err != nil {
		return false, err
	}
	currentDirMtime := dirStat.ModTime()
	fmt.Printf("Debug: os.Stat took %v for %s\n", time.Since(startDirStat), workdirPath)

	// Simple file-based cache check
	cacheFile, err := getCacheFilePath(workdirPath)
	if err != nil {
		return false, fmt.Errorf("failed to get cache file path: %v", err)
	}
	if lastMtime, err := readLastMtimeFromFile(cacheFile); err == nil {
		if !currentDirMtime.After(lastMtime) {
			fmt.Printf("Debug: Directory mtime unchanged (file cache) - no deletions detected for %s (total: %v)\n", workdirPath, time.Since(startTotal))
			return false, nil
		}
		fmt.Printf("Debug: Directory mtime changed since file cache for %s (current: %v, cached: %v)\n", workdirPath, currentDirMtime, lastMtime)
	} else {
		fmt.Printf("Debug: No file cache found for %s\n", workdirPath)
	}

	// Get HEAD SHA1 for cache key
	startHeadSHA1 := time.Now()
	headSHA1, err := getCurrentHeadSHA1(workdirPath)
	if err != nil {
		return false, err
	}
	fmt.Printf("Debug: getCurrentHeadSHA1 took %v for %s\n", time.Since(startHeadSHA1), workdirPath)

	cacheKey := fmt.Sprintf("%s:%s", workdirPath, headSHA1)

	globalCommitCache.mu.RLock()
	startCacheLookup := time.Now()
	cachedDirState, hasDirCache := globalCommitCache.directoryStateCache[cacheKey]
	cachedFileList, hasFileCache := globalCommitCache.fileListCache[cacheKey]
	globalCommitCache.mu.RUnlock()
	fmt.Printf("Debug: cache lookup took %v for %s (cacheKey=%s, hasDirCache=%v, hasFileCache=%v)\n", time.Since(startCacheLookup), workdirPath, cacheKey, hasDirCache, hasFileCache)

	// If directory hasn't been modified since last check, no files were deleted
	if hasDirCache && hasFileCache && !currentDirMtime.After(cachedDirState.directoryMtime) {
		fmt.Printf("Debug: Directory mtime unchanged - no deletions detected for %s (total: %v)\n", workdirPath, time.Since(startTotal))
		return false, nil
	}

	// Additional optimization: If directory is very old (> 1 hour), assume no recent deletions
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	if currentDirMtime.Before(oneHourAgo) {
		// Still save to persistent cache for next run
		if err := writeLastMtimeToFile(cacheFile, currentDirMtime); err != nil {
			fmt.Printf("Debug: Failed to save file cache for old directory %s: %v\n", workdirPath, err)
		} else {
			fmt.Printf("Debug: Saved file cache for old directory %s\n", workdirPath)
		}
		fmt.Printf("Debug: Directory very old (%v) - assuming no recent deletions for %s (total: %v)\n", currentDirMtime, workdirPath, time.Since(startTotal))
		return false, nil
	}

	// Directory has been modified, need to check what changed
	fmt.Printf("Debug: Directory mtime changed, checking for deletions in %s (currentMtime=%v, cachedMtime=%v)\n", workdirPath, currentDirMtime, func() interface{} {
		if hasDirCache {
			return cachedDirState.directoryMtime
		}
		return "no cache"
	}())
	startCurrentFiles := time.Now()
	currentFiles, err := getFileListInDirectory(workdirPath)
	if err != nil {
		return false, err
	}
	fmt.Printf("Debug: getFileListInDirectory took %v for %s (%d files)\n", time.Since(startCurrentFiles), workdirPath, len(currentFiles))

	var previousFiles []string
	if hasFileCache && cachedFileList.headSHA1 == headSHA1 {
		// Use cached file list from same HEAD
		fmt.Printf("Debug: Using cached file list (%d files) for %s\n", len(cachedFileList.fileList), workdirPath)
		previousFiles = cachedFileList.fileList
	} else {
		// Need to get file list from wmem tree
		fmt.Printf("Debug: Cache miss - fetching from wmem tree for %s (hasFileCache=%v, headSHA1 currentVScached=%s vs %s)\n", workdirPath, hasFileCache, headSHA1[:8], func() string {
			if hasFileCache {
				return cachedFileList.headSHA1[:8]
			}
			return "no cache"
		}())
		startWmemFiles := time.Now()
		wmemFiles, err := getTrackedFilesFromWmemTree(workdirName, currentBranchName)
		if err != nil {
			return false, err
		}
		fmt.Printf("Debug: getTrackedFilesFromWmemTree took %v for %s (%d files)\n", time.Since(startWmemFiles), workdirPath, len(wmemFiles))
		previousFiles = wmemFiles
	}

	// Check if any previously tracked files are now missing
	startDeletionCheck := time.Now()
	previousFileSet := make(map[string]bool)
	for _, file := range previousFiles {
		previousFileSet[file] = true
	}

	currentFileSet := make(map[string]bool)
	for _, file := range currentFiles {
		currentFileSet[file] = true
	}

	// Detect deletions
	hasDeletedFiles := false
	deletedCount := 0
	for file := range previousFileSet {
		if !currentFileSet[file] {
			fmt.Printf("Debug: Detected deleted file: %s\n", file)
			hasDeletedFiles = true
			deletedCount++
			if deletedCount >= 5 {
				fmt.Printf("Debug: ... (showing first 5 deleted files)\n")
				break // Early exit after showing first few deletions
			}
		}
	}
	fmt.Printf("Debug: deletion check took %v for %s (checked %d vs %d files, found %d deletions)\n", time.Since(startDeletionCheck), workdirPath, len(previousFiles), len(currentFiles), deletedCount)

	// Update caches
	startCacheUpdate := time.Now()
	globalCommitCache.mu.Lock()
	globalCommitCache.directoryStateCache[cacheKey] = directoryStateCacheEntry{
		workdirPath:    workdirPath,
		directoryMtime: currentDirMtime,
		fileCount:      len(currentFiles),
		lastChecked:    time.Now(),
	}
	globalCommitCache.fileListCache[cacheKey] = fileListCacheEntry{
		workdirPath:    workdirPath,
		directoryMtime: currentDirMtime,
		headSHA1:       headSHA1,
		fileList:       currentFiles,
		cacheTime:      time.Now(),
	}
	globalCommitCache.mu.Unlock()
	fmt.Printf("Debug: cache update took %v for %s\n", time.Since(startCacheUpdate), workdirPath)

	// Save current mtime to file cache for next run
	if err := writeLastMtimeToFile(cacheFile, currentDirMtime); err != nil {
		fmt.Printf("Debug: Failed to save file cache for %s: %v\n", workdirPath, err)
	} else {
		fmt.Printf("Debug: Saved file cache for %s\n", workdirPath)
	}

	fmt.Printf("Debug: hasFilesDeletedUsingDirectoryMtime total took %v for %s\n", time.Since(startTotal), workdirPath)
	return hasDeletedFiles, nil
}

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
