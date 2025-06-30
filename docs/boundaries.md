# Out of Scope

These are out of scope for the current version of `git-wmem`:

- MacOS, Windows, and other operating systems
- Other Linux distributions besides Linux Fedora 42+
- Other than the supported [Use Cases](use-cases.md), including variants (and error cases) not explicitly supported

# Design principles

## Simplicity

Keep it simple, stupid (KISS). Avoid unnecessary complexity in the design and implementation of `git-wmem` tools. Follow the Unix philosophy of small tools that do one thing well.

## Read-only access to `workdir-path` and `workdir-repo`

`git-wmem` tools will not write to any `workdir-path` or `workdir-repo`. Read-only access must be sufficient.

## Others

- Avoid shell scripts where Golang tools can be used.
- Use the `go-git` library for git operations. Except for e2e tests, where the `git` binary is used directly.
- Concurrent operations on more than one `workdir` are not supported.
- Use a simple Makefile for building tools.
- Accept that `wmem-uid` collision is theoretically possible, but practically unlikely.
- No performance tests yet. No large repos.