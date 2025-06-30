# UC: user-sh-cmds wds-git-cmds

## Main scenario

User works and uses git commands in workdirs.

1) User creates new branches and commits in two workdirs:
    ```sh
    > cd ~/work/my-projectA
    > git checkout -b feat/X1
    > echo "file file-featX1.txt: content A-X-a, line 1" > file-featX1.txt
    > git add file-featX1.txt
    > git commit -m "Project my-projectA, feature X, commit A-X-a"
    > echo "file file-featX1.txt: content A-X-b, line 2" >> file-featX1.txt
    > git commit -a -m "Project my-projectA, feature X, commit A-X-b"

    > cd ~/work/my-projectB
    > git checkout -b workH
    > mkdir workH-dir
    > echo "file workH-dir/file-workH1.txt: content B-W-a, line 1" > workH-dir/file-workH1.txt
    > git add workH-dir/file-workH1.txt
    > git commit -m "Project my-projectB, feature W, commit B-W-a"
    ```

## Preconditions

- [UC: user-sh-cmds wds-setup-basic](wds-setup-basic.md)
- [UC: user-sh-cmds wds-file-changes](wds-file-changes.md)
