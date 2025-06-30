package internal

import (
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

// CommitCache caches expensive operations based on commit SHA1 and directory state
// Cache architecture: docs/optimizations.md#sha1-caching
type CommitCache struct {
	mu                  sync.RWMutex
	touchedFilesCache   map[string]touchedFilesCacheEntry
	treeHashCache       map[string]treeHashCacheEntry
	directoryStateCache map[string]directoryStateCacheEntry
	fileListCache       map[string]fileListCacheEntry
	wmemTreeCache       map[string]wmemTreeCacheEntry
}

type touchedFilesCacheEntry struct {
	headSHA1      string
	lastMergeSHA1 string
	touchedFiles  []string
	cacheTime     time.Time
}

type treeHashCacheEntry struct {
	headSHA1     string
	touchedFiles []string
	treeHash     plumbing.Hash
	cacheTime    time.Time
}

// directoryStateCacheEntry caches directory modification times for efficient deletion detection
// Cache implementation: docs/optimizations.md#directory-state-cache
type directoryStateCacheEntry struct {
	workdirPath    string
	directoryMtime time.Time
	fileCount      int
	lastChecked    time.Time
}

// fileListCacheEntry caches the list of files that existed at a specific directory state
// Cache implementation: docs/optimizations.md#file-list-cache
type fileListCacheEntry struct {
	workdirPath    string
	directoryMtime time.Time
	headSHA1       string
	fileList       []string
	cacheTime      time.Time
}

type wmemTreeCacheEntry struct {
	workdirName string
	branchName  string
	commitHash  string
	fileList    []string
	cacheTime   time.Time
}

// WorkdirCommitResult contains information about a workdir commit
type WorkdirCommitResult struct {
	WorkdirName string
	BranchName  string
	CommitHash  string
	HasChanges  bool
}

// WorkdirMap represents the mapping of workdir paths to names
type WorkdirMap map[string]string

// CommitInfo represents the structure for wmem commits
type CommitInfo struct {
	WmemUID   string
	Message   string
	Author    string
	Committer string
}

// Global cache instance
var globalCommitCache = &CommitCache{
	touchedFilesCache:   make(map[string]touchedFilesCacheEntry),
	treeHashCache:       make(map[string]treeHashCacheEntry),
	directoryStateCache: make(map[string]directoryStateCacheEntry),
	fileListCache:       make(map[string]fileListCacheEntry),
	wmemTreeCache:       make(map[string]wmemTreeCacheEntry),
}
