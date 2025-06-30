# UC: user-sh-cmds wds-file-changes

User works in workdirs without committing changes.

1) User creates new branches and files in two workdirs:
    ```sh
    > cd ~/work/my-projectA
    > echo "file file-featX1.txt: content A-X-pre-a, line 1" > file-featX1.txt

    > cd ~/work/my-projectB
    > git checkout -b workH
    > mkdir workH-dir
    > echo "file workH-dir/file-workH1.txt: content B-W-pre-a, line 1" > workH-dir/file-workH1.txt
    ```

## Preconditions

- [UC: user-sh-cmds wds-setup-basic](wds-setup-basic.md)
