package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Version represents the current application version
var AppVersion = "1.0.0"

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

// Match represents a regex match with position
type Match struct {
	Start int
	End   int
	Text  string
}

// VersionUpdater handles the version update logic
type VersionUpdater struct {
	rootDir        string
	configPath     string
	config         Config
	currentVersion string
	dryRun         bool
}

// NewVersionUpdater creates a new version updater instance
func NewVersionUpdater(configPath string, dryRun bool) (*VersionUpdater, error) {
	// Get the directory containing the executable or current working directory
	rootDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Default config path if not provided
	if configPath == "" {
		configPath = filepath.Join(rootDir, "version-cli", "config-simple.json")
	}

	updater := &VersionUpdater{
		rootDir:    rootDir,
		configPath: configPath,
		dryRun:     dryRun,
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

// readVersion reads the current version from the VERSION file
func (v *VersionUpdater) readVersion() error {
	versionFile := filepath.Join(v.rootDir, "VERSION")
	data, err := ioutil.ReadFile(versionFile)
	if err != nil {
		return fmt.Errorf("❌ VERSION file not found at %s: %w", versionFile, err)
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

// findMatches finds all regex matches in content
func (v *VersionUpdater) findMatches(content, pattern string) ([]Match, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	matches := regex.FindAllStringSubmatchIndex(content, -1)
	var result []Match

	for _, match := range matches {
		if len(match) >= 2 {
			start := match[0]
			end := match[1]
			text := content[start:end]
			result = append(result, Match{
				Start: start,
				End:   end,
				Text:  text,
			})
		}
	}

	return result, nil
}

// applyExclusions filters out matches that occur on lines containing excluded strings
func (v *VersionUpdater) applyExclusions(content string, matches []Match, exclusions []string) []Match {
	if len(exclusions) == 0 {
		return matches
	}

	lines := strings.Split(content, "\n")
	var filtered []Match

	for _, match := range matches {
		// Find which line this match is on
		lineNum := v.findLineNumber(content, match.Start)
		if lineNum >= len(lines) {
			continue
		}

		currentLine := lines[lineNum]
		excludeMatch := false

		// Check if this line contains any excluded strings
		for _, exclusion := range exclusions {
			if strings.Contains(currentLine, exclusion) {
				excludeMatch = true
				break
			}
		}

		if !excludeMatch {
			filtered = append(filtered, match)
		}
	}

	return filtered
}

// findLineNumber finds the line number for a given character position
func (v *VersionUpdater) findLineNumber(content string, position int) int {
	lines := strings.Split(content[:position], "\n")
	return len(lines) - 1
}

// applyContextFiltering applies context filtering to find the right matches
func (v *VersionUpdater) applyContextFiltering(content, pattern string, context *ContextConfig) ([]Match, error) {
	allMatches, err := v.findMatches(content, pattern)
	if err != nil {
		return nil, err
	}

	if context.Near == "" {
		return allMatches, nil
	}

	maxMatches := context.MaxMatches
	if maxMatches == 0 {
		maxMatches = 1
	}

	maxDistance := context.MaxDistance
	if maxDistance == 0 {
		maxDistance = 200
	}

	// Find all "near" pattern occurrences
	nearMatches, err := v.findMatches(content, regexp.QuoteMeta(context.Near))
	if err != nil {
		return nil, err
	}

	var filtered []Match
	for _, nearMatch := range nearMatches {
		nearPos := nearMatch.Start

		// Find matches within distance of this "near" pattern
		for _, match := range allMatches {
			distance := match.Start - nearPos
			if distance < 0 {
				distance = -distance
			}

			if distance <= maxDistance {
				filtered = append(filtered, match)
				if len(filtered) >= maxMatches {
					break
				}
			}
		}

		if len(filtered) >= maxMatches {
			break
		}
	}

	return filtered, nil
}

// createPatternFromSimple converts simple config format to search/replace patterns
func (v *VersionUpdater) createPatternFromSimple(update UpdateConfig, versionPatterns map[string]string) (string, string) {
	searchPattern := update.Find
	replacePattern := update.Replace

	// Expand pattern references like {{url_version}}
	for patternName, patternValue := range versionPatterns {
		placeholder := fmt.Sprintf("{{%s}}", patternName)
		replacePattern = strings.ReplaceAll(replacePattern, placeholder, patternValue)
	}

	return searchPattern, replacePattern
}

// updateFile updates a single file using the simple configuration format
func (v *VersionUpdater) updateFile(fileConfig FileConfig) (int, error) {
	fileName := fileConfig.Name
	if fileName == "" {
		fileName = fileConfig.Path
	}

	filePath := filepath.Join(v.rootDir, fileConfig.Path)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if verbose {
			warningColor.Printf("  File not found: %s\n", fileConfig.Path)
		}
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
		description := update.Description
		if description == "" {
			description = "No description"
		}

		searchPattern, replaceTemplate := v.createPatternFromSimple(update, versionPatterns)

		// Find all matches
		allMatches, err := v.findMatches(updatedContent, searchPattern)
		if err != nil {
			if verbose {
				errorColor.Printf("  Pattern error in '%s': %v\n", description, err)
			}
			continue
		}

		matches := allMatches

		// Apply exclusions if specified
		if len(update.Exclude) > 0 {
			matches = v.applyExclusions(updatedContent, matches, update.Exclude)
			excludedCount := len(allMatches) - len(matches)
			if excludedCount > 0 && verbose {
				dimColor.Printf("  Excluded %d matches due to exclusion rules\n", excludedCount)
			}
		}

		// Apply context filtering if specified
		if update.Context != nil {
			contextMatches, err := v.applyContextFiltering(updatedContent, searchPattern, update.Context)
			if err != nil {
				if verbose {
					errorColor.Printf("  Context filtering error in '%s': %v\n", description, err)
				}
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

		changes := len(matches)
		if changes > 0 {
			// Replace matches in reverse order to maintain positions
			for i := len(matches) - 1; i >= 0; i-- {
				match := matches[i]
				regex := regexp.MustCompile(searchPattern)
				replacement := regex.ReplaceAllString(match.Text, replaceTemplate)
				updatedContent = updatedContent[:match.Start] + replacement + updatedContent[match.End:]
			}

			totalChanges += changes
			if verbose {
				successColor.Printf("  ✓ %s (%d matches)\n", description, changes)
			}
		}
	}

	// Write file if changes were made and not in dry-run mode
	if totalChanges > 0 && !v.dryRun {
		if err := ioutil.WriteFile(filePath, []byte(updatedContent), 0644); err != nil {
			return 0, fmt.Errorf("error writing %s: %w", fileConfig.Path, err)
		}
		if verbose {
			successColor.Printf("  Updated %s (%d changes)\n", fileName, totalChanges)
		}
	}

	return totalChanges, nil
}

// updateAllFiles updates all files specified in the configuration
func (v *VersionUpdater) updateAllFiles() error {
	// Header
	headerColor.Printf("FormsFlow.ai Version Updater\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
	// Basic info
	fmt.Printf("Version: %s\n", infoColor.Sprint(v.currentVersion))
	if v.dryRun {
		warningColor.Printf("Mode: DRY RUN (preview only)\n")
	} else {
		infoColor.Printf("Mode: APPLY CHANGES\n")
	}
	
	if verbose {
		dimColor.Printf("Config: %s\n", v.configPath)
		dimColor.Printf("Working directory: %s\n", v.rootDir)
	}
	
	fmt.Println()

	// Show version patterns in verbose mode
	if verbose {
		versionPatterns := v.expandVersionPatterns()
		if len(versionPatterns) > 0 {
			dimColor.Printf("Version patterns:\n")
			for name, value := range versionPatterns {
				dimColor.Printf("  %s: %s\n", name, value)
			}
			fmt.Println()
		}
	}

	totalChanges := 0
	fileCount := len(v.config.Files)
	processedCount := 0

	// Simple progress indicator
	if !verbose {
		fmt.Printf("Processing %d files...\n", fileCount)
	}

	for _, fileConfig := range v.config.Files {
		fileName := fileConfig.Name
		if fileName == "" {
			fileName = fileConfig.Path
		}

		processedCount++
		
		if verbose {
			infoColor.Printf("Processing [%d/%d]: %s\n", processedCount, fileCount, fileName)
		}

		changes, err := v.updateFile(fileConfig)
		if err != nil {
			return err
		}
		totalChanges += changes
		
		if !verbose && changes > 0 {
			successColor.Printf("✓ %s (%d changes)\n", fileName, changes)
		}
	}

	if !verbose {
		fmt.Println()
	}

	// Summary
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	if totalChanges > 0 {
		if v.dryRun {
			warningColor.Printf("DRY RUN: Would make %d changes across %d files\n", totalChanges, fileCount)
			fmt.Printf("Run without --dry-run to apply changes.\n")
		} else {
			successColor.Printf("SUCCESS: Updated %d references across %d files\n", totalChanges, fileCount)
		}
	} else {
		infoColor.Printf("No changes needed - all versions are already up to date\n")
	}

	return nil
}

var (
	// Global flags
	dryRun     bool
	configPath string
	verbose    bool
)

// Colors for consistent output
var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warningColor = color.New(color.FgYellow, color.Bold)
	infoColor    = color.New(color.FgBlue)
	headerColor  = color.New(color.FgCyan, color.Bold)
	dimColor     = color.New(color.FgHiBlack)
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "formsflow-version-updater",
		Short: "Update version references across FormsFlow.ai repository",
		Long: `FormsFlow.ai Version Update Tool

This tool reads the version from the VERSION file and updates all
configured files across the repository to maintain version consistency.`,
		Version: AppVersion,
		RunE:    runUpdate,
	}

	// Add flags
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying files")
	rootCmd.Flags().StringVar(&configPath, "config", "", "Path to configuration file")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")

	// Add examples
	rootCmd.Example = `  # Preview changes
  formsflow-version-updater --dry-run

  # Apply updates
  formsflow-version-updater

  # Use custom config
  formsflow-version-updater --config my-config.json`

	if err := rootCmd.Execute(); err != nil {
		errorColor.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Create version updater
	updater, err := NewVersionUpdater(configPath, dryRun)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Update all files
	return updater.updateAllFiles()
} 