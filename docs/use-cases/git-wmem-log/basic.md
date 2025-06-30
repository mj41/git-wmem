# UC: git-wmem-log basic

Show wmem commit history.

## Main Scenario

1) User runs:
    ```sh
    > cd ~/work/my-wmem1
    > git-wmem-log
    ```

2) `git-wmem-log` tool:
    - Reads `wmem-repo` commit history
    - Displays commit messages with `wmem-uid`
    - For each `workdir-path` included in the commit:
        - Displays `workdir-name`
        - Displays commit hash from the corresponding `repos/<workdir-name>.git`

## Example Output Format

```
wmem-250628-143022-abXY1234: projA and projB features
  my-projectA: a1b2c3d4e5f6...
  my-projectB: f6e5d4c3b2a1...

wmem-250627-120000-xyz9876A: initial setup
  my-projectA: 1234567890ab...
  my-projectB: abcdef123456...
```
