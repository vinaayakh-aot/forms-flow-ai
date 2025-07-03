package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		searchPattern, replaceTemplate := v.createPatternFromSimple(update, versionPatterns)

		// Find all matches
		allMatches, err := v.findMatches(updatedContent, searchPattern)
		if err != nil {
			continue
		}

		matches := allMatches

		// Apply exclusions if specified
		if len(update.Exclude) > 0 {
			matches = v.applyExclusions(updatedContent, matches, update.Exclude)
		}

		// Apply context filtering if specified
		if update.Context != nil {
			contextMatches, err := v.applyContextFiltering(updatedContent, searchPattern, update.Context)
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

var (
	// Global flags
	dryRun     bool
	configPath string
	verbose    bool
)

// Styles using lipgloss
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true).
			Margin(1, 0)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0EA5E9"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Margin(1, 0)

	progressStyle = lipgloss.NewStyle().
			Margin(1, 0)
)

// Messages for Bubble Tea
type startProcessingMsg struct{}
type fileProcessedMsg struct {
	fileName string
	changes  int
	err      error
}
type processingCompleteMsg struct {
	totalChanges int
	totalFiles   int
}

// FileStatus represents the processing status of a file
type FileStatus struct {
	name      string
	processed bool
	changes   int
	err       error
}

// TUI Model
type model struct {
	updater        *VersionUpdater
	files          []FileStatus
	currentFile    int
	totalChanges   int
	spinner        spinner.Model
	progress       progress.Model
	done           bool
	err            error
	startTime      time.Time
	completionTime time.Time
}

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
		fmt.Fprintf(os.Stderr, "%s\n", errorStyle.Render("Error: "+err.Error()))
		os.Exit(1)
	}
}

// Initialize the model
func initialModel(updater *VersionUpdater) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(progress.WithDefaultGradient())

	// Initialize file statuses
	files := make([]FileStatus, len(updater.config.Files))
	for i, fileConfig := range updater.config.Files {
		name := fileConfig.Name
		if name == "" {
			name = fileConfig.Path
		}
		files[i] = FileStatus{
			name:      name,
			processed: false,
			changes:   0,
			err:       nil,
		}
	}

	return model{
		updater:   updater,
		files:     files,
		spinner:   s,
		progress:  p,
		startTime: time.Now(),
	}
}

// Init command for Bubble Tea
func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, startProcessing)
}

// Start processing command
func startProcessing() tea.Msg {
	return startProcessingMsg{}
}

// Process a single file asynchronously
func (m model) processNextFile() tea.Cmd {
	if m.currentFile >= len(m.updater.config.Files) {
		return func() tea.Msg {
			return processingCompleteMsg{
				totalChanges: m.totalChanges,
				totalFiles:   len(m.updater.config.Files),
			}
		}
	}

	fileConfig := m.updater.config.Files[m.currentFile]
	fileName := fileConfig.Name
	if fileName == "" {
		fileName = fileConfig.Path
	}

	return func() tea.Msg {
		// Small delay for visual effect
		time.Sleep(300 * time.Millisecond)
		
		changes, err := m.updater.updateFile(fileConfig)
		
		return fileProcessedMsg{
			fileName: fileName,
			changes:  changes,
			err:      err,
		}
	}
}

// Update function for Bubble Tea
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case startProcessingMsg:
		return m, m.processNextFile()

	case fileProcessedMsg:
		// Update file status
		if m.currentFile < len(m.files) {
			m.files[m.currentFile].processed = true
			m.files[m.currentFile].changes = msg.changes
			m.files[m.currentFile].err = msg.err
			m.totalChanges += msg.changes
			m.currentFile++

			return m, m.processNextFile()
		}

	case processingCompleteMsg:
		m.done = true
		m.totalChanges = msg.totalChanges
		m.completionTime = time.Now() // Record actual completion time
		// Show completion message briefly, then auto-quit
		return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return tea.Quit()
		})

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View function for Bubble Tea
func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}

	var b strings.Builder

	// Header with padding and version
	b.WriteString("\n\n")
	
	// Create a beautiful header
	headerTitle := "🚀 FormsFlow.ai Version Updater"
	headerVersion := fmt.Sprintf("v%s", AppVersion)
	
	// Center the title and version
	titleLine := titleStyle.Render(headerTitle)
	versionLine := dimStyle.Render(headerVersion)
	
	b.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Width(60).Render(titleLine))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Width(60).Render(versionLine))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("━", 60))
	b.WriteString("\n\n")

	// Version info
	b.WriteString(headerStyle.Render("Version: ") + infoStyle.Render(m.updater.currentVersion))
	b.WriteString("\n")
	
	mode := "APPLY CHANGES"
	modeStyle := successStyle
	if m.updater.dryRun {
		mode = "DRY RUN (preview only)"
		modeStyle = warningStyle
	}
	b.WriteString(headerStyle.Render("Mode: ") + modeStyle.Render(mode))
	b.WriteString("\n\n")

	if verbose {
		b.WriteString(dimStyle.Render("Config: " + m.updater.configPath))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Working directory: " + m.updater.rootDir))
		b.WriteString("\n\n")
	}

	// Progress
	if !m.done {
		progress := float64(m.currentFile) / float64(len(m.files))
		if progress > 1.0 {
			progress = 1.0
		}
		
		// Progress bar with detailed info
		progressText := fmt.Sprintf("Progress: %d/%d files", m.currentFile, len(m.files))
		b.WriteString(dimStyle.Render(progressText))
		b.WriteString("\n")
		b.WriteString(progressStyle.Render(m.progress.ViewAs(progress)))
		b.WriteString("\n")
		
		if m.currentFile < len(m.files) {
			b.WriteString(m.spinner.View() + " Processing: " + infoStyle.Render(m.files[m.currentFile].name))
			b.WriteString("\n")
			
			// Show elapsed time during processing
			elapsed := time.Since(m.startTime)
			b.WriteString(dimStyle.Render(fmt.Sprintf("Elapsed: %v", elapsed.Round(time.Millisecond))))
			b.WriteString("\n\n")
		} else {
			b.WriteString("🎉 Processing complete!\n\n")
		}
	}

	// File statuses
	for i, file := range m.files {
		if i <= m.currentFile {
			var status string
			if file.processed {
				if file.err != nil {
					status = errorStyle.Render("✗ " + file.name + " (error)")
				} else if file.changes > 0 {
					status = successStyle.Render(fmt.Sprintf("✓ %s (%d changes)", file.name, file.changes))
				} else {
					status = dimStyle.Render("○ " + file.name + " (up to date)")
				}
			} else if i == m.currentFile {
				status = infoStyle.Render("◐ " + file.name + " (processing...)")
			}
			
			if status != "" {
				b.WriteString("  " + status)
				b.WriteString("\n")
			}
		}
	}

	// Summary when done
	if m.done {
		b.WriteString("\n")
		b.WriteString(strings.Repeat("━", 60))
		b.WriteString("\n\n")

		// Count files that need changes
		filesWithChanges := 0
		for _, file := range m.files {
			if file.processed && file.changes > 0 {
				filesWithChanges++
			}
		}

		if m.totalChanges > 0 {
			if m.updater.dryRun {
				summary := fmt.Sprintf("✅ Found %d changes across %d files", 
					m.totalChanges, filesWithChanges)
				b.WriteString(boxStyle.Render(successStyle.Render(summary)))
			} else {
				summary := fmt.Sprintf("✅ Updated %d references across %d files", 
					m.totalChanges, filesWithChanges)
				b.WriteString(boxStyle.Render(successStyle.Render(summary)))
			}
		} else {
			summary := "✨ All versions are already up to date"
			b.WriteString(boxStyle.Render(infoStyle.Render(summary)))
		}
		
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("🎉 Processing complete! Exiting..."))
	}

	return b.String()
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Create version updater
	updater, err := NewVersionUpdater(configPath, dryRun)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Initialize and run Bubble Tea program (without alt screen)
	p := tea.NewProgram(initialModel(updater))
	
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	// Print final summary after TUI exits
	if m, ok := finalModel.(model); ok {
		printFinalSummary(m)
	}

	return nil
}

// createFilesTable creates a beautiful table showing files and their changes
func createFilesTable(m model) string {
	// Define columns
	columns := []table.Column{
		{Title: "File", Width: 40},
		{Title: "Changes", Width: 10},
		{Title: "Status", Width: 15},
	}

	// Build rows - only show files that have changes or errors
	var rows []table.Row
	for _, file := range m.files {
		if file.processed && (file.changes > 0 || file.err != nil) {
			statusText := "✓ Success"
			
			if file.err != nil {
				statusText = "✗ Error"
			}
			
			rows = append(rows, table.Row{
				file.name,
				fmt.Sprintf("%d", file.changes),
				statusText,
			})
		}
	}

	// Create and configure table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)),
	)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t.View()
}

// Print a nice summary after the TUI exits
func printFinalSummary(m model) {
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))
	
	// Header
	fmt.Println(titleStyle.Render("🎉 FormsFlow.ai Version Update Complete"))
	fmt.Println()
	
	// Summary info
	fmt.Printf("%s %s\n", headerStyle.Render("Version Updated:"), infoStyle.Render(m.updater.currentVersion))
	
	mode := "APPLIED CHANGES"
	modeStyle := successStyle
	if m.updater.dryRun {
		mode = "DRY RUN COMPLETED"
		modeStyle = warningStyle
	}
	fmt.Printf("%s %s\n", headerStyle.Render("Mode:"), modeStyle.Render(mode))
	fmt.Println()
	
	// Count files that need changes
	filesWithChanges := 0
	filesAlreadyUpToDate := 0
	for _, file := range m.files {
		if file.processed {
			if file.changes > 0 {
				filesWithChanges++
			} else if file.err == nil {
				filesAlreadyUpToDate++
			}
		}
	}
	
	// Results
	if m.totalChanges > 0 {
		if m.updater.dryRun {
			fmt.Println(warningStyle.Render(fmt.Sprintf("📋 Would make %d changes across %d files", 
				m.totalChanges, filesWithChanges)))
			fmt.Println(dimStyle.Render("   Run without --dry-run to apply these changes"))
		} else {
			fmt.Println(successStyle.Render(fmt.Sprintf("✅ Successfully updated %d references across %d files", 
				m.totalChanges, filesWithChanges)))
		}
		
		if filesAlreadyUpToDate > 0 {
			fmt.Println(dimStyle.Render(fmt.Sprintf("   %d files already up to date (skipped)", filesAlreadyUpToDate)))
		}
	} else {
		fmt.Println(infoStyle.Render("ℹ️  All versions are already up to date - no changes needed"))
	}
	
	// Files table - only show if there are files with changes
	if m.totalChanges > 0 {
		fmt.Println()
		fmt.Println(createFilesTable(m))
	}

	// Timing with progress bar
	if !m.startTime.IsZero() && !m.completionTime.IsZero() {
		elapsed := m.completionTime.Sub(m.startTime)
		fmt.Println()
		
		// Create a temporary progress bar for completion display
		completionProgress := progress.New(progress.WithDefaultGradient())
		
		// Show 100% completion with timing
		fmt.Println(dimStyle.Render(fmt.Sprintf("Completed in %v", elapsed.Round(time.Millisecond))))
		fmt.Println(progressStyle.Render(completionProgress.ViewAs(1.0))) // 100% complete
	}
	
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println()
} 