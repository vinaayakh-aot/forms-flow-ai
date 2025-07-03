package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config represents the JSON configuration structure
type Config struct {
	VersionPatterns map[string]string `json:"version_patterns"`
	Files           []FileConfig      `json:"files"`
}

// FileConfig represents a file to be updated
type FileConfig struct {
	Name    string         `json:"name"`
	Path    string         `json:"path"`
	Updates []UpdateConfig `json:"updates"`
}

// UpdateConfig represents an update operation
type UpdateConfig struct {
	Description string            `json:"description"`
	Find        string            `json:"find"`
	Replace     string            `json:"replace"`
	Type        string            `json:"type"`
	Exclude     []string          `json:"exclude,omitempty"`
	Context     *ContextConfig    `json:"context,omitempty"`
}

// ContextConfig represents context filtering options
type ContextConfig struct {
	Near        string `json:"near"`
	MaxMatches  int    `json:"max_matches,omitempty"`
	MaxDistance int    `json:"max_distance,omitempty"`
}

// VersionUpdater handles the version update logic
type VersionUpdater struct {
	rootDir         string
	configPath      string
	versionFilePath string
	config          Config
	currentVersion  string
	dryRun          bool
}

// NewVersionUpdater creates a new version updater instance
func NewVersionUpdater(configPath string, versionFilePath string, dryRun bool) (*VersionUpdater, error) {
	// Get the directory containing the executable or current working directory
	rootDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Default config path if not provided
	if configPath == "" {
		configPath = filepath.Join(rootDir, "version-cli", "config-simple.json")
	}

	// Default version file path if not provided
	if versionFilePath == "" {
		versionFilePath = filepath.Join(rootDir, "VERSION")
	}

	updater := &VersionUpdater{
		rootDir:         rootDir,
		configPath:      configPath,
		versionFilePath: versionFilePath,
		dryRun:          dryRun,
	}

	// Load configuration
	if err := updater.loadConfig(); err != nil {
		return nil, err
	}

	// Read current version
	if err := updater.readVersion(); err != nil {
		return nil, err
	}

	return updater, nil
}

// loadConfig reads and parses the configuration file
func (v *VersionUpdater) loadConfig() error {
	data, err := ioutil.ReadFile(v.configPath)
	if err != nil {
		return fmt.Errorf("❌ Config file not found at %s: %w", v.configPath, err)
	}

	if err := json.Unmarshal(data, &v.config); err != nil {
		return fmt.Errorf("❌ Error parsing config file: %w", err)
	}

	return nil
}

// readVersion reads the current version from the specified version file
func (v *VersionUpdater) readVersion() error {
	data, err := ioutil.ReadFile(v.versionFilePath)
	if err != nil {
		return fmt.Errorf("❌ VERSION file not found at %s: %w", v.versionFilePath, err)
	}

	v.currentVersion = strings.TrimSpace(string(data))
	return nil
}

// getVersionComponents extracts version components for different formats
func (v *VersionUpdater) getVersionComponents() map[string]string {
	version := v.currentVersion
	versionNoV := strings.TrimPrefix(version, "v")
	versionWithV := version
	if !strings.HasPrefix(version, "v") {
		versionWithV = "v" + version
	}

	return map[string]string{
		"VERSION":        version,
		"VERSION_NO_V":   versionNoV,
		"VERSION_WITH_V": versionWithV,
	}
}

// expandVersionPatterns expands version patterns defined in config
func (v *VersionUpdater) expandVersionPatterns() map[string]string {
	patterns := make(map[string]string)
	versionComponents := v.getVersionComponents()

	for patternName, patternTemplate := range v.config.VersionPatterns {
		expanded := patternTemplate
		for component, value := range versionComponents {
			expanded = strings.ReplaceAll(expanded, "{"+component+"}", value)
		}
		patterns[patternName] = expanded
	}

	return patterns
}

// updateFile updates a single file using the simple configuration format
func (v *VersionUpdater) updateFile(fileConfig FileConfig) (int, error) {
	filePath := filepath.Join(v.rootDir, fileConfig.Path)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return 0, nil
	}

	// Read file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("error reading %s: %w", fileConfig.Path, err)
	}

	originalContent := string(content)
	updatedContent := originalContent
	totalChanges := 0
	versionPatterns := v.expandVersionPatterns()

	for _, update := range fileConfig.Updates {
		searchPattern, replaceTemplate := createPatternFromSimple(update, versionPatterns)

		// Find all matches
		allMatches, err := findMatches(updatedContent, searchPattern)
		if err != nil {
			continue
		}

		matches := allMatches

		// Apply exclusions if specified
		if len(update.Exclude) > 0 {
			matches = applyExclusions(updatedContent, matches, update.Exclude)
		}

		// Apply context filtering if specified
		if update.Context != nil {
			contextMatches, err := applyContextFiltering(updatedContent, searchPattern, update.Context)
			if err != nil {
				continue
			}

			// Intersect with exclusion-filtered matches
			contextMatchPositions := make(map[string]bool)
			for _, match := range contextMatches {
				key := fmt.Sprintf("%d-%d", match.Start, match.End)
				contextMatchPositions[key] = true
			}

			var filteredMatches []Match
			for _, match := range matches {
				key := fmt.Sprintf("%d-%d", match.Start, match.End)
				if contextMatchPositions[key] {
					filteredMatches = append(filteredMatches, match)
				}
			}
			matches = filteredMatches
		}

		// Check if matches actually need updating (aren't already the target version)
		var actualChanges int
		var matchesToUpdate []Match
		
		for _, match := range matches {
			regex := regexp.MustCompile(searchPattern)
			replacement := regex.ReplaceAllString(match.Text, replaceTemplate)
			
			// Only count as a change if the replacement is different from the original
			if replacement != match.Text {
				actualChanges++
				matchesToUpdate = append(matchesToUpdate, match)
			}
		}
		
		if actualChanges > 0 {
			// Replace matches in reverse order to maintain positions
			for i := len(matchesToUpdate) - 1; i >= 0; i-- {
				match := matchesToUpdate[i]
				regex := regexp.MustCompile(searchPattern)
				replacement := regex.ReplaceAllString(match.Text, replaceTemplate)
				updatedContent = updatedContent[:match.Start] + replacement + updatedContent[match.End:]
			}

			totalChanges += actualChanges
		}
	}

	// Write file if changes were made and not in dry-run mode
	if totalChanges > 0 && !v.dryRun {
		if err := ioutil.WriteFile(filePath, []byte(updatedContent), 0644); err != nil {
			return 0, fmt.Errorf("error writing %s: %w", fileConfig.Path, err)
		}
	}

	return totalChanges, nil
}

// getVersionPatterns returns the expanded version patterns (used by TUI)
func (v *VersionUpdater) getVersionPatterns() map[string]string {
	return v.expandVersionPatterns()
} 