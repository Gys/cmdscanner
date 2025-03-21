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

	"github.com/fatih/color"

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

		// Only process .go files that are not test files
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
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

// isGoOfficialPackage checks if a package is from the Go project itself
func isGoOfficialPackage(packagePath string) bool {
	return strings.HasPrefix(packagePath, "golang.org/") || strings.HasPrefix(packagePath, "google.golang.org/")
}

// shouldSkipPackage checks if a package should be skipped based on user-defined patterns
func shouldSkipPackage(packagePath string, skipPackages []string) bool {
	for _, skipPattern := range skipPackages {
		if strings.Contains(packagePath, skipPattern) {
			return true
		}
	}
	return false
}

// findGoModInParentDirs searches for a go.mod file in the current directory
// and all parent directories, returning the path to the first one found.
// If no go.mod file is found, it returns an empty string.
func findGoModInParentDirs() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return goModPath
		}

		// Move to parent directory
		parentDir := filepath.Dir(dir)
		// If we've reached the root directory and haven't found go.mod
		if parentDir == dir {
			break
		}
		dir = parentDir
	}

	return ""
}

func main() {
	// Define command-line flags
	goModPath := flag.String("file", "go.mod", "Path to the go.mod file to parse")
	includeGoOfficial := flag.Bool("include-go-official", false, "Include packages from *.golang.org")
	skipPackagesFlag := flag.String("skip", "", "Comma-separated list of packages to skip scanning")
	noColor := flag.Bool("no-color", false, "Disable color output")
	flag.Parse()

	// Apply the setting
	color.NoColor = *noColor

	// Parse the skip packages flag
	skipPackages := []string{}
	if *skipPackagesFlag != "" {
		skipPackages = strings.Split(*skipPackagesFlag, ",")
		for i, pkg := range skipPackages {
			skipPackages[i] = strings.TrimSpace(pkg)
		}
	}

	// Check if the file exists
	if _, err := os.Stat(*goModPath); os.IsNotExist(err) {
		// Try to find go.mod in parent directories
		if foundGoMod := findGoModInParentDirs(); foundGoMod != "" {
			*goModPath = foundGoMod
			fmt.Printf("Found go.mod in parent directory: %s\n", *goModPath)
		} else {
			log.Fatalf("Error: go.mod file not found at %s or in any parent directory", *goModPath)
		}
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
	fmt.Printf("Searching for command patterns: %s\n", strings.Join(CommandPatterns, ", "))
	fmt.Printf("Skipping test files (*_test.go)\n")
	if *includeGoOfficial {
		fmt.Printf("Including official Go packages (*.golang.org/*)\n")
	} else {
		fmt.Printf("Skipping official Go packages (*.golang.org/*)\n")
	}
	if len(skipPackages) > 0 {
		fmt.Printf("Skipping user-specified packages: %s\n", strings.Join(skipPackages, ", "))
	}
	fmt.Println()

	// Store all files containing command patterns
	var allMatches []FileMatch

	// Process all dependencies
	for _, req := range file.Require {
		// Skip Go official packages if requested
		if !*includeGoOfficial && isGoOfficialPackage(req.Mod.Path) {
			// fmt.Printf("- %s %s (skipped - Go official package)\n\n", req.Mod.Path, req.Mod.Version)
			continue
		}

		// Skip user-specified packages
		if shouldSkipPackage(req.Mod.Path, skipPackages) {
			// fmt.Printf("- %s %s (skipped - user-specified)\n\n", req.Mod.Path, req.Mod.Version)
			continue
		}

		installPath := getPackageInstallPath(req.Mod.Path, req.Mod.Version, moduleCachePath)
		exists := checkPackageExists(installPath)

		indirectStr := ""
		if req.Indirect {
			indirectStr = " (indirect)"
		}

		if !exists {
			fmt.Printf("- %s %s%s\n", req.Mod.Path, req.Mod.Version, indirectStr)
			fmt.Printf("  Location not found (%s)\n\n", installPath)
			continue
		}

		// Find all .go files containing command patterns
		matches, err := findCommandPatternsInGoFiles(installPath, CommandPatterns)
		if err != nil {
			fmt.Printf("- %s %s%s\n", req.Mod.Path, req.Mod.Version, indirectStr)
			fmt.Printf("  Error scanning: %v\n\n", err)
		} else {
			allMatches = append(allMatches, matches...)
		}
		// fmt.Println()
	}

	// Process replace directives
	if len(file.Replace) > 0 {
		for _, rep := range file.Replace {
			// Skip user-specified packages
			if shouldSkipPackage(rep.Old.Path, skipPackages) || shouldSkipPackage(rep.New.Path, skipPackages) {
				// fmt.Printf("- %s %s => %s %s (skipped - user-specified)\n\n", rep.Old.Path, rep.Old.Version, rep.New.Path, rep.New.Version)
				continue
			}

			var replacementPath string
			// var isLocalPath bool

			if rep.New.Version == "" {
				// Local replacement (filesystem path)
				replacementPath = rep.New.Path
				// isLocalPath = true
			} else {
				// Module replacement
				replacementPath = getPackageInstallPath(rep.New.Path, rep.New.Version, moduleCachePath)
			}

			exists := checkPackageExists(replacementPath)
			if !exists {
				fmt.Printf("- %s %s => %s %s\n", rep.Old.Path, rep.Old.Version, rep.New.Path, rep.New.Version)
				fmt.Printf("  Location not found (%s)\n\n", replacementPath)
				continue
			}

			// if isLocalPath {
			// 	fmt.Printf("  Location: %s (local filesystem)\n", replacementPath)
			// } else {
			// 	fmt.Printf("  Location: %s\n", replacementPath)
			// }

			// Find all .go files containing command patterns
			matches, err := findCommandPatternsInGoFiles(replacementPath, CommandPatterns)
			if err != nil {
				fmt.Printf("- %s %s => %s %s\n", rep.Old.Path, rep.Old.Version, rep.New.Path, rep.New.Version)
				fmt.Printf("  Error scanning: %v\n\n", err)
			} else {
				allMatches = append(allMatches, matches...)
			}
			// fmt.Println()
		}
	}

	// Print detailed summary of all files containing command patterns
	fmt.Printf("Results:\n\n")
	if len(allMatches) == 0 {
		fmt.Printf("No command patterns found in any files.\n\n")
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

		// Print detailed file matches
		style := color.New(color.FgHiYellow)
		for _, fileMatch := range allMatches {
			for _, line := range fileMatch.Lines {
				fmt.Printf("%s:%d\n", fileMatch.FilePath, line.LineNumber)
				style.Printf("%s\n", strings.TrimSpace(line.Content))
			}
			fmt.Println()
		}
	}
}
