package slackbot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/breadchris/flow/claude"
	"github.com/breadchris/flow/config"
	"github.com/breadchris/flow/deps"
	"github.com/breadchris/flow/worklet"
	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// SlackBot manages Slack interactions and Claude sessions
type SlackBot struct {
	client         *slack.Client
	socketMode     *socketmode.Client
	claudeService  *claude.Service
	workletManager *worklet.Manager
	sessions       map[string]*SlackClaudeSession // thread_ts -> session
	mu             sync.RWMutex
	config         *config.SlackBotConfig
	ctx            context.Context
	cancel         context.CancelFunc
}

// SlackClaudeSession represents a Claude session tied to a Slack thread
type SlackClaudeSession struct {
	ThreadTS     string          `json:"thread_ts"`
	ChannelID    string          `json:"channel_id"`
	UserID       string          `json:"user_id"`
	SessionID    string          `json:"session_id"` // Claude session ID
	ProcessID    string          `json:"process_id"` // Claude process correlation ID
	LastActivity time.Time       `json:"last_activity"`
	Context      string          `json:"context"` // Working directory context
	Active       bool            `json:"active"`  // Whether the session is currently active
	Process      *claude.Process `json:"-"`       // Active Claude process (not serialized)
}

// New creates a new SlackBot instance
func New(d deps.Deps) (*SlackBot, error) {
	slackConfig := d.Config.GetSlackBotConfig()
	if !d.Config.IsSlackBotEnabled() {
		return nil, fmt.Errorf("slack bot is disabled")
	}

	// Create Slack client
	client := slack.New(
		slackConfig.BotToken,
		slack.OptionDebug(slackConfig.Debug),
		slack.OptionAppLevelToken(slackConfig.SlackToken),
	)
	// Create socket mode client
	socketClient := socketmode.New(
		client,
		socketmode.OptionDebug(slackConfig.Debug),
	)

	// Create Claude service with appropriate configuration
	claudeConfig := claude.Config{
		Debug:    slackConfig.Debug,
		DebugDir: "/tmp/slackbot-claude",
		Tools:    []string{"Read", "Write", "Bash"},
	}
	claudeService := claude.NewService(claudeConfig)

	// Create worklet manager
	workletManager := worklet.NewManager(&d)

	ctx, cancel := context.WithCancel(context.Background())

	bot := &SlackBot{
		client:         client,
		socketMode:     socketClient,
		claudeService:  claudeService,
		workletManager: workletManager,
		sessions:       make(map[string]*SlackClaudeSession),
		config:         slackConfig,
		ctx:            ctx,
		cancel:         cancel,
	}

	return bot, nil
}

// Start begins listening for Slack events
func (b *SlackBot) Start(ctx context.Context) error {
	slog.Info("Starting Slack bot", "debug", b.config.Debug)

	// Start session cleanup goroutine
	go b.cleanupSessions()

	// Handle socket mode events
	go func() {
		for evt := range b.socketMode.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				slog.Info("Slack bot connecting...")

			case socketmode.EventTypeConnectionError:
				slog.Error("Slack bot connection error", "error", evt.Data)

			case socketmode.EventTypeConnected:
				slog.Info("Slack bot connected")

			case socketmode.EventTypeSlashCommand:
				cmd, ok := evt.Data.(slack.SlashCommand)
				if !ok {
					slog.Error("Failed to type assert slash command")
					continue
				}
				b.handleSlashCommand(&evt, &cmd)

			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					slog.Error("Failed to type assert events API event")
					continue
				}
				b.handleEventsAPI(&evt, &eventsAPIEvent)

			default:
				if b.config.Debug {
					slog.Debug("Unhandled socket mode event", "type", evt.Type)
				}
			}
		}
	}()

	// Start the socket mode client
	return b.socketMode.Run()
}

// Stop gracefully shuts down the bot
func (b *SlackBot) Stop() error {
	slog.Info("Stopping Slack bot")

	// Cancel context to stop goroutines
	b.cancel()

	// Close all active Claude sessions
	b.mu.Lock()
	for _, session := range b.sessions {
		session.Active = false
		// TODO: Properly close Claude sessions when that API is available
	}
	b.mu.Unlock()

	return nil
}

// getSession retrieves a session by thread timestamp
func (b *SlackBot) getSession(threadTS string) (*SlackClaudeSession, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	session, exists := b.sessions[threadTS]
	return session, exists
}

// setSession stores a session by thread timestamp
func (b *SlackBot) setSession(threadTS string, session *SlackClaudeSession) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check max sessions limit
	if len(b.sessions) >= b.config.MaxSessions {
		// Remove oldest inactive session
		var oldestTS string
		var oldestTime time.Time
		for ts, s := range b.sessions {
			if !s.Active && (oldestTS == "" || s.LastActivity.Before(oldestTime)) {
				oldestTS = ts
				oldestTime = s.LastActivity
			}
		}
		if oldestTS != "" {
			delete(b.sessions, oldestTS)
			slog.Info("Removed oldest session to make room for new one", "thread_ts", oldestTS)
		}
	}

	b.sessions[threadTS] = session
}

// removeSession removes a session by thread timestamp
func (b *SlackBot) removeSession(threadTS string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.sessions, threadTS)
}

// cleanupSessions periodically removes inactive sessions
func (b *SlackBot) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.mu.Lock()
			for threadTS, session := range b.sessions {
				if time.Since(session.LastActivity) > b.config.SessionTimeout {
					delete(b.sessions, threadTS)
					session.Active = false
					slog.Info("Cleaned up inactive session",
						"thread_ts", threadTS,
						"session_id", session.SessionID,
						"idle_time", time.Since(session.LastActivity))
				}
			}
			b.mu.Unlock()

		case <-b.ctx.Done():
			return
		}
	}
}

// updateSessionActivity updates the last activity time for a session
func (b *SlackBot) updateSessionActivity(threadTS string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if session, exists := b.sessions[threadTS]; exists {
		session.LastActivity = time.Now()
	}
}

// createSessionID generates a unique session ID
func (b *SlackBot) createSessionID(userID string) (string, string) {
	sessionID := uuid.New().String()
	correlationID := fmt.Sprintf("slack-%s-%s", userID, sessionID[:8])
	return sessionID, correlationID
}
