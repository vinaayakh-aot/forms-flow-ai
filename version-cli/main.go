package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Build information (set by linker flags)
var (
	AppVersion = "1.0.0"        // Application version
	BuildTime  = "unknown"      // Build timestamp
	GitCommit  = "unknown"      // Git commit hash
	GitBranch  = "unknown"      // Git branch name
)

var (
	// Global flags
	dryRun      bool
	configPath  string
	versionFile string
	verbose     bool
)



func main() {
	var rootCmd = &cobra.Command{
		Use:   "formsflow-version-updater",
		Short: "Update version references across FormsFlow.ai repository",
		Long: `FormsFlow.ai Version Update Tool

This tool reads the version from a VERSION file and updates all
configured files across the repository to maintain version consistency.
By default, it looks for a VERSION file in the current directory, but
you can specify a custom path using the --version-file flag.`,
		Version: AppVersion,
		RunE:    runUpdate,
	}

	// Add flags
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying files")
	rootCmd.Flags().StringVar(&configPath, "config", "", "Path to configuration file")
	rootCmd.Flags().StringVar(&versionFile, "version-file", "", "Path to VERSION file (default: ./VERSION)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")

	// Add examples
	rootCmd.Example = `  # Preview changes
  formsflow-version-updater --dry-run

  # Apply updates
  formsflow-version-updater

  # Use custom config
  formsflow-version-updater --config my-config.json

  # Use custom version file
  formsflow-version-updater --version-file ./app-version.txt

  # Use both custom config and version file
  formsflow-version-updater --config my-config.json --version-file ./v.txt`

	// Customize version output
	versionTemplate := fmt.Sprintf(`%s
%s
%s

%s %s
%s Enterprise-grade version synchronization tool
%s Build: %s
%s Commit: %s (%s)
%s Copyright © 2024 FormsFlow.ai

`,
		strings.Repeat("━", 50),
		"🚀 FormsFlow.ai Version Updater",
		strings.Repeat("━", 50),
		"Version:",
		"v{{.Version}}",
		"📦",
		"📅",
		BuildTime,
		"🔗",
		GitCommit,
		GitBranch,
		"©")
	
	rootCmd.SetVersionTemplate(versionTemplate)

	// Customize help output
	helpTemplate := fmt.Sprintf(`%s
%s
%s

%s
%s

%s
{{if .Runnable}}  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%s
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

%s
{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

%s{{end}}

%s

`,
		strings.Repeat("━", 60),
		"🚀 FormsFlow.ai Version Updater",
		strings.Repeat("━", 60),
		"📋 DESCRIPTION",
		"   Enterprise-grade version synchronization tool for FormsFlow.ai",
		"🔧 USAGE",
		"📝 ALIASES",
		"💡 EXAMPLES",
		"🎯 AVAILABLE COMMANDS",
		"⚙️  FLAGS",
		"🌐 GLOBAL FLAGS",
		"❓ ADDITIONAL HELP TOPICS",
		"Use \"{{.CommandPath}} [command] --help\" for more information about a command.",
		strings.Repeat("━", 60))

	rootCmd.SetHelpTemplate(helpTemplate)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Create version updater
	updater, err := NewVersionUpdater(configPath, versionFile, dryRun)
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