package slackbot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/breadchris/flow/claude"
)

// createClaudeSession initializes a new Claude session for a Slack thread
func (b *SlackBot) createClaudeSession(userID, channelID, threadTS string) (*SlackClaudeSession, error) {
	sessionID, correlationID := b.createSessionID(userID)
	if err := os.MkdirAll(b.config.WorkingDirectory, 0755); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to ensure working directory: %w", err)
	}
	session := &SlackClaudeSession{
		ThreadTS:     threadTS,
		ChannelID:    channelID,
		UserID:       userID,
		SessionID:    sessionID,
		ProcessID:    correlationID,
		LastActivity: time.Now(),
		Context:      b.config.WorkingDirectory,
		Active:       true,
	}

	// Store session
	b.setSession(threadTS, session)

	if b.config.Debug {
		slog.Debug("Created Claude session",
			"thread_ts", threadTS,
			"session_id", sessionID,
			"correlation_id", correlationID)
	}

	return session, nil
}

// streamClaudeInteraction handles the bidirectional communication with Claude
func (b *SlackBot) streamClaudeInteraction(session *SlackClaudeSession, prompt string) {
	if b.config.Debug {
		slog.Debug("Starting Claude interaction",
			"session_id", session.SessionID,
			"prompt_length", len(prompt))
	}

	ctx := context.Background()

	// Create Claude session with working directory
	process, err := b.claudeService.CreateSessionWithOptions(session.Context)
	if err != nil {
		slog.Error("Failed to create Claude session", "error", err)
		b.updateMessage(session.ChannelID, session.ThreadTS,
			"❌ Failed to create Claude session. Please try again later.")
		return
	}

	// Store process reference (ProcessID is just for tracking, not used to find process)
	session.Process = process

	// Send prompt to Claude
	if err := b.claudeService.SendMessage(process, prompt); err != nil {
		slog.Error("Failed to send prompt to Claude", "error", err)
		b.updateMessage(session.ChannelID, session.ThreadTS,
			"❌ Failed to send prompt to Claude. Please try again.")
		return
	}

	if b.config.Debug {
		slog.Debug("Sent prompt to Claude, starting response stream",
			"session_id", session.SessionID,
			"prompt_length", len(prompt))
	}

	// Stream responses back to Slack
	b.handleClaudeResponseStream(ctx, process, session)
}

// handleClaudeResponseStream processes the streaming response from Claude
func (b *SlackBot) handleClaudeResponseStream(ctx context.Context, process *claude.Process, session *SlackClaudeSession) {
	// Get message channel from Claude service
	messageChan := b.claudeService.ReceiveMessages(process)
	timeout := time.After(5 * time.Minute)

	if b.config.Debug {
		slog.Debug("Starting to receive messages from Claude",
			"session_id", session.SessionID,
			"channel_available", messageChan != nil)
	}

	messageCount := 0
	for {
		select {
		case <-timeout:
			slog.Error("Claude response timeout", 
				"session_id", session.SessionID,
				"messages_received", messageCount)
			_, err := b.postMessage(session.ChannelID, session.ThreadTS, "❌ Claude response timed out. Please try again.")
			if err != nil {
				slog.Error("Failed to post timeout message", "error", err)
			}
			return

		case <-ctx.Done():
			slog.Debug("Context cancelled during Claude interaction")
			return

		case claudeMsg, ok := <-messageChan:
			messageCount++
			if !ok {
				// Channel closed - Claude finished
				if b.config.Debug {
					slog.Debug("Claude message channel closed", 
						"session_id", session.SessionID,
						"total_messages", messageCount)
				}
				return
			}

			if b.config.Debug {
				slog.Debug("Received Claude message",
					"type", claudeMsg.Type,
					"subtype", claudeMsg.Subtype,
					"session_id", claudeMsg.SessionID,
					"message_length", len(claudeMsg.Message),
					"has_result", claudeMsg.Result != "",
					"raw_message", string(claudeMsg.Message))
			}

			// Update session activity
			b.updateSessionActivity(session.ThreadTS)

			// Process different message types - post individual messages for each
			switch claudeMsg.Type {
			case "text":
				// Post text content as individual message
				if len(claudeMsg.Message) > 0 {
					textContent := string(claudeMsg.Message)
					formattedContent := b.formatClaudeResponse(textContent)
					
					_, err := b.postMessage(session.ChannelID, session.ThreadTS, formattedContent)
					if err != nil {
						slog.Error("Failed to post text message", "error", err)
					} else if b.config.Debug {
						slog.Debug("Posted text message to Slack", 
							"content_length", len(textContent))
					}
				}

			case "tool_use":
				// Post tool usage as individual message
				if claudeMsg.Subtype == "start" {
					// Tool is starting
					_, err := b.postMessage(session.ChannelID, session.ThreadTS, "🔧 _Claude is using tools..._")
					if err != nil {
						slog.Error("Failed to post tool start message", "error", err)
					} else if b.config.Debug {
						slog.Debug("Posted tool start message to Slack")
					}
				} else if claudeMsg.Subtype == "result" {
					// Tool completed - show result
					toolDisplay := b.formatToolUse(&claudeMsg)
					if toolDisplay != "" {
						_, err := b.postMessage(session.ChannelID, session.ThreadTS, toolDisplay)
						if err != nil {
							slog.Error("Failed to post tool result message", "error", err)
						} else if b.config.Debug {
							slog.Debug("Posted tool result message to Slack")
						}
					}
				} else {
					// Generic tool use message
					toolDisplay := b.formatToolUse(&claudeMsg)
					if toolDisplay != "" {
						_, err := b.postMessage(session.ChannelID, session.ThreadTS, toolDisplay)
						if err != nil {
							slog.Error("Failed to post tool message", "error", err)
						} else if b.config.Debug {
							slog.Debug("Posted tool message to Slack")
						}
					}
				}

			case "error":
				// Post error as individual message
				var errorText string
				if len(claudeMsg.Message) > 0 {
					errorText = string(claudeMsg.Message)
				} else if claudeMsg.Result != "" {
					errorText = claudeMsg.Result
				} else {
					errorText = "Unknown error occurred"
				}
				
				errorMsg := fmt.Sprintf("❌ **Error:** %s", errorText)
				_, err := b.postMessage(session.ChannelID, session.ThreadTS, errorMsg)
				if err != nil {
					slog.Error("Failed to post error message", "error", err)
				}

			case "completion":
				// Claude has finished - optionally post completion message
				if b.config.Debug {
					slog.Debug("Claude interaction completed", 
						"session_id", session.SessionID,
						"total_messages", messageCount)
				}
				// Note: Not posting a completion message to keep the conversation clean
				return

			case "system":
				// Handle system messages (like init messages)
				if b.config.Debug {
					slog.Debug("Received system message", "subtype", claudeMsg.Subtype)
				}
				// Don't forward system messages to Slack
				continue

			default:
				// Handle unknown message types
				if b.config.Debug {
					slog.Debug("Unhandled Claude message type", 
						"type", claudeMsg.Type,
						"subtype", claudeMsg.Subtype,
						"message", string(claudeMsg.Message),
						"result", claudeMsg.Result)
				}
				
				// Try to post unknown message types if they have content
				if len(claudeMsg.Message) > 0 {
					content := b.formatClaudeResponse(string(claudeMsg.Message))
					_, err := b.postMessage(session.ChannelID, session.ThreadTS, content)
					if err != nil {
						slog.Error("Failed to post unknown message type", "error", err)
					} else if b.config.Debug {
						slog.Debug("Posted unknown message type to Slack", "type", claudeMsg.Type)
					}
				}
			}
		}
	}
}

// sendToClaudeSession sends a follow-up message to an existing Claude session
func (b *SlackBot) sendToClaudeSession(session *SlackClaudeSession, message string) {
	if !session.Active {
		slog.Warn("Attempted to send message to inactive session", "session_id", session.SessionID)
		return
	}

	if b.config.Debug {
		slog.Debug("Sending follow-up to Claude session",
			"session_id", session.SessionID,
			"message_length", len(message))
	}

	go func() {
		ctx := context.Background()

		// Post immediate acknowledgment that we received the message
		_, err := b.postMessage(session.ChannelID, session.ThreadTS, "🤔 _Processing your message..._")
		if err != nil {
			slog.Error("Failed to post processing acknowledgment", "error", err)
		}

		// Use the stored Claude process for this session
		process := session.Process
		if process == nil {
			slog.Error("Claude process not found for session", "process_id", session.ProcessID)
			_, err := b.postMessage(session.ChannelID, session.ThreadTS,
				"❌ Claude session expired. Use `/flow <your message>` to start a new conversation.")
			if err != nil {
				slog.Error("Failed to post error message", "error", err)
			}
			return
		}

		// Send follow-up message to existing Claude process
		if err := b.claudeService.SendMessage(process, message); err != nil {
			slog.Error("Failed to send follow-up to Claude", "error", err)
			_, err := b.postMessage(session.ChannelID, session.ThreadTS,
				"❌ Failed to send message to Claude. Please try again, or use `/flow <your message>` to start a new conversation.")
			if err != nil {
				slog.Error("Failed to post error message", "error", err)
			}
			return
		}

		if b.config.Debug {
			slog.Debug("Sent follow-up message to Claude successfully",
				"session_id", session.SessionID,
				"message_length", len(message))
		}

		// Handle the response stream
		b.handleClaudeResponseStream(ctx, process, session)
	}()
}
