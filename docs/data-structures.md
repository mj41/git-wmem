# Data Structures and Values

## `commit-info`

Format:
- `wmem-uid` - see `wmem-uid` value below
- `msg-prefix` - commit message prefix, see below for details on how the commit message is generated
- `author` - author from `md/commit/author` file (if not empty)
- `committer` - committer from `md/commit/committer` file (if not empty)

Both `md/commit/author` and `md/commit/committer` files are mandatory and their content must be in valid git author/committer format. Default values (file contents) are created during [UC: git-wmem-init basic](use-cases/git-wmem-init/basic.md). If the file doesn't exist or is empty then a fatal error is raised.

### Commit message generation example

`md/commit/msg-prefix` with content:
```
My git-wmem commit prefix
```

Commit message example:
```
My git-wmem commit prefix

wmem-uid: wmem-250628-143022-abXY1234

<msg-body>
```

where `<msg-body>` is the body of the commit message.

E.g. `wmem-wd-repo` `projectB` `<msg-body>` could be:
```
wmem-commit to merge workdir branch `feature/X2` by accepting its tree hash
```

`wmem-wd-repo` `projectA` `<msg-body>` could be:
```
wmem-commit of workdir uncommitted changes.
```

`wmem-repo` `<msg-body>` could be:
```
Meta wmem-commit of workdir commits
- `my-projectA` `main` `c123456`
- `my-projectB` `feature/X2` `c789012`
```


## `wmem-uid`

Unique identifier that is a combination of the date, time, and a random string. It is used to track commits across all configured workdirs and to create a unique reference for each commit in the `wmem-repo`. This identifier is generated during a `git-wmem-commit` run and is stored in the `commit-info` structure.

`wmem-uid` examples:
- `wmem-250628-143022-abXY1234`

`wmem-uid` format:
- `wmem-` prefix is used to indicate that this is a `wmem-uid`
- `250628` - date in format `YYMMDD`
- `143022` - time in format `HHMMSS`
- `abXY1234` - random 8-character string `[a-zA-Z0-9]`


## `workdir-map`

Saved in the `md-internal/workdir-map.json` file, it maps `workdir-name` to `workdir-path`. It is used to track all workdirs and their paths in the `wmem-repo`.

The structure is "append only" (only new entries are added). The mapping preserves the complete history of the `wmem-repo` even for workdirs that are no longer tracked.

`workdir-map` example:
```json
{
    "my-projectA": "../my-projectA",
    "my-projectB": "../my-projectB",
    "my-projectA-2": "../my-second-clones/my-projectA"
}
```
