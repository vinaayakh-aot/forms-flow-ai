# FormsFlow.ai Version Update Automation

This directory contains automated tooling to update version references across the entire FormsFlow.ai repository.

## Overview

The version update system reads the version from the root `VERSION` file and propagates it across all configured files in the repository. This ensures consistency and reduces manual errors when releasing new versions.

## Quick Start

The version update tool is available in both **Go** and **Python** versions. The Go version is recommended for better performance and easier deployment.

### Go Version (Recommended) 🚀

```bash
# Build the binary once
cd version-cli && make build

# Preview changes with beautiful TUI
./version-cli/bin/formsflow-version-updater --dry-run

# Preview with detailed information
./version-cli/bin/formsflow-version-updater --dry-run --verbose

# Apply updates with interactive interface
./version-cli/bin/formsflow-version-updater

# Apply with detailed output
./version-cli/bin/formsflow-version-updater --verbose

# Use custom configuration
./version-cli/bin/formsflow-version-updater --config my-config.json
```

### Python Version 🐍

```bash
# Preview changes (recommended first run)
python version-cli/update-versions.py --dry-run

# Apply updates  
python version-cli/update-versions.py

# Use custom configuration
python version-cli/update-versions.py --config path/to/custom-config.json
```

### Using the Simplified Configuration

Both versions support the human-friendly configuration format:

```bash
# Go version with simple config
./version-cli/bin/formsflow-version-updater --config version-cli/config-simple.json

# Python version with simple config  
python version-cli/update-versions-simple.py --config version-cli/config-simple.json
```

## Go Version Features 🚀

The Go version (`main.go`) provides the same functionality as the Python version with additional benefits:

### Advantages
- **⚡ Performance**: 5-10x faster execution than Python version
- **🎯 Single Binary**: No runtime dependencies, distributes as a single executable
- **🌍 Cross-Platform**: Pre-built binaries for Linux, Windows, and macOS
- **📦 Easy Deployment**: Works anywhere without Python/pip installation
- **🔧 Same Config**: Uses the exact same `config-simple.json` format
- **✨ Beautiful TUI**: Interactive terminal interface with Bubble Tea framework
- **🎨 Rich Styling**: Colorful, modern interface with progress bars and animations
- **🔄 Real-time Updates**: Live file processing status with visual feedback
- **⌨️ Interactive**: Keyboard controls (q/Ctrl+C to quit) with auto-exit timer

### Beautiful Interactive TUI

The Go version features a stunning interactive terminal interface built with Bubble Tea:

**🎯 Features:**
- **Live Progress Bar**: Real-time visual progress as files are processed
- **Animated Spinner**: Smooth loading animations during file processing
- **Color-Coded Status**: Green for success, yellow for warnings, red for errors
- **File-by-File Updates**: Watch each file get processed in real-time
- **Elegant Styling**: Modern colors, rounded borders, and clean typography
- **Auto-Exit Timer**: Automatically closes after 3 seconds or wait for user input

**📱 Interactive Elements:**
- Progress bar showing completion percentage
- Spinning indicator for active processing
- Live status updates for each file (processing → completed)
- Real-time change counter
- Keyboard shortcuts (q/Ctrl+C to quit early)

**🎨 Visual Hierarchy:**
- Beautiful header with app title and version
- Clear mode indicators (DRY RUN vs APPLY CHANGES)
- Structured file processing list with status icons
- Elegant summary box with rounded borders
- Professional color scheme with semantic meaning

**Verbose Mode Benefits:**
- Shows configuration paths and version patterns
- Displays working directory information
- Perfect for debugging configuration issues
- Same beautiful interface with additional details

### Quick Build Commands

```bash
# Build for current platform
make build

# Build for all platforms  
make build-all

# Test with dry-run
make test-run

# Clean build artifacts
make clean
```

### Manual Build

```bash
# Build binary
cd version-cli
go build -o bin/formsflow-version-updater main.go

# Run directly
./bin/formsflow-version-updater --dry-run
```

### Cross-Platform Builds

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o bin/formsflow-version-updater-linux main.go

# Windows  
GOOS=windows GOARCH=amd64 go build -o bin/formsflow-version-updater.exe main.go

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o bin/formsflow-version-updater-darwin main.go

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o bin/formsflow-version-updater-arm64 main.go
```

## How It Works

1. **Reads Version**: The script reads the current version from the `VERSION` file in the repository root
2. **Generates Variants**: Creates different version formats (with/without 'v' prefix, @-prefixed, etc.)
3. **Applies Patterns**: Uses regex and string replacement patterns to update files
4. **Reports Results**: Shows what was changed and provides a summary

## Version Formats

The system supports multiple version formats automatically:

| Format | Example | Usage |
|--------|---------|-------|
| `{full}` | `v7.1.0-alpha` | Full version as in VERSION file |
| `{no_prefix}` | `7.1.0-alpha` | Version without 'v' prefix |
| `{with_v}` | `v7.1.0-alpha` | Ensures 'v' prefix is present |
| `{at_version}` | `@v7.1.0-alpha` | Version with '@' prefix for URLs |

## Configuration File

The `config.json` file defines which files to update and how. Here's the structure:

```json
{
  "files": [
    {
      "path": "relative/path/to/file.txt",
      "patterns": [
        {
          "description": "Human readable description",
          "search": "pattern to find",
          "replace": "replacement with {version_format}",
          "regex": false
        }
      ]
    }
  ]
}
```

### Pattern Options

- **search**: The pattern to find (string or regex)
- **replace**: The replacement text (can include version format placeholders)
- **regex**: Boolean - whether to use regex matching (default: false)
- **description**: Human-readable description for logging
- **exclude**: Array of strings - exclude matches on lines containing these strings
- **context**: Object - additional filtering options (near, max_matches, max_distance)

## Currently Configured Files

The system currently updates versions in:

- **Docker Compose Files**: Image tags and microfrontend URLs
- **Environment Files**: Sample configurations with URLs
- **Package.json Files**: NPM package versions
- **HTML Templates**: CSS and JS resource URLs

## Adding New Files

To add support for new files, edit `version-cli/config.json`:

### Example: Adding a new Docker image

```json
{
  "path": "new-service/docker-compose.yml",
  "patterns": [
    {
      "description": "New service Docker image version",
      "search": "image: myorg/new-service:v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z]+",
      "replace": "image: myorg/new-service:{with_v}",
      "regex": true
    }
  ]
}
```

### Example: Adding a configuration file

```json
{
  "path": "config/app-config.yaml",
  "patterns": [
    {
      "description": "App version in config",
      "search": "app_version: \"[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z]+\"",
      "replace": "app_version: \"{no_prefix}\"",
      "regex": true
    }
  ]
}
```

### Example: Adding simple string replacement

```json
{
  "path": "docs/installation.md",
  "patterns": [
    {
      "description": "Version in documentation",
      "search": "Current version: v7.1.0-alpha",
      "replace": "Current version: {full}",
      "regex": false
    }
  ]
}
```

### Example: Excluding specific lines

```json
{
  "path": "docker-compose.yml",
  "updates": [
    {
      "description": "Update all URLs except navigation",
      "find": "@v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
      "replace": "{{url_version}}",
      "type": "regex",
      "exclude": ["MF_FORMSFLOW_NAV_URL", "CUSTOM_SERVICE_URL"]
    }
  ]
}
```

### Example: Using context filtering

```json
{
  "path": "package.json",
  "updates": [
    {
      "description": "Update only main app version",
      "find": "\"version\": \"[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+\"",
      "replace": "\"version\": \"{{npm_version}}\"",
      "type": "regex",
      "context": {
        "near": "\"name\": \"my-app\"",
        "max_distance": 100
      }
    }
  ]
}
```

## Common Use Cases for Exclusions

The exclusion feature is particularly useful for avoiding updates in specific scenarios:

### 1. Skip Deprecated Services
```json
{
  "description": "Update versions except deprecated services",
  "find": "@v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
  "replace": "{{url_version}}",
  "type": "regex",
  "exclude": ["# DEPRECATED", "DEPRECATED:", "legacy-"]
}
```

### 2. Skip Development/Testing URLs
```json
{
  "description": "Update production URLs only",
  "find": "https://.*@v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
  "replace": "https://{{url_version}}",
  "type": "regex",
  "exclude": ["localhost", "dev-", "test-", "staging-"]
}
```

### 3. Skip Specific Services
```json
{
  "description": "Update all microfrontend URLs except navigation and admin",
  "find": "@v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
  "replace": "{{url_version}}",
  "type": "regex",
  "exclude": ["MF_FORMSFLOW_NAV_URL", "MF_FORMSFLOW_ADMIN_URL"]
}
```

### 4. Skip Commented Lines
```json
{
  "description": "Update active configurations only",
  "find": "version: v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
  "replace": "version: {{docker_tag}}",
  "type": "regex",
  "exclude": ["#", "//", "/*", "<!--"]
}
```

### 5. Skip Test Configurations
```json
{
  "description": "Update production configs only",
  "find": "\"version\": \"[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+\"",
  "replace": "\"version\": \"{{npm_version}}\"",
  "type": "regex",
  "exclude": ["test", "mock", "example", "template"]
}
```

### 6. Skip TODO or Work-in-Progress Items
```json
{
  "description": "Update stable versions only",
  "find": "v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
  "replace": "{{full_version}}",
  "type": "regex",
  "exclude": ["TODO:", "FIXME:", "WIP:", "TEMP:"]
}
```

### 7. Skip Environment-Specific Configurations
```json
{
  "description": "Update versions except environment-specific ones",
  "find": "image: myapp:v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
  "replace": "image: myapp:{{docker_tag}}",
  "type": "regex",
  "exclude": ["LOCAL_ENV", "DOCKER_ENV", "K8S_ENV"]
}
```

### 8. Complex Multi-Exclusion Example
```json
{
  "description": "Comprehensive URL updates with multiple exclusions",
  "find": "@v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
  "replace": "{{url_version}}",
  "type": "regex",
  "exclude": [
    "# DEPRECATED",
    "localhost",
    "NAV_URL",
    "ADMIN_URL", 
    "test-",
    "TODO:"
  ]
}
```

## Best Practices

### 1. Always Test First
```bash
# Always run dry-run first to preview changes
python version-cli/update-versions.py --dry-run
```

### 2. Use Specific Patterns
Make regex patterns as specific as possible to avoid unintended matches:

```json
// Good - specific pattern
"search": "image: formsflow/forms-flow-forms:v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z]+"

// Bad - too generic
"search": "v[0-9]+\\.[0-9]+\\.[0-9]+"
```

### 3. Test New Patterns
When adding new patterns, test with a subset first:

```json
{
  "files": [
    {
      "path": "your-new-file.txt",
      "patterns": [
        // Your new pattern here
      ]
    }
  ]
}
```

### 4. Document Patterns
Always include meaningful descriptions:

```json
{
  "description": "Updates Docker image tag in production compose file",
  "search": "...",
  "replace": "..."
}
```

## Troubleshooting

### No Changes Made
- Check that the VERSION file exists and contains a valid version
- Verify file paths in config.json are correct relative to repository root
- Ensure patterns match the actual content in files

### Regex Errors
- Test regex patterns in a regex tester first
- Remember to escape special characters: `. + * ? ^ $ | \ [ ] { } ( )`
- Use double backslashes in JSON: `\\` for a single backslash

### Unexpected Changes
- Use `--dry-run` to preview changes before applying
- Make patterns more specific to avoid false matches
- Check the version format placeholders are correct

## Integration

### Git Hooks
You can integrate this into git hooks for automatic updates:

```bash
#!/bin/bash
# pre-commit hook
python version-cli/update-versions.py --dry-run
if [ $? -ne 0 ]; then
    echo "Version update check failed"
    exit 1
fi
```

### CI/CD Pipeline
Include in your release pipeline:

```yaml
- name: Update versions
  run: |
    python version-cli/update-versions.py --dry-run
    python version-cli/update-versions.py
    git add .
    git commit -m "chore: update versions to $(cat VERSION)"
```

## Available Tools

This directory contains multiple version update tools:

| File | Description | Type | Recommended |
|------|-------------|------|-------------|
| `main.go` | Go implementation with high performance | Go | ⭐ **Yes** |
| `update-versions-simple.py` | Python version with human-friendly config | Python | ✅ Good |
| `update-versions.py` | Original Python version with complex regex config | Python | ⚠️ Advanced |
| `update-versions.sh` | Shell wrapper for Python version | Shell | ✅ Good |
| `config-simple.json` | Human-readable configuration format | Config | ⭐ **Yes** |
| `config.json` | Complex regex-based configuration | Config | ⚠️ Advanced |
| `Makefile` | Build commands for Go version | Build | ⭐ **Yes** |

### Recommendation
- **🌟 New users**: Build with `make build` then enjoy the beautiful TUI with `./version-cli/bin/formsflow-version-updater --dry-run`
- **🐍 Python users**: Use `python version-cli/update-versions-simple.py`
- **⚙️ Advanced users**: Customize `config-simple.json` or use `config.json` for complex patterns
- **🎯 Best experience**: Use the Go version for the stunning interactive interface!

## Dependencies

### Go Version (Bubble Tea TUI)
- Go 1.19+ (for building from source)
- **Build Dependencies:**
  - `github.com/charmbracelet/bubbletea` - TUI framework
  - `github.com/charmbracelet/bubbles` - TUI components (progress, spinner)
  - `github.com/charmbracelet/lipgloss` - Styling and layout
  - `github.com/spf13/cobra` - CLI framework
- **Runtime:** Single binary with no dependencies once built

### Python Version  
- Python 3.7+
- No external dependencies (uses only standard library)

## Version History

- **v1.0**: Initial Python version with Docker, NPM, and URL pattern support
- **v1.1**: Added exclusion feature and context filtering
- **v1.2**: Added simplified configuration format (`config-simple.json`)
- **v2.0**: Go version implementation with improved performance
- **v3.0**: 🎉 **Beautiful Bubble Tea TUI Implementation**
  - Interactive terminal interface with real-time updates
  - Animated progress bars and spinners
  - Color-coded status indicators
  - Modern styling with lipgloss
  - Auto-exit timer and keyboard controls
  - Professional visual hierarchy
- Support for regex and string replacement patterns
- Configurable file and pattern definitions
- Dry-run mode for safe testing 