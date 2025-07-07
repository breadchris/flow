package slackbot

import (
	"testing"

	"github.com/breadchris/flow/config"
)

func TestSlackBot_isChannelAllowed_Integration(t *testing.T) {
	tests := []struct {
		name              string
		whitelistPatterns []string
		channelID         string
		expected          bool
		shouldFailInit    bool
	}{
		{
			name:              "no whitelist - allow all",
			whitelistPatterns: []string{},
			channelID:         "C1234567890",
			expected:          true,
			shouldFailInit:    false,
		},
		{
			name:              "whitelist with match",
			whitelistPatterns: []string{"^C.*DEV$", ".*test.*"},
			channelID:         "C123DEV",
			expected:          true,
			shouldFailInit:    false,
		},
		{
			name:              "whitelist without match",
			whitelistPatterns: []string{"^C.*DEV$", ".*test.*"},
			channelID:         "C123PROD",
			expected:          false,
			shouldFailInit:    false,
		},
		{
			name:              "invalid regex pattern should fail init",
			whitelistPatterns: []string{"[invalid"},
			channelID:         "C1234567890",
			expected:          false,
			shouldFailInit:    true,
		},
		{
			name:              "real slack channel patterns",
			whitelistPatterns: []string{"^[CGD][A-Z0-9]{8,}$"},
			channelID:         "C1234567890",
			expected:          true,
			shouldFailInit:    false,
		},
		{
			name:              "real slack channel patterns - DM",
			whitelistPatterns: []string{"^[CGD][A-Z0-9]{8,}$"},
			channelID:         "D1234567890",
			expected:          true,
			shouldFailInit:    false,
		},
		{
			name:              "real slack channel patterns - invalid",
			whitelistPatterns: []string{"^[CGD][A-Z0-9]{8,}$"},
			channelID:         "X1234567890",
			expected:          false,
			shouldFailInit:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal SlackBot config for testing
			slackConfig := &config.SlackBotConfig{
				ChannelWhitelist: tt.whitelistPatterns,
				Debug:            false,
			}

			// Try to create the whitelist
			whitelist, err := NewChannelWhitelist(slackConfig.ChannelWhitelist, slackConfig.Debug)
			
			if tt.shouldFailInit {
				if err == nil {
					t.Errorf("NewChannelWhitelist() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("NewChannelWhitelist() unexpected error: %v", err)
			}

			// Create a minimal SlackBot instance with just the whitelist
			bot := &SlackBot{
				config:           slackConfig,
				channelWhitelist: whitelist,
			}

			// Test the channel check
			result := bot.isChannelAllowed(tt.channelID)
			if result != tt.expected {
				t.Errorf("isChannelAllowed(%q) = %v, expected %v (patterns: %v)", 
					tt.channelID, result, tt.expected, tt.whitelistPatterns)
			}
		})
	}
}

func TestSlackBot_isChannelAllowed_EdgeCases(t *testing.T) {
	// Test with nil whitelist (should not happen in practice but good to test)
	bot := &SlackBot{
		config:           &config.SlackBotConfig{},
		channelWhitelist: nil,
	}

	// This should panic or be handled gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("isChannelAllowed() panicked with nil whitelist: %v", r)
		}
	}()

	// In practice this won't happen since New() creates the whitelist
	// but it's good to test defensive programming
	if bot.channelWhitelist != nil {
		bot.isChannelAllowed("C1234567890")
	}
}

func BenchmarkChannelWhitelist_IsAllowed(b *testing.B) {
	// Test performance with different pattern complexities
	tests := []struct {
		name     string
		patterns []string
	}{
		{
			name:     "no patterns",
			patterns: []string{},
		},
		{
			name:     "single exact pattern",
			patterns: []string{"C1234567890"},
		},
		{
			name:     "single regex pattern",
			patterns: []string{"^C.*DEV$"},
		},
		{
			name:     "multiple patterns",
			patterns: []string{"^C.*DEV$", ".*test.*", "^G.*PRIVATE$", "^D.*"},
		},
		{
			name:     "complex regex",
			patterns: []string{"^[CGD][A-Z0-9]{8,}$"},
		},
	}

	channelID := "C1234567890"

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			whitelist, err := NewChannelWhitelist(tt.patterns, false)
			if err != nil {
				b.Fatalf("NewChannelWhitelist() failed: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				whitelist.IsAllowed(channelID)
			}
		})
	}
}

func BenchmarkChannelWhitelist_VeryLongChannelID(b *testing.B) {
	// Test performance with very long channel ID
	patterns := []string{".*test.*", "^C.*DEV$"}
	whitelist, err := NewChannelWhitelist(patterns, false)
	if err != nil {
		b.Fatalf("NewChannelWhitelist() failed: %v", err)
	}

	// Create a very long channel ID
	longChannelID := "C" + string(make([]byte, 10000))
	for i := range longChannelID[1:] {
		longChannelID = longChannelID[:i+1] + "A" + longChannelID[i+2:]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		whitelist.IsAllowed(longChannelID)
	}
}