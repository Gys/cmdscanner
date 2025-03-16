package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/mod/modfile"
)

/*

	A simple Go program that parses a go.mod file and prints out the module information.

	./cmdscanner -file /path/to/go.mod

*/

func main() {
	// Define command-line flags
	goModPath := flag.String("file", "go.mod", "Path to the go.mod file to parse")
	flag.Parse()

	// Check if the file exists
	if _, err := os.Stat(*goModPath); os.IsNotExist(err) {
		log.Fatalf("Error: go.mod file not found at %s", *goModPath)
	}

	// Read the go.mod file
	data, err := os.ReadFile(*goModPath)
	if err != nil {
		log.Fatalf("Error reading go.mod file: %v", err)
	}

	// Parse the go.mod file
	file, err := modfile.Parse(*goModPath, data, nil)
	if err != nil {
		log.Fatalf("Error parsing go.mod file: %v", err)
	}

	// Print module information
	fmt.Printf("Module: %s\n", file.Module.Mod.Path)
	fmt.Printf("Go version: %s\n\n", file.Go.Version)

	// Print direct dependencies
	fmt.Println("Direct Dependencies:")
	for _, req := range file.Require {
		if !req.Indirect {
			fmt.Printf("  - %s %s\n", req.Mod.Path, req.Mod.Version)
		}
	}

	// Print indirect dependencies
	fmt.Println("\nIndirect Dependencies:")
	for _, req := range file.Require {
		if req.Indirect {
			fmt.Printf("  - %s %s\n", req.Mod.Path, req.Mod.Version)
		}
	}

	// Print replace directives if any
	if len(file.Replace) > 0 {
		fmt.Println("\nReplace Directives:")
		for _, rep := range file.Replace {
			fmt.Printf("  - %s %s => %s %s\n",
				rep.Old.Path, rep.Old.Version,
				rep.New.Path, rep.New.Version)
		}
	}

	// Print exclude directives if any
	if len(file.Exclude) > 0 {
		fmt.Println("\nExclude Directives:")
		for _, excl := range file.Exclude {
			fmt.Printf("  - %s %s\n", excl.Mod.Path, excl.Mod.Version)
		}
	}
}
