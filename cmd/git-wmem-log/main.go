package main

import (
	"fmt"
	"os"

	"git-wmem/internal"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: git-wmem-log\n")
		os.Exit(1)
	}

	err := internal.LogWmem()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
