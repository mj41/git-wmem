# UC: git-wmem-commit basic

Create a wmem commit.

## Preconditions:
- Must be executed from within a `wmem-repo` directory (containing `.git-wmem` file)
- `md/commit-workdir-paths` file must exist and contain at least one valid `workdir-path`

1) User runs:
    ```sh
    > git-wmem-commit
    ```

2) `git-wmem-commit` tool executes:
    - [UC: init-repos](#uc-git-wmem-commit-init-repos)
    - [UC: commit-all](#uc-git-wmem-commit-commit-all)


# UC: git-wmem-commit init-repos

- 1) Tool for each `workdir-path` specified in `md/commit-workdir-paths`
    - 1.1) checks that `workdir-name` doesn't exist in `md-internal/workdir-map.json`
    - 1.2) validates new path, see [validations workdir-path](../../validations.md#workdir-path-requirements)
    - 1.3) generates `workdir-name` based on the `workdir-path` and saves that to `md-internal/workdir-map.json` file
    - 1.4) initializes git bare repository in `repos/<workdir-name>.git`
    - 1.5) for bare repository
        - adds git remote named `wmem-wd` pointing to the `workdir-path`
        - `git fetch wmem-wd`

## Details

- `md/commit-workdir-paths` - file containing paths to working directories to commit, one per line (e.g. `../my-projectA`)
- `md-internal/workdir-map.json` - described in [data-structures workdir-map](../../data-structures.md#workdir-map)
- In 1.3) `workdir-name` is the last part of the `workdir-path` (e.g. `../my-dirX/my-projY` → `my-projY`). It is used to name the bare repository in `repos/` directory and to map it in `md-internal/workdir-map.json`. If `workdir-name` already exists in `md-internal/workdir-map.json` then suffix `-2` is added to the `workdir-name` and the process is repeated until a unique name is found.
- `wmem-br/<current-branch-name>` details can be found in [validations branch name requirements](../../validations.md#branch-name-requirements).
- `repos/` directory contains bare repositories, not git submodules

## Alternatives:

- 1b) If `md/commit-workdir-paths` is empty (or doesn't exist) then the tool exits with error: "No workdirs configured for commit. Add paths to your workdirs in md/commit-workdir-paths file."


# UC: git-wmem-commit commit-all

- 1) Tool checks if `md/commit-workdir-paths` file exists
- 2) Tool reads `md/commit/*` and prepares `commit-info` structure
- 3) Tool calls for each `workdir-path` in `md/commit-workdir-paths` the [UC: sync-workdir](#uc-sync-workdir)
- 4) Tool `git add -A` and `git commit` with commit message based on `commit-info` in `wmem-repo`

## Details

- `commit-info` structure details: [data-structures.md](../../data-structures.md#commit-info)
- Commit message generation is described in [data-structures commit-msg](../../data-structures.md#commit-msg)
- `wmem-br/<current-branch-name>` details can be found in [validations branch name requirements](../../validations.md#branch-name-requirements)
- `wmem-br/head` is a special tracking branch that always points to the same commit as the currently checked out branch in `workdir-path`

# UC: sync-workdir

- 1) Tool gets the `<current-branch-name>` of `workdir-path` (e.g. `main`, `feat/X1`)
- 2) Tool ensures `wmem-br/<current-branch-name>` branch exists in `wmem-wd-repo`
- 3) Tool ensures `wmem-br/head` branch exists and points to the current `wmem-br/<current-branch-name>`
- 4) Tool fetches latest changes from `wmem-wd` remote repo
- 5) Tool ensures that `wmem-wd` `<current-branch-name>` commit is already merged to `wmem-wd-repo`'s `wmem-br/<current-branch-name>` branch
- 6) Tool checks that there are modified files in the `workdir-path` compared to `wmem-wd-repo`'s `wmem-br/<current-branch-name>` branch by comparing the current filesystem state with the wmem-tracked tree
- 7) Tool adds all files (like `git add -A`) in `workdir-path` to the "index" in `wmem-wd-repo`
- 8) Tool creates a new commit to `wmem-br/<current-branch-name>` branch based on the "index", new commit message and `commit-info` structure
- 9) Tool updates `wmem-br/head` to point to the new commit

## Details

- 7) Tool will never modify index of `workdir-repo` (the `workdir-path`). All operations must work with read-only access to `workdir-path` and `workdir-repo`.
- 7) Tool will try to add sub-directories that are inner working directories (with `.git` sub-directory inside) in `workdir-path` the same way as `git add -A` does.

## Alternatives:

- 2b) If `wmem-br/<current-branch-name>` branch doesn't exist then the tool creates a new branch named `wmem-br/<current-branch-name>` pointing to the current `workdir-path` HEAD commit.
- 3b) If `wmem-br/head` doesn't exist or points to a different branch then the tool creates or updates `wmem-br/head` to point to the same commit as `wmem-br/<current-branch-name>`.
- 5b) Tool creates a merge commit in `wmem-wd-repo` following [ALG: wmem merge](#alg-wmem-merge) if `workdir-path` HEAD commit is not already merged.
- 6b) If no modified files exist in `workdir-path` compared to the wmem-tracked state, skip steps 7-9 for this workdir with info message about no changes detected

## Error cases:

- 1z.1) If `workdir-path` is not accessible or not a git repository, exit with error

# ALG: wmem merge

A merge commit must be created in `wmem-wd-repo`'s `wmem-br/<current-branch-name>` branch by the tool by accepting the `workdir-path`'s branch HEAD commit's `tree hash`. This will never result in conflicts. The commit message will be based on `commit-info` previously created in `git-wmem-commit commit-all` step.

After creating the merge commit, `wmem-br/head` must be updated to point to the new merge commit to maintain current state tracking.

Golang will be used for implementation. Example approach in shell script:
```sh
# Get current branch name from `workdir-path` (`workdir-repo`)
CURRENT_BRANCH=$(git -C $WORKDIR_PATH branch --show-current)

# Get the commit hash of the `wmem-br/<current-branch-name>` in `wmem-wd-repo`
TARGET_HEAD_COMMIT=$(git rev-parse wmem-br/$CURRENT_BRANCH)

# Get the tree hash (git SHA-1) from `workdir-repo` current branch HEAD commit
WORKDIR_COMMIT=$(git -C $WORKDIR_PATH rev-parse HEAD)
WORKDIR_TREE_SHA1=$(git -C $WORKDIR_PATH rev-parse ${WORKDIR_COMMIT}^{tree})

# Do `git commit-tree <tree> -p <parent1> -p <parent2> -m <commit-msg>` in `wmem-wd-repo`
MERGE_COMMIT_HASH=$(git commit-tree $WORKDIR_TREE_SHA1 -p $TARGET_HEAD_COMMIT -p $WORKDIR_COMMIT -m $COMMIT_MSG)

# Update the wmem-br/<current-branch-name> branch to point to the new merge commit in `wmem-wd-repo`
git update-ref refs/heads/wmem-br/$CURRENT_BRANCH $MERGE_COMMIT_HASH

# Update wmem-br/head to track the current branch in `wmem-wd-repo`
git update-ref refs/heads/wmem-br/head $MERGE_COMMIT_HASH
```
