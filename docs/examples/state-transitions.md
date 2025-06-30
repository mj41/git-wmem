# Git-Wmem State Transitions

This document demonstrates how git-wmem tracks repository states through various user actions, showing the evolution of working directory repositories, git repositories, and wmem repositories.

## Initial Setup

### Scenario: Starting with a Simple Git Repository

**Initial State - Working Directory Repository (wd-repo)**:
```
my-project/
├── .git/
├── README.md
└── src/
    └── main.py
```

**Initial State - Git Repository**:
```
commit abc123 (HEAD -> main): Initial commit
Changes:
A  README.md
A  src/main.py
```

**Initial State - Wmem Repository**: Not yet created

---

## State 1: Creating Wmem Repository

### User Action: `git-wmem-init`

**Command**: Initialize wmem repository to track `my-project`
```bash
cd /workspace
git-wmem-init
# Creates wmem-repo tracking working directory: ../my-project
```

**state 1a - working directory repository (wd-repo)**: Unchanged

**state 1b - git repository**: Unchanged

**state 1c - wmem repository**: Created
```
wmem-repo/
├── .git/
├── .git-wmem
├── .gitignore
└── workdirs/
    └── my-project.git/     # Bare git repository clone
```

**state 1d - wmem-wd-repo**: Bare repository created
```
wmem-repo/repos/my-project.git/
├── HEAD                    # Points to refs/heads/wmem-br/main
├── config                  # Git config with wmem-wd remote
├── objects/                # Git objects mirrored from workdir
├── refs/
│   └── heads/
│       └── wmem-br/
│           ├── main        # Points to abc123
│           └── head        # Points to abc123 (tracks current branch)
└── logs/                   # Git reflogs
```

**Wmem Git Log**: 
```
commit wmem001 (HEAD -> wmem-br/main): Initial wmem commit for workdir my-project
Workdir: my-project
Branch: main  
HEAD: abc123
Tree: tree456
```

**Key Changes from Initial**:
- Wmem repository created
- Bare clone of `my-project` created in `workdirs/my-project.git/`
- Initial wmem commit tracks current state of `my-project`

---

## State 2: Create Feature Branch and Wmem Commit

### User Action: Create new branch and sync wmem

**Command**: User creates new branch and runs wmem commit
```bash
cd my-project
git checkout -b feat/add-config
cd ../wmem-repo
git-wmem-commit
```

**state 2a - working directory repository (wd-repo)**: Unchanged files
```
my-project/
├── .git/
├── README.md
└── src/
    └── main.py
```

**state 2b - git repository**: Branch change only
```
commit abc123 (HEAD -> feat/add-config): Initial commit  # SAME COMMIT, different branch
```

**state 2c - wmem repository**: Updated to track new branch
```
wmem-repo/
├── workdirs/my-project.git/    # Now tracking feat/add-config branch
```

**wmem repository git log view**:
```
$ git log --oneline --stat
wmem002 (HEAD -> wmem-br/feat/add-config) Track branch change to feat/add-config
 .wmem-workdir-info           | 2 +-
 md/commit-workdir-paths      | 1 +
 2 files changed, 2 insertions(+), 1 deletion(-)

wmem001 Initial wmem commit for workdir my-project
 .wmem-workdir-info           | 5 +++++
 md/commit-workdir-paths      | 1 +
 md-internal/workdir-map.json | 3 +++
 3 files changed, 9 insertions(+)
```

**state 2d - wmem-wd-repo**: Updated branch tracking
```
wmem-repo/repos/my-project.git/
├── HEAD                    # Points to refs/heads/wmem-br/feat/add-config
├── config                  # Git config with wmem-wd remote
├── objects/                # Same git objects (no new commits)
├── refs/
│   └── heads/
│       └── wmem-br/
│           ├── main        # Points to abc123
│           ├── feat/       
│           │   └── add-config  # NEW: Points to abc123
│           └── head        # Updated: Points to abc123 (tracks feat/add-config)
└── logs/                   # Git reflogs updated
```

**Wmem Git Log**: New wmem commit for branch change
```
commit wmem002 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config                    # Now tracking feature branch
HEAD: abc123                                  # Same commit, different branch
Tree: tree123                                 # Same tree hash

Branch change detected: main → feat/add-config

commit wmem001: Initial wmem commit for workdir my-project
HEAD: abc123
```

**Key Changes from State 1**:
- Git repository switched to feature branch (same commit, different branch)
- Wmem repository now tracks feat/add-config branch instead of main
- Wmem created new commit to track branch change

---

## State 3: Add File Commit and Wmem Commit

### User Action: Add new file, commit to git, and sync wmem

**Command**: User adds file, commits to git, and runs wmem commit
```bash
cd my-project
echo "config = {}" > config.py
git add config.py
git commit -m "Add configuration file"
cd ../wmem-repo
git-wmem-commit
```

**state 3a - working directory repository (wd-repo)**: New file added
```
my-project/
├── .git/
├── README.md
├── config.py          # NEW FILE
└── src/
    └── main.py
```

**state 3b - git repository**: New commit on feature branch
```
commit ghi456 (HEAD -> feat/add-config): Add configuration file  # NEW COMMIT
Changes:
A  config.py

commit abc123 (main): Initial commit
```

**state 3c - wmem repository**: Updated with new file changes
```
$ git log --oneline --stat
wmem003 (HEAD -> wmem-br/feat/add-config) Update workdir my-project
 .wmem-workdir-info           | 2 +-
 repos/my-project.git/config.py | 1 +
 2 files changed, 2 insertions(+), 1 deletion(-)

wmem002 Track branch change to feat/add-config
 .wmem-workdir-info           | 2 +-
 md/commit-workdir-paths      | 1 +
 2 files changed, 2 insertions(+), 1 deletion(-)

wmem001 Initial wmem commit for workdir my-project
 .wmem-workdir-info           | 5 +++++
 md/commit-workdir-paths      | 1 +
 md-internal/workdir-map.json | 3 +++
 3 files changed, 9 insertions(+)
```

**state 3d - wmem-wd-repo**: New git commit and file tracked
```
wmem-repo/repos/my-project.git/
├── HEAD                    # Points to refs/heads/wmem-br/feat/add-config
├── config                  # Git config with wmem-wd remote
├── objects/                # New git objects for ghi456 commit
│   ├── ab/c123...         # Original commit object
│   ├── gh/i456...         # New commit object
│   └── co/nfig...         # New config.py blob object
├── refs/
│   └── heads/
│       └── wmem-br/
│           ├── main        # Points to abc123
│           ├── feat/       
│           │   └── add-config  # Updated: Points to ghi456
│           └── head        # Updated: Points to ghi456 (tracks feat/add-config)
└── logs/                   # Git reflogs updated
```

**Wmem Git Log**: New commit for file addition
```
commit wmem003 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config                    # Still tracking feature branch
HEAD: ghi456                                  # Updated to new commit
Tree: tree456                                 # New tree with config.py

Files changed since last wmem commit:
A  config.py

commit wmem002: Update workdir my-project (branch change)
commit wmem001: Initial wmem commit for workdir my-project
```

**Key Changes from State 2**:
- Working directory has new file: `config.py`
- Git repository has new commit: `ghi456` with the new file
- Wmem repository synced with new commit and file changes

---

## State 4: Add New File and Wmem Commit

### User Action: Add new file and run wmem commit

**Command**: User adds new file and runs wmem commit
```bash
cd my-project
echo "print('Hello World')" > src/hello.py
# File added but not yet committed to git

cd ../wmem-repo
git-wmem-commit
```

**state 4a - working directory repository (wd-repo)**: Changed
```
my-project/
├── .git/
├── README.md
├── config.py
└── src/
    ├── main.py
    └── hello.py           # NEW UNTRACKED FILE
```

**state 4b - git repository**: Unchanged

**state 4c - wmem repository**: Updated with untracked file
```
commit wmem003 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config
HEAD: ghi456                              # Same git HEAD, but includes untracked
Tree: tree789

Files changed since last wmem commit:
A  src/hello.py                           # Untracked file included

commit wmem002: Update workdir my-project
HEAD: ghi456
```

**Key Changes from State 3**:
- Working directory has new untracked file: `src/hello.py`
- Wmem repository tracks the new untracked file

---

## State 5: Modify File and Wmem Commit

### User Action: Modify the file and run wmem commit

**Command**: User modifies the file and runs wmem commit
```bash
cd my-project
echo "print('Hello World, updated!')" > src/hello.py
# File modified but still not committed to git

cd ../wmem-repo
git-wmem-commit
```

**state 5a - working directory repository (wd-repo)**: Changed (file content updated)

**state 5b - git repository**: Unchanged

**state 5c - wmem repository**: Updated with modified file
```
commit wmem004 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config
HEAD: ghi456                              # Same git HEAD
Tree: tree890

Files changed since last wmem commit:
M  src/hello.py                           # Modified file

commit wmem003: Update workdir my-project
HEAD: ghi456
```

**Key Changes from State 4**:
- File `src/hello.py` content modified
- Wmem repository tracks the modification

---

## State 6: Remove File and Wmem Commit

### User Action: Remove the file and run wmem commit

**Command**: User removes the file and runs wmem commit
```bash
cd my-project
rm src/hello.py
# File removed from filesystem

cd ../wmem-repo
git-wmem-commit
```

**state 6a - working directory repository (wd-repo)**: Changed
```
my-project/
├── .git/
├── README.md
├── config.py
└── src/
    └── main.py            # hello.py removed
```

**state 6b - git repository**: Unchanged

**state 6c - wmem repository**: Updated with deleted file
```
commit wmem005 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config
HEAD: ghi456                              # Same git HEAD
Tree: tree999

Files changed since last wmem commit:
D  src/hello.py                           # Deleted file

commit wmem004: Update workdir my-project
HEAD: ghi456
```

**Key Changes from State 5**:
- File `src/hello.py` removed from filesystem
- Wmem repository tracks the deletion

---

## State 7: Remove Existing File and Wmem Commit

### User Action: Remove an existing tracked file and run wmem commit

**Command**: User removes an existing git-tracked file and runs wmem commit
```bash
cd my-project
rm config.py
# File removed from filesystem (was tracked by git)

cd ../wmem-repo
git-wmem-commit
```

**state 7a - working directory repository (wd-repo)**: Changed
```
my-project/
├── .git/
├── README.md
└── src/
    └── main.py            # config.py removed
```

**state 7b - git repository**: Unchanged (file still tracked by git)

**state 7c - wmem repository**: Updated with deleted tracked file
```
commit wmem006 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config
HEAD: ghi456                              # Same git HEAD
Tree: tree111

Files changed since last wmem commit:
D  config.py                              # Deleted tracked file

commit wmem005: Update workdir my-project
HEAD: ghi456
```

**Key Changes from State 6**:
- File `config.py` removed from filesystem (was git-tracked)
- Wmem repository tracks the deletion of git-tracked file

---

## State 8: Add File Back and Wmem Commit

### User Action: Add the file back and run wmem commit

**Command**: User recreates the file and runs wmem commit
```bash
cd my-project
echo "config = {'restored': True}" > config.py
# File recreated with new content

cd ../wmem-repo
git-wmem-commit
```

**state 8a - working directory repository (wd-repo)**: Changed
```
my-project/
├── .git/
├── README.md
├── config.py              # File restored (but with new content)
└── src/
    └── main.py
```

**state 8b - git repository**: Unchanged

**state 8c - wmem repository**: Updated with restored file
```
commit wmem007 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config
HEAD: ghi456                              # Same git HEAD
Tree: tree222

Files changed since last wmem commit:
A  config.py                              # Re-added file (treated as new)

commit wmem006: Update workdir my-project
HEAD: ghi456
```

**Key Changes from State 7**:
- File `config.py` recreated with new content
- Wmem repository tracks the file as newly added (not a restoration)

---

## State 9: Restore File to Original Version and Commit

### User Action: Restore config.py to original version and commit to git

**Command**: User restores config.py to original content and commits to git
```bash
cd my-project
echo "config = {}" > config.py
git add config.py
git commit -m "Restore config.py to original version"
cd ../wmem-repo
git-wmem-commit
```

**state 9a - working directory repository (wd-repo)**: File content restored
```
my-project/
├── .git/
├── README.md
├── config.py              # Content restored to original
└── src/
    └── main.py
```

**state 9b - git repository**: New commit with restored file
```
commit jkl789 (HEAD -> feat/add-config): Restore config.py to original version  # NEW COMMIT
Changes:
A  config.py

commit ghi456 (feat/add-config): Add configuration file
commit abc123 (main): Initial commit
```

**state 9c - wmem repository**: Updated with git commit
```
commit wmem009 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config
HEAD: jkl789                              # Updated to new commit
Tree: tree333

Files changed since last wmem commit:
M  config.py                              # File content updated, now git-tracked again

commit wmem008: Update workdir my-project
HEAD: ghi456
```

**Key Changes from State 8**:
- File `config.py` content restored to original version
- New git commit: `jkl789` 
- Wmem repository synced with git commit
- File now tracked by both git and wmem again

---

## State 10: Remove Existing Tracked File

### User Action: Remove config.py that is tracked by git

**Command**: User removes config.py from filesystem
```bash
cd my-project
rm config.py
# File removed from filesystem (still tracked by git)

cd ../wmem-repo
git-wmem-commit
```

**state 10a - working directory repository (wd-repo)**: File removed
```
my-project/
├── .git/
├── README.md
└── src/
    └── main.py            # config.py removed
```

**state 10b - git repository**: Unchanged (file still tracked)
```
commit jkl789 (HEAD -> feat/add-config): Restore config.py to original version
```

**state 10c - wmem repository**: Updated with file deletion
```
commit wmem010 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config
HEAD: jkl789                              # Same git HEAD
Tree: tree444

Files changed since last wmem commit:
D  config.py                              # Deleted from filesystem

commit wmem009: Update workdir my-project
HEAD: jkl789
```

**Key Changes from State 9**:
- File `config.py` removed from filesystem
- Git repository unchanged (file still tracked by git)
- Wmem repository tracks the filesystem deletion

---

## State 11: Add File Back Again

### User Action: Recreate config.py and run wmem commit

**Command**: User recreates config.py with different content
```bash
cd my-project
echo "config = {'version': 2}" > config.py
# File recreated with new content

cd ../wmem-repo
git-wmem-commit
```

**state 11a - working directory repository (wd-repo)**: File recreated
```
my-project/
├── .git/
├── README.md
├── config.py              # File recreated with different content
└── src/
    └── main.py
```

**state 11b - git repository**: Unchanged
```
commit jkl789 (HEAD -> feat/add-config): Restore config.py to original version
```

**state 11c - wmem repository**: Updated with file recreation
```
commit wmem011 (HEAD -> wmem-br/feat/add-config): Update workdir my-project   # NEW WMEM COMMIT
Workdir: my-project
Branch: feat/add-config
HEAD: jkl789                              # Same git HEAD
Tree: tree555

Files changed since last wmem commit:
A  config.py                              # Added back with new content

commit wmem010: Update workdir my-project
HEAD: jkl789
```

**Key Changes from State 10**:
- File `config.py` recreated with new content
- Git repository unchanged
- Wmem repository tracks the file recreation

---

## State 12: Git Merge Commit and Wmem Sync

### User Action: Create feature branch, make changes, merge back, and sync wmem

**Command**: User creates new branch, adds feature, merges, and runs wmem commit
```bash
cd my-project
git checkout -b feat/new-feature
echo "new_feature = True" > feature.py
git add feature.py
git commit -m "Add new feature"

# Switch back and merge
git checkout feat/add-config
git merge feat/new-feature
# Creates merge commit

cd ../wmem-repo
git-wmem-commit
```

**state 12a - working directory repository (wd-repo)**: New file from merge
```
my-project/
├── .git/
├── README.md
├── config.py              # Still present
├── feature.py             # NEW FILE from merged branch
└── src/
    └── main.py
```

**state 12b - git repository**: Merge commit created
```
commit mno012 (HEAD -> feat/add-config): Merge branch 'feat/new-feature' into feat/add-config  # MERGE COMMIT
Parents: jkl789 pqr456
Changes (from merge):
A  feature.py

commit pqr456 (feat/new-feature): Add new feature    # Feature branch commit  
Changes:
A  feature.py

commit jkl789: Restore config.py to original version # Previous HEAD
```

**state 12c - wmem repository**: Updated with merge commit
```
commit wmem012 (HEAD -> wmem-br/feat/add-config): Merge workdir 'feat/add-config' into 'wmem-br/feat/add-config' accepting workdir's branch tree hash   # NEW WMEM MERGE COMMIT

Workdir: my-project
Branch: feat/add-config
HEAD: mno012                              # Merge commit hash
Tree: tree666                             # New tree with feature.py

wmem-uid: wmem-250706-220000-xyz89abc

Files changed since last wmem commit:
A  feature.py                             # Added from merge

commit wmem011: Update workdir my-project
HEAD: jkl789
```

**Key Changes from State 11**:
- New git merge commit: `mno012` (2 parents: `jkl789` + `pqr456`)
- New file: `feature.py` from merged feature branch
- Wmem creates merge commit using ALG: wmem merge (Alternative 5b)
- Demonstrates wmem handling of git merge workflows

---

## State Comparison Summary

> **Note**: `feat/add` is abbreviated from `feat/add-config` for table readability.
> **File shortcuts**: A, B = 2 files; A, B, C = 3 files; A, B, C, D = 4 files (+ modifiers)

| State | Working Dir | Git Repo HEAD | Wmem Repo HEAD | Sync Status |
|-------|-------------|---------------|----------------|-------------|
| 1 (init) | A, B | `abc123` (main) | `wmem001 -> abc123` | Synced |
| 2 (feature branch) | A, B | `abc123` (feat/add) | `wmem002 -> abc123` | Synced |
| 3 (add file) | A, B, C | `ghi456` (feat/add) | `wmem003 -> ghi456` | Synced |
| 4 (add file) | A, B, C, D (1 untracked) | `ghi456` (feat/add) | `wmem004 -> ghi456` | Synced |
| 5 (modify file) | A, B, C, D (1 modified) | `ghi456` (feat/add) | `wmem005 -> ghi456` | Synced |
| 6 (remove file) | A, B, C | `ghi456` (feat/add) | `wmem006 -> ghi456` | Synced |
| 7 (remove tracked) | A, B | `ghi456` (feat/add) | `wmem007 -> ghi456` | Synced |
| 8 (add back) | A, B, C | `ghi456` (feat/add) | `wmem008 -> ghi456` | Synced |
| 9 (restore & commit) | A, B, C | `jkl789` (feat/add) | `wmem009 -> jkl789` | Synced |
| 10 (remove tracked) | A, B | `jkl789` (feat/add) | `wmem010 -> jkl789` | Synced |
| 11 (add back) | A, B, C | `jkl789` (feat/add) | `wmem011 -> jkl789` | Synced |
| 12 (merge commit) | A, B, C, D | `mno012` (feat/add) | `wmem012 -> mno012` | Synced |

## Algorithm Behavior Analysis

### Change Detection Results

**State 1 → 2**: `git-wmem-commit` executed:
- User creates feature branch (same commit, different branch)
- Branch change detected (main → feat/add-config)
- Created wmem commit for branch change tracking

**State 2 → 3**: `git-wmem-commit` executed:
- New git commit: `ghi456` with new file
- New file: `config.py`
- Created wmem commit with file addition

**State 3 → 4**: `git-wmem-commit` executed:
- Untracked file in working directory: `src/hello.py`
- Created wmem commit (includes untracked files)

**State 4 → 5**: `git-wmem-commit` executed:
- Modified file: `src/hello.py`
- Created wmem commit with modification

**State 5 → 6**: `git-wmem-commit` executed:
- Deleted file: `src/hello.py`
- Created wmem commit tracking deletion

**State 6 → 7**: `git-wmem-commit` executed:
- Deleted git-tracked file: `config.py`
- Created wmem commit tracking deletion of tracked file

**State 7 → 8**: `git-wmem-commit` executed:
- Re-added file: `config.py` (with new content)
- Created wmem commit treating it as new file addition

**State 8 → 9**: `git-wmem-commit` executed:
- File content updated and committed to git
- New git commit: `jkl789`
- Created wmem commit syncing with git commit

**State 9 → 10**: `git-wmem-commit` executed:
- Deleted git-tracked file: `config.py` from filesystem
- Created wmem commit tracking deletion of git-tracked file

**State 10 → 11**: `git-wmem-commit` executed:
- Re-added file: `config.py` (with different content)
- Created wmem commit treating it as new file addition

**State 11 → 12**: `git-wmem-commit` executed:
- Git merge commit: `mno012` (merges feat/new-feature into feat/add-config)
- New file from merge: `feature.py`
- Created wmem merge commit preserving both git and wmem histories

### Algorithm Purpose Analysis

**Why wmemTree.Files().ForEach() is Essential**:

- **State 5 → 6**: Must detect `src/hello.py` deletion from working directory
- **State 6 → 7**: Must detect `config.py` deletion from git-tracked files  
- **State 7 → 8**: Must detect `config.py` re-addition vs original deletion
- **State 9 → 10**: Must detect `config.py` deletion from git-tracked files
- **State 10 → 11**: Must detect `config.py` recreation vs original deletion
- **State 11 → 12**: Must detect `feature.py` addition from git merge

The ForEach iteration compares current working directory state against wmem's last known state, enabling detection of all file lifecycle changes including deletions, recreations, and merge additions.
