# cmdscanner - Go Module Dependency Scanner

A command-line tool that parses Go module files (`go.mod`) and scans all dependencies for command pattern usage. It identifies and extracts code lines containing specific command patterns like `.Command(`, `.RunCommand(`, and `.Cmd(`.

## Features

- Parse `go.mod` files to extract module information
- List all direct and indirect dependencies
- Show the exact filesystem location where each dependency is installed
- Verify if dependencies are actually present on disk
- Handle replace directives and show replacement locations
- **Scan all Go files in dependencies for specific command patterns**:
 - `.Command(`
 - `.RunCommand(`
 - `.Cmd(`
- **Extract and display the actual lines of code containing these patterns**
- Generate a detailed summary of all matching files and code lines
- Group and count occurrences by pattern type

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/cmdscanner
cd cmdscanner

# Build the application
go build -o cmdscanner
```

## Usage

```bash
# Scan the go.mod file in the current directory
./cmdscanner

# Scan a specific go.mod file
./cmdscanner -file /path/to/go.mod
```

## Example Output

```
Module: github.com/example/myproject
Go version: 1.20
Module cache location: /home/user/go/pkg/mod
Searching for command patterns: .Command(, .RunCommand(, .Cmd(

Scanning dependencies for command patterns in Go files:
======================================================
- github.com/spf13/cobra v1.7.0
 Location: /home/user/go/pkg/mod/github.com/spf13/cobra@v1.7.0
 Scanning for command patterns in Go files...
 Found 3 files with 12 command pattern occurrences

- github.com/urfave/cli/v2 v2.25.7 (indirect)
 Location: /home/user/go/pkg/mod/github.com/urfave/cli/v2@v2.25.7
 Scanning for command patterns in Go files...
 Found 2 files with 5 command pattern occurrences

Detailed summary of all Go files containing command patterns:
===========================================================
Found 17 command pattern occurrences in 5 files:

Pattern summary:
- .Command(: 10 occurrences
- .Cmd(: 5 occurrences
- .RunCommand(: 2 occurrences

1. /home/user/go/pkg/mod/github.com/spf13/cobra@v1.7.0/command.go (8 occurrences)
  Line 142 [.Command(]: rootCmd.Command("version", "Print the version number")
  Line 157 [.Command(]: cmd := &Command{Use: "app"}
  Line 203 [.Cmd(]: app.Cmd("init", "Initialize the application")
  Line 245 [.RunCommand(]: err := cli.RunCommand(args)
  Line 301 [.Command(]: subCmd := cmd.Command("serve", "Start the server")
  Line 350 [.Command(]: helpCmd := rootCmd.Command("help", "Help about any command")
  Line 412 [.Cmd(]: return app.Cmd("config", "Manage configuration")
  Line 489 [.RunCommand(]: return cmd.RunCommand(context.Background())

2. /home/user/go/pkg/mod/github.com/spf13/cobra@v1.7.0/cobra.go (4 occurrences)
  Line 78 [.Command(]: NewCommand returns a new Command
  Line 120 [.Cmd(]: c := root.Cmd("status", "Show status")
  Line 156 [.Command(]: cmd := cobra.Command("serve", "Start the server")
  Line 203 [.Cmd(]: return app.Cmd("version", "Print version information")
```

## Technical Notes

### Command Pattern Detection

The tool specifically looks for these command patterns:
- `.Command(` - Common in command-line libraries like Cobra, Cli, etc.
- `.RunCommand(` - Used for executing commands
- `.Cmd(` - A shorter variant used in many libraries

These patterns help identify where and how commands are defined and used in Go code, which is particularly useful for understanding command-line applications and their structure.

### Go Module Cache

Go stores downloaded packages in a central module cache. The location of this cache is determined by:

1. The `GOMODCACHE` environment variable (Go 1.14+)
2. Fallback: `$GOPATH/pkg/mod` (older Go versions)

You can find your module cache location by running:

```bash
go env GOMODCACHE
```

### Module Path Encoding

Module paths in the filesystem are encoded to handle special characters:

- Uppercase letters are converted to '!' followed by the lowercase letter
- The '!' character is converted to '!!'
- Other special characters may also be encoded

Our tool handles this encoding automatically using Go's `module.EscapePath` function.

### Version Handling

Module versions in the filesystem may have special suffixes:

- `+incompatible` suffix for modules that don't follow semantic versioning
- Pseudo-versions like `v0.0.0-20230822171919-f59e071c2a15`

The tool handles these special cases when determining installation paths.

### Replace Directives

Replace directives in `go.mod` files can point to:

1. Another module version (stored in the module cache)
2. A local filesystem path (not in the module cache)

The tool distinguishes between these cases and shows the appropriate location.

### File Scanning

The tool recursively scans all `.go` files in each dependency's directory:

- It skips directories like `.git`, `testdata`, and `vendor`
- It searches for the specified command patterns in each Go file
- It extracts and displays the actual lines of code containing these patterns
- It provides a detailed summary of all matching files and code lines at the end
- It groups and counts occurrences by pattern type

### Extending the Command Patterns

To add more command patterns to search for, modify the `CommandPatterns` slice in the code:

```go
var CommandPatterns = []string{
   `.Command(`,
   `.RunCommand(`,
   `.Cmd(`,
   // Add more patterns here
   `.NewCommand(`,
   `.AddCommand(`,
}
```

### Vendoring

If your project uses vendoring (`go mod vendor`), dependencies are also copied to the `vendor` directory in your project. This tool focuses on the module cache, not vendored dependencies.

### Proxy Settings

Go may use a proxy for downloading modules (controlled by the `GOPROXY` environment variable). This doesn't affect where packages are stored locally, but it affects how they're downloaded.

### Sum Database

Go maintains a checksum database (`go.sum` file) to verify the integrity of modules. This is separate from the actual module storage.

## License

[MIT License](LICENSE)
