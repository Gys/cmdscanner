# cmdscanner - Go Module Dependency Scanner

A command-line tool that parses Go module files (`go.mod`) and provides detailed information about dependencies, including their installation locations on disk.

## Features

- Parse `go.mod` files to extract module information
- List all direct and indirect dependencies
- Show the exact filesystem location where each dependency is installed
- Verify if dependencies are actually present on disk
- Handle replace directives and show replacement locations
- Support for both modern Go module cache and older GOPATH structures

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

Dependencies and their installation paths:
==========================================
- golang.org/x/mod v0.12.0
  Location: /home/user/go/pkg/mod/golang.org/x/mod@v0.12.0
  Status: INSTALLED

- github.com/stretchr/testify v1.8.4 (indirect)
  Location: /home/user/go/pkg/mod/github.com/stretchr/testify@v1.8.4
  Status: INSTALLED

Replace Directives:
===================
- golang.org/x/crypto v0.0.0 => golang.org/x/crypto v0.12.0
  Location: /home/user/go/pkg/mod/golang.org/x/crypto@v0.12.0
  Status: INSTALLED

- github.com/local/package v0.0.0 => ./local/package
  Location: ./local/package (local filesystem)
```

## Technical Notes

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

### Vendoring

If your project uses vendoring (`go mod vendor`), dependencies are also copied to the `vendor` directory in your project. This tool focuses on the module cache, not vendored dependencies.

### Proxy Settings

Go may use a proxy for downloading modules (controlled by the `GOPROXY` environment variable). This doesn't affect where packages are stored locally, but it affects how they're downloaded.

### Sum Database

Go maintains a checksum database (`go.sum` file) to verify the integrity of modules. This is separate from the actual module storage.

## License

[MIT License](LICENSE)
