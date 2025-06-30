# UC: git-wmem-init basic

## Main Scenario: Initialize a New Working Memory Repository

1) User runs:
    ```sh
    > cd ~/work
    > git-wmem-init my-wmem1
    ```
2) `git-wmem-init` checks that the directory `my-wmem1` doesn't exist
3) `git-wmem-init` creates a new `wmem-repo` directory structure:
    - `.git-wmem` file (empty) - indicating that this is a `wmem-repo`
    - `.git/` directory - initialized git repository with default branch `main`
    - `.gitignore` file with content `repos/` - gitignored directories
    - `md/` directory - metadata directory
    - `md/commit-workdir-paths` file (empty) - used to store `workdir-path`s
    - `md/commit/msg-prefix` file (empty) - used to store the commit message prefix
    - `md/commit/author` file (`WMem Git <git-wmem@mj41.cz>`) - used to store author information
    - `md/commit/committer` file (`WMem Git <git-wmem@mj41.cz>`) - used to store committer information
    - `md-internal/` directory - internal metadata directory
    - `md-internal/workdir-map.json` file with content `{}` - used to store `workdir-path` to `workdir-name` mapping
    - `repos/` directory (empty) - used to store bare repositories
4) `git-wmem-init` creates an initial commit:   
    ```
    Initialize git-wmem repository `my-wmem1`
    ```
5) User navigates to the new directory:
    ```sh
    > cd my-wmem1
    ```

## Alternatives:

- 1b) User can also run:
    ```sh
    > cd ~/work
    > mkdir my-wmem1
    > cd my-wmem1
    > git-wmem-init .
    ```
- 2b) If the `my-wmem1` directory already exists, then `git-wmem-init` checks that it is empty. If not empty, then it exits with an error: "Directory is not empty. Please specify an empty directory to initialize wmem-repo."
