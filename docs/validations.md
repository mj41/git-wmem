# Validations

## Workdir Path Requirements

When specifying `workdir-path`s in `md/commit-workdir-paths`, the following rules apply:

`workdir-path` rules:
- Must be relative paths starting with `../`. Absolute paths are not allowed
- Must point to readable directories containing valid git repositories
- Paths can start with one or more `..` but cannot contain `..` later in the path
- Workdir cannot be the `wmem-repo` or its subdirectories. Cannot contain a `.git-wmem` file anywhere in the path

Valid examples:
```
../my-projectA
../other-projects/my-projectB
../../../workspace/external-project
```

Invalid examples:
```
/absolute/path/to/project          # Absolute paths not allowed
../my-project/../../../etc         # Excessive path traversal
./relative-subdir                  # Must start with ../
../wmem-repo/subdir                # Cannot point to wmem-repo subdirs
```

## Branch Name Requirements

When creating branches in bare repositories within the `repos/` directory, the following rules apply:
- Original workdir branch names are prefixed with `wmem-br/` to distinguish them from regular branches
- Branch names must follow the same naming pattern as regular git branches

Examples:
- `main` → `wmem-br/main`
- `feat/X1` → `wmem-br/feat/X1`
- `dev-branch` → `wmem-br/dev-branch`
