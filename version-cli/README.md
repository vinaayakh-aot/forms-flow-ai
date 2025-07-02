# FormsFlow.ai Version Update Automation

This directory contains automated tooling to update version references across the entire FormsFlow.ai repository.

## Overview

The version update system reads the version from the root `VERSION` file and propagates it across all configured files in the repository. This ensures consistency and reduces manual errors when releasing new versions.

## Quick Start

### Preview Changes (Recommended)
```bash
# See what would be changed without modifying files
python version-cli/update-versions.py --dry-run
```

### Apply Updates
```bash
# Update all files with the current version from VERSION file
python version-cli/update-versions.py
```

### Using Custom Configuration
```bash
# Use a different configuration file
python version-cli/update-versions.py --config path/to/custom-config.json
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

## Version History

- **v1.0**: Initial version with support for Docker, NPM, and URL patterns
- Support for regex and string replacement patterns
- Configurable file and pattern definitions
- Dry-run mode for safe testing 