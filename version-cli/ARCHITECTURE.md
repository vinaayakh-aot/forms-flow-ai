# FormsFlow.ai Version Update CLI - Architecture

## Overview

The version update CLI tool has been refactored into a clean, modular architecture following separation of concerns principles. The codebase is split into four main components.

## File Structure

```
version-cli/
├── main.go        # Entry point and CLI setup
├── core.go        # Business logic and version updating
├── presenter.go   # UI/TUI presentation layer 
├── utils.go       # Helper functions and utilities
├── config-simple.json  # Configuration file
├── Makefile       # Build and deployment scripts
└── go.mod         # Go module dependencies
```

## Component Responsibilities

### 1. Entry Point (`main.go`)
**Purpose**: Application bootstrap and command-line interface setup

**Contains**:
- `main()` function
- Cobra CLI command setup and configuration
- Command-line flag definitions
- Help and version templates
- Error handling for the application entry

**Dependencies**: 
- `presenter.go` (for TUI models)
- `core.go` (for business logic)

### 2. Core Logic (`core.go`)
**Purpose**: Business logic for version processing and file updates

**Contains**:
- `VersionUpdater` struct and methods
- Configuration loading and parsing
- Version file reading and component extraction
- File processing and content modification
- Pattern expansion and matching logic

**Key Types**:
- `Config`, `FileConfig`, `UpdateConfig`, `ContextConfig`
- `VersionUpdater`

**Dependencies**:
- `utils.go` (for helper functions)

### 3. Presenter (`presenter.go`)
**Purpose**: User interface and visual presentation

**Contains**:
- Bubble Tea TUI models and views
- Progress indicators and spinners
- Styled output with lipgloss
- File status tracking and display
- Final summary reporting

**Key Types**:
- `model` (Bubble Tea model)
- `FileStatus`
- Message types for TUI state management

**Dependencies**:
- `core.go` (for business logic integration)

### 4. Utilities (`utils.go`)
**Purpose**: Helper functions and common operations

**Contains**:
- Regex pattern matching utilities
- Text processing functions
- Pattern filtering and exclusions
- Context-aware matching
- Predefined regex pattern definitions

**Key Types**:
- `Match` struct
- Pattern manipulation functions

## Data Flow

```
1. main.go
   ├── Parses CLI arguments
   ├── Creates VersionUpdater (core.go)
   └── Launches TUI (presenter.go)

2. core.go
   ├── Loads configuration from JSON
   ├── Reads version from file
   ├── Processes each configured file
   └── Uses utils.go for pattern matching

3. presenter.go
   ├── Displays progress and status
   ├── Coordinates with core.go for processing
   ├── Shows real-time file updates
   └── Generates final summary report

4. utils.go
   ├── Provides pattern matching support
   ├── Handles text filtering and exclusions
   └── Manages regex operations
```

## Benefits of This Architecture

### 🎯 **Separation of Concerns**
- Each file has a single, well-defined responsibility
- Business logic is isolated from presentation
- Utilities are reusable across components

### 🔧 **Maintainability**
- Easy to locate and modify specific functionality
- Clear boundaries between components
- Reduced coupling between modules

### 🧪 **Testability**
- Components can be tested in isolation
- Business logic separated from UI concerns
- Utils can be unit tested independently

### 📈 **Scalability**
- Easy to add new features to specific components
- TUI can be enhanced without affecting core logic
- Core logic can be extended without UI changes

## Build Process

The refactored architecture uses Go's package system:

```bash
# Build all components together
make build

# The Makefile uses "." to build entire package
GO_FILES = .
```

This ensures all `.go` files in the package are compiled together, maintaining proper dependencies and cross-references between components.

## Design Patterns Used

- **MVC Pattern**: Clear separation between Model (core), View (presenter), and Controller (main)
- **Command Pattern**: CLI commands encapsulate operations
- **Observer Pattern**: TUI observes and displays processing state
- **Utility Pattern**: Common functions centralized in utils

This architecture provides a solid foundation for future enhancements while maintaining code quality and developer productivity. 