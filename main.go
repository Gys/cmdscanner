package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// getModuleCachePath returns the path to the Go module cache
func getModuleCachePath() (string, error) {
	cmd := exec.Command("go", "env", "GOMODCACHE")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to older method if GOMODCACHE is not available
		cmd := exec.Command("go", "env", "GOPATH")
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get GOPATH: %v", err)
		}
		return filepath.Join(strings.TrimSpace(string(output)), "pkg", "mod"), nil
	}
	return strings.TrimSpace(string(output)), nil
}

// getPackageInstallPath returns the filesystem path where a package is installed
func getPackageInstallPath(modulePath, version string, moduleCachePath string) string {
	// Encode the module path to handle special characters
	encodedPath, err := module.EscapePath(modulePath)
	if err != nil {
		log.Printf("Warning: Could not encode module path %s: %v", modulePath, err)
		encodedPath = modulePath
	}

	// For the version, we need to handle the "v" prefix and any "+incompatible" suffix
	cleanVersion := version
	if strings.HasSuffix(cleanVersion, "+incompatible") {
		cleanVersion = strings.TrimSuffix(cleanVersion, "+incompatible")
	}

	// Construct the path
	return filepath.Join(moduleCachePath, encodedPath+"@"+cleanVersion)
}

// checkPackageExists verifies if the package is installed at the expected location
func checkPackageExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

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

	// Get the module cache path
	moduleCachePath, err := getModuleCachePath()
	if err != nil {
		log.Fatalf("Error getting module cache path: %v", err)
	}

	// Print module information
	fmt.Printf("Module: %s\n", file.Module.Mod.Path)
	fmt.Printf("Go version: %s\n", file.Go.Version)
	fmt.Printf("Module cache location: %s\n\n", moduleCachePath)

	// Print all dependencies with their installation paths
	fmt.Println("Dependencies and their installation paths:")
	fmt.Println("==========================================")

	for _, req := range file.Require {
		installPath := getPackageInstallPath(req.Mod.Path, req.Mod.Version, moduleCachePath)
		exists := checkPackageExists(installPath)

		status := "INSTALLED"
		if !exists {
			status = "NOT FOUND"
		}

		indirectStr := ""
		if req.Indirect {
			indirectStr = " (indirect)"
		}

		fmt.Printf("- %s %s%s\n", req.Mod.Path, req.Mod.Version, indirectStr)
		fmt.Printf("  Location: %s\n", installPath)
		fmt.Printf("  Status: %s\n\n", status)
	}

	// Print replace directives if any
	if len(file.Replace) > 0 {
		fmt.Println("\nReplace Directives:")
		fmt.Println("===================")
		for _, rep := range file.Replace {
			fmt.Printf("- %s %s => %s %s\n",
				rep.Old.Path, rep.Old.Version,
				rep.New.Path, rep.New.Version)

			// For replaced modules, show the replacement location
			if rep.New.Version == "" {
				// Local replacement (filesystem path)
				fmt.Printf("  Location: %s (local filesystem)\n\n", rep.New.Path)
			} else {
				// Module replacement
				replacementPath := getPackageInstallPath(rep.New.Path, rep.New.Version, moduleCachePath)
				exists := checkPackageExists(replacementPath)
				status := "INSTALLED"
				if !exists {
					status = "NOT FOUND"
				}
				fmt.Printf("  Location: %s\n", replacementPath)
				fmt.Printf("  Status: %s\n\n", status)
			}
		}
	}
}
