package main

import (
	"fmt"
	"os"

	"git-wmem/internal"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: git-wmem-init <directory>\n")
		os.Exit(1)
	}

	targetDir := os.Args[1]

	err := internal.InitWmemRepo(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
