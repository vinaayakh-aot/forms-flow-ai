package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Match represents a regex match with position
type Match struct {
	Start int
	End   int
	Text  string
}

// getPredefinedRegexPatterns returns commonly used regex patterns
func getPredefinedRegexPatterns() map[string]string {
	return map[string]string{
		"ff_url_version_pattern":     "@v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
		"ff_json_version_pattern":    "\"version\": \"[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+\"",
		"ff_docker_image_pattern":    "image: formsflow/forms-flow-forms:v[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
		"ff_version_basic":           "[0-9]+\\.[0-9]+\\.[0-9]+",
		"ff_version_with_prerelease": "[0-9]+\\.[0-9]+\\.[0-9]+-[a-zA-Z0-9]+",
	}
}

// findMatches finds all regex matches in content
func findMatches(content, pattern string) ([]Match, error) {
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
func applyExclusions(content string, matches []Match, exclusions []string) []Match {
	if len(exclusions) == 0 {
		return matches
	}

	lines := strings.Split(content, "\n")
	var filtered []Match

	for _, match := range matches {
		// Find which line this match is on
		lineNum := findLineNumber(content, match.Start)
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
func findLineNumber(content string, position int) int {
	lines := strings.Split(content[:position], "\n")
	return len(lines) - 1
}

// applyContextFiltering applies context filtering to find the right matches
func applyContextFiltering(content, pattern string, context *ContextConfig) ([]Match, error) {
	allMatches, err := findMatches(content, pattern)
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
	nearMatches, err := findMatches(content, regexp.QuoteMeta(context.Near))
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
func createPatternFromSimple(update UpdateConfig, versionPatterns map[string]string) (string, string) {
	searchPattern := update.Find
	replacePattern := update.Replace

	// Expand predefined regex pattern references (pattern names without {{}} syntax)
	predefinedPatterns := getPredefinedRegexPatterns()
	for patternName, patternValue := range predefinedPatterns {
		searchPattern = strings.ReplaceAll(searchPattern, patternName, patternValue)
	}

	// Expand version pattern references like {{url_version}}
	for patternName, patternValue := range versionPatterns {
		placeholder := fmt.Sprintf("{{%s}}", patternName)
		replacePattern = strings.ReplaceAll(replacePattern, placeholder, patternValue)
	}

	return searchPattern, replacePattern
} 