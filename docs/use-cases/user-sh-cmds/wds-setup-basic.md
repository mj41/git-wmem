# UC: user-sh-cmds wds-setup-basic

## Main scenario

Add workdirs.

1) User creates two git projects:
    ```sh
    > cd ~/work

    > mkdir my-projectA
    > cd my-projectA
    > git init
    > echo "file A content" > fileA.txt
    > git add fileA.txt
    > git commit -m "Initial commit in my-projectA"

    > cd ~/work
    > mkdir my-projectB
    > cd my-projectB
    > git init
    > echo "file B content" > fileB.txt
    > git add fileB.txt
    > git commit -m "Initial commit in my-projectB"
    > cd ~/work
    ```
2) User adds workdirs to wmem:
    ```sh
    > cd ~/work/my-wmem1
    > echo "../my-projectA" >> md/commit-workdir-paths
    > echo "../my-projectB" >> md/commit-workdir-paths
    ```
