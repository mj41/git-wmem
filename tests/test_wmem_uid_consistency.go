package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"git-wmem/internal"
)

// Test script to verify that wmem-uid is consistent across all commits in a single git-wmem-commit run
func main() {
	fmt.Println("Testing wmem-uid consistency...")

	// Step 1: Generate commit info (this should generate a single wmem-uid)
	commitInfo, err := internal.TestReadCommitInfo()
	if err != nil {
		log.Fatalf("Failed to read commit info: %v", err)
	}

	fmt.Printf("Generated wmem-uid: %s\n", commitInfo.WmemUID)
	fmt.Printf("Commit message contains: %s\n", commitInfo.Message)

	// Step 2: Verify the wmem-uid format
	wmemUIDPattern := regexp.MustCompile(`wmem-\d{6}-\d{6}-[a-zA-Z0-9]{8}`)
	if !wmemUIDPattern.MatchString(commitInfo.WmemUID) {
		log.Fatalf("Invalid wmem-uid format: %s", commitInfo.WmemUID)
	}

	// Step 3: Verify the commit message contains the wmem-uid
	expectedUIDLine := fmt.Sprintf("wmem-uid: %s", commitInfo.WmemUID)
	if !strings.Contains(commitInfo.Message, expectedUIDLine) {
		log.Fatalf("Commit message does not contain expected wmem-uid line: %s", expectedUIDLine)
	}

	// Step 4: Test that multiple calls to readCommitInfo generate different wmem-uids
	// (to ensure uniqueness between different runs)
	commitInfo2, err := internal.TestReadCommitInfo()
	if err != nil {
		log.Fatalf("Failed to read commit info second time: %v", err)
	}

	if commitInfo.WmemUID == commitInfo2.WmemUID {
		log.Fatalf("Expected different wmem-uids between runs, got same: %s", commitInfo.WmemUID)
	}

	// Step 5: Simulate the flow - same commitInfo should be used for both workdir and wmem-repo commits
	workdirResults := []internal.WorkdirCommitResult{
		{
			WorkdirName: "test-workdir",
			BranchName:  "main",
			CommitHash:  "abc123def456",
			HasChanges:  true,
		},
	}

	// Generate wmem-repo commit message using the same commitInfo
	wmemRepoMessage := internal.TestGenerateWmemRepoCommitMessage(commitInfo, workdirResults)

	// Verify the wmem-repo commit message contains the same wmem-uid
	if !strings.Contains(wmemRepoMessage, expectedUIDLine) {
		log.Fatalf("Wmem-repo commit message does not contain expected wmem-uid line: %s", expectedUIDLine)
	}

	fmt.Println("✅ wmem-uid consistency test passed!")
	fmt.Printf("✅ Same wmem-uid used in both workdir commits and wmem-repo commit: %s\n", commitInfo.WmemUID)
	fmt.Printf("✅ Wmem-repo commit message: %s\n", wmemRepoMessage)
}
