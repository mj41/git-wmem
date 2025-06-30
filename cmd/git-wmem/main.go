package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"git-wmem/internal"
)

//go:embed git-wmem.md
var readmeContent string

//go:embed help.txt
var helpContent string

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile = flag.String("memprofile", "", "write memory profile to file")
	readme     = flag.Bool("readme", false, "show full documentation")
	showHelp   = flag.Bool("help", false, "show usage information")
	version    = flag.Bool("version", false, "show version information")
)

// GitSHA is set at build time
var GitSHA = "dev"

func main() {
	// Define flags first
	flag.Usage = func() {
		fmt.Print(helpContent)
	}

	flag.Parse()

	if *readme {
		fmt.Println(readmeContent)
		return
	}

	if *showHelp {
		fmt.Print(helpContent)
		return
	}

	if *version {
		fmt.Printf("git-wmem version %s\n", GitSHA)
		return
	}

	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Start CPU profiling if requested
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "init":
		if len(commandArgs) != 1 {
			fmt.Fprintf(os.Stderr, "Usage: git-wmem init <directory>\n")
			os.Exit(1)
		}
		err := internal.InitWmemRepo(commandArgs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "commit":
		if len(commandArgs) != 0 {
			fmt.Fprintf(os.Stderr, "Usage: git-wmem commit\n")
			os.Exit(1)
		}
		err := internal.CommitWmem()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "log":
		if len(commandArgs) != 0 {
			fmt.Fprintf(os.Stderr, "Usage: git-wmem log\n")
			os.Exit(1)
		}
		err := internal.LogWmem()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintf(os.Stderr, "Available commands: init, commit, log\n")
		os.Exit(1)
	}

	// Write memory profile if requested
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
