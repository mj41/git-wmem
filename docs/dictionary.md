# git-wmem Dictionary

## Core Terms

- `wmem-repo` - Main git repository containing the `.git-wmem` file and the `md/` and `repos/` directories where all wmem operations are coordinated.

- `workdir` - A working directory containing a git project that is being tracked by wmem.

- `workdir-path` - Relative path to a `workdir` (e.g. `../my-projectA`) from `wmem-repo` directory.

- `workdir-name` - Derived name from `workdir-path` used for naming repos (e.g. `my-projectA`).

- `workdir-repo` - The `workdir`'s Git repository located at `workdir-path`, e.g. `../my-projectA/.git`.

- `wmem-wd-repo` - Git repository located in the `repos/` directory, specifically `repos/<workdir-name>.git`, which is a bare repository tracking changes for a specific `workdir`.

## Files and Directories

- `md/` - Metadata directory containing configuration files (not gitignored).

- `md/commit-workdir-paths` - Text file listing `workdir-path`s, one per line, that will be used for the next commit.

- `md/commit/msg-prefix` - Text file containing the commit message prefix for the next `git-wmem-commit`.

- `md-internal/` - Internal metadata directory (not gitignored). Should not be modified manually.

- `md-internal/workdir-map.json` - JSON file mapping all `workdir-path`s to `workdir-name`s (not gitignored).

- `repos/` - Directory containing all wmem bare repositories (gitignored).

- `repos/<workdir-name>.git` - Bare git repository tracking changes for a specific workdir.

- `.git-wmem` - Marker file indicating a directory is a `wmem-repo`.

## Git Terms

- `bare repository` - Git repository without a working directory, used for storage only.

- `tree hash` - Unique identifier for a specific state of the file tree (content, blobs), directories, and related filesystem metadata. Each commit points to exactly one tree hash.

- `wmem-wd` - Git remote name pointing from the `wmem-repo` to the `workdir`.

- `wmem-br/<branch-name>` - Branch naming pattern in wmem repos (e.g. `wmem-br/main`).

- `wmem-br/head` - Special tracking branch in wmem repos that always points to the same commit as the currently checked out branch in the corresponding workdir. Used to detect branch switches and maintain current state tracking.

## Data Structures

- `commit-info` - Structure containing commit metadata including `wmem-uid`, commit message, author, and committer information. See [Data Structures and Values](data-structures.md#commit-info) for details.

- `wmem-uid` - Unique identifier with format `wmem-YYMMDD-HHMMSS-abXY1234`. See [Data Structures and Values](data-structures.md#wmem-uid) for details.


## Operations

- `git-wmem-init` - Initialize a new `wmem-repo` with basic structure.

- `git-wmem-commit` - Create commits across all configured repos/workdirs and create a `wmem-repo` commit referencing them.

- `init-repos` - Sub-operation to create bare repos for new workdirs.

- `commit-all` - Sub-operation to commit changes across all workdirs.

- `git-wmem-log` - Display wmem commit history with `wmem-uid` and workdir information.

- `commit-workdir` - Sub-operation to commit changes in a single workdir.
