package internal

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// LogWmem displays wmem commit history
// Reference: docs/use-cases/git-wmem-log/basic.md
func LogWmem() error {
	// Check if we're in a wmem-repo
	if !isWmemRepo() {
		return fmt.Errorf("not in a wmem repository (missing .git-wmem file). Run this command from a wmem-repo directory.")
	}

	// Open wmem repository
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open wmem repository: %w", err)
	}

	// Get commit iterator
	ref, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	commitIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return fmt.Errorf("failed to get commit log: %w", err)
	}

	// Read workdir map
	workdirMap, err := readWorkdirMap()
	if err != nil {
		return fmt.Errorf("failed to read workdir map: %w", err)
	}

	// Process commits
	err = commitIter.ForEach(func(commit *object.Commit) error {
		return displayCommit(commit, workdirMap)
	})

	if err != nil {
		return fmt.Errorf("failed to process commits: %w", err)
	}

	return nil
}

// displayCommit displays a single commit in the wmem log format
func displayCommit(commit *object.Commit, workdirMap WorkdirMap) error {
	message := commit.Message

	// Extract wmem-uid from commit message
	wmemUID := extractWmemUID(message)
	if wmemUID == "" {
		// Skip non-wmem commits
		return nil
	}

	// Extract the main message (everything before wmem-uid line)
	mainMessage := extractMainMessage(message)

	// Display commit header
	fmt.Printf("%s: %s\n", wmemUID, mainMessage)

	// Display workdir information
	// Show workdir paths with their commit status
	for workdirName, workdirPath := range workdirMap {
		hash, err := getWorkdirCommitHash(workdirName)
		if err == nil && hash != "" {
			fmt.Printf("  %s: %s\n", workdirPath, hash[:12]+"...")
		} else {
			fmt.Printf("  %s: %s\n", workdirPath, "unknown")
		}
	}

	fmt.Println() // Empty line between commits
	return nil
}

// extractWmemUID extracts wmem-uid from commit message
func extractWmemUID(message string) string {
	// Look for wmem-uid: wmem-YYMMDD-HHMMSS-abXY1234 pattern
	re := regexp.MustCompile(`wmem-uid:\s*(wmem-\d{6}-\d{6}-[a-zA-Z0-9]{8})`)
	matches := re.FindStringSubmatch(message)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractMainMessage extracts the main message before wmem-uid line
func extractMainMessage(message string) string {
	lines := strings.Split(message, "\n")
	var mainLines []string

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "wmem-uid:") {
			break
		}
		mainLines = append(mainLines, line)
	}

	result := strings.TrimSpace(strings.Join(mainLines, "\n"))
	if result == "" {
		return "wmem commit"
	}
	return result
}

// getWorkdirCommitHash gets the latest commit hash for a workdir
func getWorkdirCommitHash(workdirName string) (string, error) {
	repoPath := filepath.Join("repos", workdirName+".git")

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", err
	}

	// Try to find the wmem-br/main branch first, then wmem-br/master
	branches := []string{"wmem-br/main", "wmem-br/master"}

	for _, branchName := range branches {
		ref, err := repo.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchName)), true)
		if err == nil {
			return ref.Hash().String(), nil
		}
	}

	// If no wmem-br branches found, try to get any branch
	refs, err := repo.References()
	if err != nil {
		return "", err
	}

	var firstHash string
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() && strings.HasPrefix(ref.Name().Short(), "wmem-br/") {
			if firstHash == "" {
				firstHash = ref.Hash().String()
			}
		}
		return nil
	})

	return firstHash, err
}
