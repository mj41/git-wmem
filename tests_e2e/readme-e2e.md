# Git-Wmem E2E Tests

End-to-end tests for the git-wmem-v2 tools, written in Go and based on the use cases defined in the specification.

## Overview

This test suite validates the complete functionality of git-wmem tools by running actual commands and verifying their behavior. Each test includes references to the corresponding specification documents.

## Test Structure

### Core Test Files

- `test_helper.go` - Common utilities and test infrastructure
- `init_test.go` - Tests for `git-wmem-init` command
  - Reference: `docs/use-cases/git-wmem-init/basic.md`
- `commit_test.go` - Tests for `git-wmem-commit` command
  - Reference: `docs/use-cases/git-wmem-commit/basic.md`
- `log_test.go` - Tests for `git-wmem-log` command
  - Reference: `docs/use-cases/git-wmem-log/basic.md`
- `workflow_test.go` - Complete basic development workflow
  - Reference: `docs/use-cases/use-cases.md#uc-basic-development-workflow`

### Validation Tests

- `validations_test.go` - Path and branch name validation tests
  - Reference: `docs/validations.md`
- `data_structures_test.go` - Data structure format and behavior tests
  - Reference: `docs/data-structures.md`
- `advanced_test.go` - Advanced scenarios and edge cases
  - Reference: Various specification sections

## Test Execution

### Prerequisites

1. Go 1.21 or later
2. Git binary available in PATH
3. `git-wmem` tools available in PATH (init, commit, log)

### Running Tests

```bash
# Run all tests
cd e2e
go test -v

# Run specific test file
go test -v -run TestGitWmemInit

# Run specific test function
go test -v -run TestGitWmemInit_Basic

# Run with timeout (for long-running tests)
go test -v -timeout 10m
```

### Test Environment

- Tests use isolated temporary directories under `/tmp/git-wmem-e2e/`
- Each test gets a unique temporary directory with format: `YYMMDD-HHMMSS-<random>`
- Temporary directories are cleaned up after successful tests
- Failed tests preserve directories for debugging

## Test Coverage

### Use Cases Covered

1. Basic Development Workflow (Complete end-to-end)
   - git-wmem-init → setup workdirs → commit → file changes → commit → git commands → commit → log

2. git-wmem-init
   - ✅ Basic initialization
   - ✅ Current directory initialization
   - ✅ Error: directory not empty
   - ✅ Error: directory exists and not empty

3. git-wmem-commit
   - ✅ Basic commit functionality
   - ✅ Init-repos sub-operation
   - ✅ Commit-all sub-operation
   - ✅ Workdir-map.json management
   - ✅ Bare repository creation
   - ✅ File changes handling
   - ✅ Git commands in workdirs
   - ✅ Commit message prefix
   - ✅ Error: no workdirs configured
   - ✅ Error: not in wmem-repo

4. git-wmem-log
   - ✅ Basic log functionality
   - ✅ Multiple commits display
   - ✅ wmem-uid format in output
   - ✅ Workdir information display
   - ✅ Error: not in wmem-repo

### Validation Rules Covered

5. Workdir Path Requirements
   - ✅ Valid relative paths (../)
   - ✅ Valid nested relative paths
   - ✅ Valid deep relative paths
   - ✅ Error: absolute paths not allowed
   - ✅ Error: excessive path traversal
   - ✅ Error: local subdirectory paths
   - ✅ Error: pointing to wmem-repo itself
   - ✅ Error: pointing to wmem-repo subdirs

6. Branch Name Requirements
   - ✅ Simple branch names (main → wmem-br/main)
   - ✅ Feature branches (feat/X1 → wmem-br/feat/X1)
   - ✅ Hyphenated branches (dev-branch → wmem-br/dev-branch)

### Data Structures Covered

7. wmem-uid Format
   - ✅ Format validation (wmem-YYMMDD-HHMMSS-abXY1234)
   - ✅ Uniqueness across multiple commits
   - ✅ Proper embedding in commit messages

8. commit-info Structure
   - ✅ Message prefix inclusion
   - ✅ wmem-uid field presence
   - ✅ Author/committer information
   - ✅ Default values from md/commit/ files

9. workdir-map Management
   - ✅ Initial empty state
   - ✅ Mapping creation and updates
   - ✅ Append-only behavior
   - ✅ Name collision handling

### Advanced Scenarios Covered

10. Wmem Merge Algorithm
    - ✅ Merge commit creation when branches diverge
    - ✅ Tree hash acceptance from workdir
    - ✅ Conflict-free merge behavior

11. Branch Handling
    - ✅ Automatic wmem-br/ branch creation
    - ✅ Branch switching in workdirs
    - ✅ Multiple branch tracking

12. Error Conditions
    - ✅ Skipping workdirs with no changes
    - ✅ Invalid/non-existent workdir paths
    - ✅ Non-git directories
    - ✅ Operations outside wmem-repo

## Reference Documentation

Each test function includes detailed references to the specification:

```go
// Reference: docs/use-cases/git-wmem-init/basic.md#main-scenario
// Reference: docs/data-structures.md#wmem-uid
// Reference: docs/validations.md#workdir-path-requirements
```

These references ensure tests remain synchronized with the specification and provide traceability.

## Implementation Notes

- Uses Git binary via `exec.Command` as specified in boundaries.md
- Avoids the `go-git` library per design principles
- Creates realistic test scenarios matching actual usage patterns
- Validates both success and error conditions comprehensively
- Provides detailed assertion messages for debugging
- Preserves test isolation through unique temporary directories
