package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	// Add processing phase header
	b.WriteString(headerStyle.Render("🔍 LIVE PROCESSING"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("   Scanning configuration files for version references"))
	b.WriteString("\n\n")
	
	// Version info
	b.WriteString(headerStyle.Render("Target Version: ") + infoStyle.Render(m.updater.currentVersion))
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
		b.WriteString(dimStyle.Render("Version file: " + m.updater.versionFilePath))
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
					status = dimStyle.Render("○ " + file.name + " (already synchronized)")
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

		// Add section header to clarify this is the processing summary
		b.WriteString(headerStyle.Render("📊 SCAN RESULTS"))
		b.WriteString("\n\n")

		// Count files that need changes
		filesWithChanges := 0
		filesUpToDate := 0
		for _, file := range m.files {
			if file.processed {
				if file.changes > 0 {
					filesWithChanges++
				} else if file.err == nil {
					filesUpToDate++
				}
			}
		}

		// Create stats box
		var statsBox strings.Builder
		statsBox.WriteString(fmt.Sprintf("Files Scanned: %d    │ References Found: %d\n", len(m.files), m.totalChanges))
		statsBox.WriteString(fmt.Sprintf("Files to Update: %d   │ Files Up-to-date: %d\n", filesWithChanges, filesUpToDate))
		statsBox.WriteString(fmt.Sprintf("Config: %s", filepath.Base(m.updater.configPath)))
		
		statsBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			Margin(0, 0, 1, 0)
		
		b.WriteString(statsBoxStyle.Render(statsBox.String()))
		b.WriteString("\n")

		if m.totalChanges > 0 {
			if m.updater.dryRun {
				summary := fmt.Sprintf("✅ Found %d version references to update across %d files", 
					m.totalChanges, filesWithChanges)
				b.WriteString(boxStyle.Render(successStyle.Render(summary)))
			} else {
				summary := fmt.Sprintf("✅ Updated %d version references across %d files", 
					m.totalChanges, filesWithChanges)
				b.WriteString(boxStyle.Render(successStyle.Render(summary)))
			}
		} else {
			summary := "✨ All versions are already synchronized"
			b.WriteString(boxStyle.Render(infoStyle.Render(summary)))
		}
		
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("🎉 Scan complete! Generating final report..."))
	}

	return b.String()
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
	
	// Header with clear section identification
	fmt.Println(titleStyle.Render("📋 FINAL SUMMARY REPORT"))
	fmt.Println(dimStyle.Render("   Complete details of the version synchronization operation"))
	fmt.Println()
	fmt.Println(titleStyle.Render("🎉 FormsFlow.ai Version Synchronization Complete"))
	fmt.Println()
	
	// Summary info
	fmt.Printf("%s %s\n", headerStyle.Render("Target Version:"), infoStyle.Render(m.updater.currentVersion))
	
	mode := "CHANGES APPLIED"
	modeStyle := successStyle
	if m.updater.dryRun {
		mode = "DRY RUN COMPLETED"
		modeStyle = warningStyle
	}
	fmt.Printf("%s %s\n", headerStyle.Render("Operation:"), modeStyle.Render(mode))
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
			fmt.Println(warningStyle.Render(fmt.Sprintf("📋 Would update %d version references across %d files", 
				m.totalChanges, filesWithChanges)))
			fmt.Println(dimStyle.Render("   Run without --dry-run to apply these changes"))
		} else {
			fmt.Println(successStyle.Render(fmt.Sprintf("✅ Successfully updated %d version references across %d files", 
				m.totalChanges, filesWithChanges)))
		}
		
		if filesAlreadyUpToDate > 0 {
			fmt.Println(dimStyle.Render(fmt.Sprintf("   %d files already synchronized (skipped)", filesAlreadyUpToDate)))
		}
	} else {
		fmt.Println(infoStyle.Render("ℹ️  All versions are already synchronized - no changes needed"))
	}
	
	// Files table - only show if there are files with changes
	if m.totalChanges > 0 {
		fmt.Println()
		if m.updater.dryRun {
			fmt.Println(headerStyle.Render("📁 FILES TO BE UPDATED"))
		} else {
			fmt.Println(headerStyle.Render("📁 FILES UPDATED"))
		}
		fmt.Println(createFilesTable(m))
	}

	// Next steps section for dry run
	if m.updater.dryRun && m.totalChanges > 0 {
		fmt.Println()
		fmt.Println(headerStyle.Render("🚀 NEXT STEPS"))
		
		nextStepsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F59E0B")).
			Padding(1, 2).
			Margin(0, 0, 1, 0)
		
		nextStepsText := fmt.Sprintf("Ready to apply changes!\n\n%s\n\nThis will modify %d files with %d version updates", 
			successStyle.Render("Run the CLI command without --dry-run flag to make the changes"), 
			filesWithChanges, 
			m.totalChanges)
		
		fmt.Println(nextStepsBox.Render(nextStepsText))
	}

	// Timing with progress bar
	if !m.startTime.IsZero() && !m.completionTime.IsZero() {
		elapsed := m.completionTime.Sub(m.startTime)
		fmt.Println()
		
		// Create a temporary progress bar for completion display
		completionProgress := progress.New(progress.WithDefaultGradient())
		
		// Show 100% completion with timing
		fmt.Println(headerStyle.Render("⏱️  EXECUTION SUMMARY"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("Operation completed in %v", elapsed.Round(time.Millisecond))))
		fmt.Println(progressStyle.Render(completionProgress.ViewAs(1.0))) // 100% complete
	}
	
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println()
} 