package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// FileMatch represents a match of the search string in a file
type FileMatch struct {
	FilePath string
	Lines    []LineMatch
}

// LineMatch represents a single line match with line number and content
type LineMatch struct {
	LineNumber int
	Content    string
	Pattern    string // Which pattern matched
}

// CommandPatterns defines the specific command patterns we're looking for
var CommandPatterns = []string{
	`.Command(`,
	`.RunCommand(`,
	`.Cmd(`,
}

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

	cleanVersion := strings.TrimSuffix(version, "+incompatible")

	// Construct the path
	return filepath.Join(moduleCachePath, encodedPath+"@"+cleanVersion)
}

// checkPackageExists verifies if the package is installed at the expected location
func checkPackageExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// findCommandPatternsInGoFiles searches for command patterns in all .go files in the given directory and subdirectories
func findCommandPatternsInGoFiles(rootPath string, patterns []string) ([]FileMatch, error) {
	var matches []FileMatch

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't access
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories like .git, testdata, etc.
		if d.IsDir() {
			dirName := filepath.Base(path)
			if strings.HasPrefix(dirName, ".") || dirName == "testdata" || dirName == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		// Open and scan the file
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		var lineMatches []LineMatch

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Check each pattern
			for _, pattern := range patterns {
				if strings.Contains(line, pattern) {
					// Trim trailing whitespace but preserve indentation
					trimmedLine := strings.TrimRight(line, " \t\r\n")
					lineMatches = append(lineMatches, LineMatch{
						LineNumber: lineNum,
						Content:    trimmedLine,
						Pattern:    pattern,
					})
					// We found a match with this pattern, no need to check other patterns for this line
					break
				}
			}
		}

		if len(lineMatches) > 0 {
			matches = append(matches, FileMatch{
				FilePath: path,
				Lines:    lineMatches,
			})
		}

		return scanner.Err()
	})

	return matches, err
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
	fmt.Printf("Module cache location: %s\n", moduleCachePath)
	fmt.Printf("Searching for command patterns: %s\n\n", strings.Join(CommandPatterns, ", "))

	// Store all files containing command patterns
	var allMatches []FileMatch

	// Process all dependencies
	fmt.Println("Scanning dependencies for command patterns in Go files:")
	fmt.Println("======================================================")
	for _, req := range file.Require {
		installPath := getPackageInstallPath(req.Mod.Path, req.Mod.Version, moduleCachePath)
		exists := checkPackageExists(installPath)
		indirectStr := ""
		if req.Indirect {
			indirectStr = " (indirect)"
		}
		fmt.Printf("- %s %s%s\n", req.Mod.Path, req.Mod.Version, indirectStr)
		if !exists {
			fmt.Printf("  Status: NOT FOUND - Skipping scan\n\n")
			continue
		}
		fmt.Printf("  Location: %s\n", installPath)

		fmt.Printf("  Scanning for command patterns in Go files...\n")

		// Find all .go files containing command patterns
		matches, err := findCommandPatternsInGoFiles(installPath, CommandPatterns)
		if err != nil {
			fmt.Printf("  Error scanning files: %v\n", err)
		} else if len(matches) == 0 {
			fmt.Printf("  No files containing command patterns found\n")
		} else {
			totalLines := 0
			for _, match := range matches {
				totalLines += len(match.Lines)
			}

			fmt.Printf("  Found %d files with %d command pattern occurrences\n",
				len(matches), totalLines)
			allMatches = append(allMatches, matches...)
		}
		fmt.Println()
	}

	// Process replace directives
	if len(file.Replace) > 0 {
		fmt.Println("\nScanning replaced modules:")
		fmt.Println("=========================")
		for _, rep := range file.Replace {

			fmt.Printf("- %s %s => %s %s\n",
				rep.Old.Path, rep.Old.Version,
				rep.New.Path, rep.New.Version)

			var replacementPath string
			var isLocalPath bool

			if rep.New.Version == "" {
				// Local replacement (filesystem path)
				replacementPath = rep.New.Path
				isLocalPath = true
			} else {
				// Module replacement
				replacementPath = getPackageInstallPath(rep.New.Path, rep.New.Version, moduleCachePath)
			}

			exists := checkPackageExists(replacementPath)
			if !exists {
				fmt.Printf("  Status: NOT FOUND - Skipping scan\n\n")
				continue
			}

			if isLocalPath {
				fmt.Printf("  Location: %s (local filesystem)\n", replacementPath)
			} else {
				fmt.Printf("  Location: %s\n", replacementPath)
			}

			fmt.Printf("  Scanning for command patterns in Go files...\n")

			// Find all .go files containing command patterns
			matches, err := findCommandPatternsInGoFiles(replacementPath, CommandPatterns)
			if err != nil {
				fmt.Printf("  Error scanning files: %v\n", err)
			} else if len(matches) == 0 {
				fmt.Printf("  No files containing command patterns found\n")
			} else {
				totalLines := 0
				for _, match := range matches {
					totalLines += len(match.Lines)
				}

				fmt.Printf("  Found %d files with %d command pattern occurrences\n",
					len(matches), totalLines)
				allMatches = append(allMatches, matches...)
			}
			fmt.Println()
		}
	}

	// Print detailed summary of all files containing command patterns
	fmt.Println("\nDetailed summary of all Go files containing command patterns:")
	fmt.Println("===========================================================")
	if len(allMatches) == 0 {
		fmt.Println("No files found containing command patterns.")
	} else {
		totalOccurrences := 0
		for _, fileMatch := range allMatches {
			totalOccurrences += len(fileMatch.Lines)
		}

		fmt.Printf("Found %d command pattern occurrences in %d files:\n\n", totalOccurrences, len(allMatches))

		// Group matches by pattern
		patternCounts := make(map[string]int)
		for _, fileMatch := range allMatches {
			for _, line := range fileMatch.Lines {
				patternCounts[line.Pattern]++
			}
		}

		// Print pattern summary
		fmt.Println("Pattern summary:")
		for pattern, count := range patternCounts {
			fmt.Printf("- %s: %d occurrences\n", pattern, count)
		}
		fmt.Println()

		// Print detailed file matches
		for i, fileMatch := range allMatches {
			fmt.Printf("%d. %s (%d occurrences)\n", i+1, fileMatch.FilePath, len(fileMatch.Lines))
			for _, line := range fileMatch.Lines {
				fmt.Printf("   Line %d [%s]: %s\n", line.LineNumber, line.Pattern, line.Content)
			}
			fmt.Println()
		}
	}
}
