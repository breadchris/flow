package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// setConfigDefaults sets default values for all configuration sections
func setConfigDefaults(config *AppConfig) {
	// SlackBot defaults
	config.SlackBot = SlackBotConfig{
		Enabled:          true,
		SessionTimeout:   30 * time.Minute,
		MaxSessions:      10,
		WorkingDirectory: "/tmp/slackbot",
		Debug:            true,
	}

	// Claude defaults
	config.Claude = ClaudeConfig{
		Debug:    true,
		DebugDir: "/tmp/claude",
		Tools:    []string{"Read", "Write", "Bash"},
	}

	// Worklet defaults
	config.Worklet = WorkletConfig{
		BaseDir:       "/tmp/worklet-repos",
		CleanupMaxAge: 24 * time.Hour,
		MaxConcurrent: 5,
	}

	// Git defaults
	config.Git = GitConfig{
		BaseDir: "/tmp/git-repos",
	}
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(config *AppConfig) {
	// Claude environment variables
	if debugStr := os.Getenv("CLAUDE_DEBUG"); debugStr != "" {
		config.Claude.Debug = debugStr == "true" || debugStr == "1"
	}
	if debugDir := os.Getenv("CLAUDE_DEBUG_DIR"); debugDir != "" {
		config.Claude.DebugDir = debugDir
	}
	if tools := os.Getenv("CLAUDE_TOOLS"); tools != "" {
		// Split comma-separated tools
		config.Claude.Tools = parseCommaSeparated(tools)
	}

	// Worklet environment variables
	if baseDir := os.Getenv("WORKLET_BASE_DIR"); baseDir != "" {
		config.Worklet.BaseDir = baseDir
	}
	if maxAgeStr := os.Getenv("WORKLET_CLEANUP_MAX_AGE"); maxAgeStr != "" {
		if maxAge, err := time.ParseDuration(maxAgeStr); err == nil {
			config.Worklet.CleanupMaxAge = maxAge
		}
	}
	if maxConcurrentStr := os.Getenv("WORKLET_MAX_CONCURRENT"); maxConcurrentStr != "" {
		if maxConcurrent, err := strconv.Atoi(maxConcurrentStr); err == nil {
			config.Worklet.MaxConcurrent = maxConcurrent
		}
	}

	// Git environment variables
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		config.Git.Token = token
	}
	if baseDir := os.Getenv("GIT_BASE_DIR"); baseDir != "" {
		config.Git.BaseDir = baseDir
	}
}

// parseCommaSeparated splits a comma-separated string into a slice of strings
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}

	var result []string
	for _, item := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
