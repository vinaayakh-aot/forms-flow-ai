package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Version represents the current application version
const AppVersion = "1.0.0"

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
		fmt.Printf("⚠️  File not found: %s\n", fileConfig.Path)
		return 0, nil
	}

	// Read file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("❌ Error reading %s: %w", fileConfig.Path, err)
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
			fmt.Printf("  ❌ Pattern error in '%s': %v\n", description, err)
			continue
		}

		matches := allMatches

		// Apply exclusions if specified
		if len(update.Exclude) > 0 {
			matches = v.applyExclusions(updatedContent, matches, update.Exclude)
			excludedCount := len(allMatches) - len(matches)
			if excludedCount > 0 {
				fmt.Printf("  🚫 Excluded %d matches due to exclusion rules\n", excludedCount)
			}
		}

		// Apply context filtering if specified
		if update.Context != nil {
			contextMatches, err := v.applyContextFiltering(updatedContent, searchPattern, update.Context)
			if err != nil {
				fmt.Printf("  ❌ Context filtering error in '%s': %v\n", description, err)
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
			fmt.Printf("  ✅ %s (%d matches)\n", description, changes)
		}
	}

	// Write file if changes were made and not in dry-run mode
	if totalChanges > 0 && !v.dryRun {
		if err := ioutil.WriteFile(filePath, []byte(updatedContent), 0644); err != nil {
			return 0, fmt.Errorf("❌ Error writing %s: %w", fileConfig.Path, err)
		}
		fmt.Printf("📝 Updated %s (%d changes)\n", fileName, totalChanges)
	} else if totalChanges > 0 {
		fmt.Printf("🔍 [DRY RUN] Would update %s (%d changes)\n", fileName, totalChanges)
	}

	return totalChanges, nil
}

// updateAllFiles updates all files specified in the configuration
func (v *VersionUpdater) updateAllFiles() error {
	fmt.Printf("🚀 Starting version update to: %s\n", v.currentVersion)
	fmt.Printf("📁 Working directory: %s\n", v.rootDir)

	if v.dryRun {
		fmt.Println("🔍 DRY RUN MODE - No files will be modified")
	}

	fmt.Printf("📋 Using config: %s\n", v.configPath)

	// Show expanded version patterns
	versionPatterns := v.expandVersionPatterns()
	if len(versionPatterns) > 0 {
		fmt.Println("\n🔧 Version patterns:")
		for name, value := range versionPatterns {
			fmt.Printf("  %s: %s\n", name, value)
		}
	}

	totalChanges := 0

	for _, fileConfig := range v.config.Files {
		fileName := fileConfig.Name
		if fileName == "" {
			fileName = fileConfig.Path
		}

		fmt.Printf("\n📄 Processing: %s\n", fileName)
		changes, err := v.updateFile(fileConfig)
		if err != nil {
			return err
		}
		totalChanges += changes
	}

	if v.dryRun {
		fmt.Printf("\n🔍 [DRY RUN] ✨ Update complete! Total changes: %d\n", totalChanges)
		if totalChanges > 0 {
			fmt.Println("Run without --dry-run to apply changes.")
		}
	} else {
		fmt.Printf("\n✨ Update complete! Total changes: %d\n", totalChanges)
	}

	return nil
}

func main() {
	var (
		dryRun     = flag.Bool("dry-run", false, "Preview changes without modifying files")
		configPath = flag.String("config", "", "Path to configuration file (default: version-cli/config-simple.json)")
		version    = flag.Bool("version", false, "Show version information")
		help       = flag.Bool("help", false, "Show help information")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "FormsFlow.ai Version Update Tool (Go Edition) v%s\n\n", AppVersion)
		fmt.Fprintf(os.Stderr, "This tool updates version references across the FormsFlow.ai repository\n")
		fmt.Fprintf(os.Stderr, "based on the version specified in the VERSION file.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --dry-run              # Preview changes\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s                        # Apply updates\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --config my.json       # Use custom config\n", os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	if *version {
		fmt.Printf("FormsFlow.ai Version Update Tool v%s\n", AppVersion)
		return
	}

	// Create version updater
	updater, err := NewVersionUpdater(*configPath, *dryRun)
	if err != nil {
		log.Fatalf("Failed to initialize version updater: %v", err)
	}

	// Update all files
	if err := updater.updateAllFiles(); err != nil {
		log.Fatalf("Update failed: %v", err)
	}
} 