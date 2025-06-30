package main

import (
	"fmt"
	"os"

	"git-wmem/internal"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: git-wmem-commit\n")
		os.Exit(1)
	}

	err := internal.CommitWmem()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
