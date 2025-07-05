package slackbot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/breadchris/flow/worklet"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// handleSlashCommand processes incoming slash commands
func (b *SlackBot) handleSlashCommand(evt *socketmode.Event, cmd *slack.SlashCommand) {
	defer b.socketMode.Ack(*evt.Request)

	switch cmd.Command {
	case "/flow":
		b.handleFlowCommand(evt, cmd)
	default:
		// Send ephemeral response for unknown commands
		response := map[string]interface{}{
			"response_type": "ephemeral",
			"text":          fmt.Sprintf("Unknown command: %s", cmd.Command),
		}
		
		payload, _ := json.Marshal(response)
		b.socketMode.Ack(*evt.Request, payload)
	}
}

// handleFlowCommand processes /flow slash commands
func (b *SlackBot) handleFlowCommand(evt *socketmode.Event, cmd *slack.SlashCommand) {
	if b.config.Debug {
		slog.Debug("Handling /flow command", 
			"user_id", cmd.UserID, 
			"channel_id", cmd.ChannelID,
			"text", cmd.Text)
	}

	// Validate that we have content to work with
	content := strings.TrimSpace(cmd.Text)
	if content == "" {
		response := map[string]interface{}{
			"response_type": "ephemeral",
			"text":          "Please provide a prompt for Claude.\nExamples:\n• `/flow Help me debug this Go code`\n• `/flow https://github.com/user/repo.git Add dark mode support`",
		}
		payload, _ := json.Marshal(response)
		b.socketMode.Ack(*evt.Request, payload)
		return
	}

	// Parse the command to check for repository URL
	repoURL, prompt := b.parseFlowCommand(content)
	
	// Send immediate response to acknowledge the command
	var responseText string
	if repoURL != "" {
		responseText = "🚀 Creating worklet for repository..."
	} else {
		responseText = "🤖 Starting Claude session..."
	}
	
	response := map[string]interface{}{
		"response_type": "in_channel",
		"text":          responseText,
	}
	_, _ = json.Marshal(response)
	
	// Create the initial message and thread
	go func() {
		// Post initial message to create thread
		_, threadTS, err := b.client.PostMessage(cmd.ChannelID,
			slack.MsgOptionText(responseText, false),
			slack.MsgOptionAsUser(true),
		)
		if err != nil {
			slog.Error("Failed to create thread", "error", err)
			return
		}

		if repoURL != "" {
			// Repository workflow - create worklet
			b.handleRepositoryWorkflow(cmd.UserID, cmd.ChannelID, threadTS, repoURL, prompt)
		} else {
			// Simple prompt workflow - direct Claude session
			b.handleSimpleWorkflow(cmd.UserID, cmd.ChannelID, threadTS, content)
		}
	}()
}

// handleEventsAPI processes Events API events
func (b *SlackBot) handleEventsAPI(evt *socketmode.Event, eventsAPIEvent *slackevents.EventsAPIEvent) {
	defer b.socketMode.Ack(*evt.Request)

	switch eventsAPIEvent.Type {
	case slackevents.CallbackEvent:
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			b.handleMessageEvent(ev)
		case *slackevents.AppMentionEvent:
			b.handleAppMentionEvent(ev)
		}
	default:
		if b.config.Debug {
			slog.Debug("Unhandled events API event", "type", eventsAPIEvent.Type)
		}
	}
}

// handleMessageEvent processes message events (including thread replies)
func (b *SlackBot) handleMessageEvent(ev *slackevents.MessageEvent) {
	// Ignore messages from bots and our own messages
	if ev.BotID != "" || ev.User == "" {
		return
	}

	// Only handle thread replies (messages with ThreadTimeStamp)
	if ev.ThreadTimeStamp == "" {
		return
	}

	// Check if this is a thread we're managing
	session, exists := b.getSession(ev.ThreadTimeStamp)
	if !exists {
		return
	}

	// Update session activity
	b.updateSessionActivity(ev.ThreadTimeStamp)

	if b.config.Debug {
		slog.Debug("Handling thread reply", 
			"user_id", ev.User,
			"channel_id", ev.Channel,
			"thread_ts", ev.ThreadTimeStamp,
			"text", ev.Text)
	}

	// Send the message to Claude
	b.sendToClaudeSession(session, ev.Text)
}

// handleAppMentionEvent processes app mention events
func (b *SlackBot) handleAppMentionEvent(ev *slackevents.AppMentionEvent) {
	// For now, treat app mentions like /flow commands
	// Remove the bot mention from the text
	text := strings.TrimSpace(ev.Text)
	
	// Remove bot mention (format: <@BOTID>)
	if strings.HasPrefix(text, "<@") {
		parts := strings.SplitN(text, ">", 2)
		if len(parts) == 2 {
			text = strings.TrimSpace(parts[1])
		}
	}

	if text == "" {
		_, _, err := b.client.PostMessage(ev.Channel,
			slack.MsgOptionText("👋 Hi! Use `/flow <your prompt>` to start a conversation with Claude.", false),
			slack.MsgOptionTS(ev.ThreadTimeStamp), // Reply in thread if mentioned in a thread
		)
		if err != nil {
			slog.Error("Failed to respond to app mention", "error", err)
		}
		return
	}

	if b.config.Debug {
		slog.Debug("Handling app mention", 
			"user_id", ev.User,
			"channel_id", ev.Channel,
			"text", text)
	}

	// Create a new thread for the Claude session
	go func() {
		_, threadTS, err := b.client.PostMessage(ev.Channel,
			slack.MsgOptionText("🤖 Starting Claude session...", false),
			slack.MsgOptionAsUser(true),
		)
		if err != nil {
			slog.Error("Failed to create thread for app mention", "error", err)
			return
		}

		// Create Claude session
		session, err := b.createClaudeSession(ev.User, ev.Channel, threadTS)
		if err != nil {
			slog.Error("Failed to create Claude session for app mention", "error", err)
			b.updateMessage(ev.Channel, threadTS, "❌ Failed to start Claude session. Please try again.")
			return
		}

		// Start Claude interaction
		b.streamClaudeInteraction(session, text)
	}()
}

// updateMessage updates a Slack message
func (b *SlackBot) updateMessage(channel, timestamp, text string) error {
	_, _, _, err := b.client.UpdateMessage(channel, timestamp,
		slack.MsgOptionText(text, false),
		slack.MsgOptionAsUser(true),
	)
	return err
}

// postMessage posts a new message to a channel/thread
func (b *SlackBot) postMessage(channel, threadTS, text string) (string, error) {
	options := []slack.MsgOption{
		slack.MsgOptionText(text, false),
		slack.MsgOptionAsUser(true),
	}
	
	if threadTS != "" {
		options = append(options, slack.MsgOptionTS(threadTS))
	}
	
	_, timestamp, err := b.client.PostMessage(channel, options...)
	return timestamp, err
}

// parseFlowCommand parses the /flow command to extract repository URL and prompt
func (b *SlackBot) parseFlowCommand(content string) (repoURL, prompt string) {
	// Regular expression to match GitHub repository URLs
	repoRegex := regexp.MustCompile(`https://github\.com/[\w\-\.]+/[\w\-\.]+(?:\.git)?`)
	
	// Check if content contains a repository URL
	match := repoRegex.FindString(content)
	if match != "" {
		// Found repository URL, extract it and the remaining text as prompt
		repoURL = match
		// Remove the repo URL from content to get the prompt
		prompt = strings.TrimSpace(strings.Replace(content, match, "", 1))
		
		// If no prompt provided, use a default
		if prompt == "" {
			prompt = "Help me understand and improve this codebase"
		}
		
		return repoURL, prompt
	}
	
	// Check for other git URL patterns (git@github.com, etc.)
	gitRegex := regexp.MustCompile(`(?:git@github\.com:|https://github\.com/)[\w\-\.]+/[\w\-\.]+(?:\.git)?`)
	match = gitRegex.FindString(content)
	if match != "" {
		repoURL = match
		prompt = strings.TrimSpace(strings.Replace(content, match, "", 1))
		if prompt == "" {
			prompt = "Help me understand and improve this codebase"
		}
		return repoURL, prompt
	}
	
	// No repository URL found, treat entire content as prompt
	return "", content
}

// handleSimpleWorkflow handles direct Claude sessions without repositories
func (b *SlackBot) handleSimpleWorkflow(userID, channelID, threadTS, prompt string) {
	// Create Claude session
	session, err := b.createClaudeSession(userID, channelID, threadTS)
	if err != nil {
		slog.Error("Failed to create Claude session", "error", err)
		_ = b.updateMessage(channelID, threadTS, "❌ Failed to start Claude session. Please try again.")
		return
	}

	// Start Claude interaction in the thread
	b.streamClaudeInteraction(session, prompt)
}

// handleRepositoryWorkflow handles worklet creation and repository-based workflows
func (b *SlackBot) handleRepositoryWorkflow(userID, channelID, threadTS, repoURL, prompt string) {
	ctx := context.Background()
	
	// Update initial message to show progress
	_ = b.updateMessage(channelID, threadTS, "🔄 Creating worklet...")
	
	// Create worklet request
	workletReq := worklet.CreateWorkletRequest{
		Name:        fmt.Sprintf("Slack Flow - %s", b.extractRepoName(repoURL)),
		Description: fmt.Sprintf("Created via Slack /flow command for user %s", userID),
		GitRepo:     repoURL,
		Branch:      "main", // Default to main branch
		BasePrompt:  prompt,
		Environment: map[string]string{
			"SLACK_USER_ID":   userID,
			"SLACK_CHANNEL":   channelID,
			"SLACK_THREAD_TS": threadTS,
		},
	}
	
	// Create worklet
	workletObj, err := b.workletManager.CreateWorklet(ctx, workletReq, userID)
	if err != nil {
		slog.Error("Failed to create worklet", "error", err)
		_ = b.updateMessage(channelID, threadTS, 
			fmt.Sprintf("❌ Failed to create worklet: %s", err.Error()))
		return
	}
	
	// Update message with worklet creation success
	_ = b.updateMessage(channelID, threadTS, 
		fmt.Sprintf("✅ Worklet created successfully!\n🆔 ID: `%s`\n🔗 Repository: %s\n\n🔄 Building and deploying...", 
			workletObj.ID, repoURL))
	
	// Start monitoring worklet status and update Slack accordingly
	go b.monitorWorkletProgress(ctx, workletObj.ID, channelID, threadTS, repoURL, prompt)
}

// extractRepoName extracts the repository name from a Git URL
func (b *SlackBot) extractRepoName(repoURL string) string {
	// Parse the URL to extract repository name
	if u, err := url.Parse(repoURL); err == nil {
		parts := strings.Split(strings.TrimSuffix(u.Path, ".git"), "/")
		if len(parts) >= 2 {
			return parts[len(parts)-1]
		}
	}
	
	// Fallback: extract from string patterns
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		return strings.TrimSuffix(name, ".git")
	}
	
	return "unknown-repo"
}

// monitorWorkletProgress monitors worklet deployment and updates Slack with progress
func (b *SlackBot) monitorWorkletProgress(ctx context.Context, workletID, channelID, threadTS, repoURL, prompt string) {
	// Poll worklet status until it's running or failed
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(10 * time.Minute) // 10 minute timeout
	
	for {
		select {
		case <-timeout:
			_ = b.updateMessage(channelID, threadTS, 
				"❌ Worklet deployment timed out after 10 minutes. Please try again.")
			return
			
		case <-ctx.Done():
			return
			
		case <-ticker.C:
			workletObj, err := b.workletManager.GetWorklet(workletID)
			if err != nil {
				slog.Error("Failed to get worklet status", "error", err)
				continue
			}
			
			switch workletObj.Status {
			case worklet.StatusRunning:
				// Worklet is ready, create PR and send link
				_ = b.updateMessage(channelID, threadTS, 
					fmt.Sprintf("🎉 Worklet is running!\n🌐 Web URL: <%s>\n\n🔄 Creating pull request...", 
						workletObj.WebURL))
				
				// Create PR for the changes
				b.createPullRequestForWorklet(ctx, workletObj, channelID, threadTS, prompt)
				return
				
			case worklet.StatusError:
				errorMsg := "❌ Worklet deployment failed"
				if workletObj.LastError != "" {
					errorMsg += fmt.Sprintf(": %s", workletObj.LastError)
				}
				_ = b.updateMessage(channelID, threadTS, errorMsg)
				return
				
			case worklet.StatusBuilding:
				_ = b.updateMessage(channelID, threadTS, 
					"🔨 Building Docker container...")
				
			case worklet.StatusDeploying:
				_ = b.updateMessage(channelID, threadTS, 
					"🚀 Deploying worklet...")
			}
		}
	}
}

// createPullRequestForWorklet creates a pull request for the worklet changes and posts the PR link to Slack
func (b *SlackBot) createPullRequestForWorklet(ctx context.Context, workletObj *worklet.Worklet, channelID, threadTS, prompt string) {
	// Generate branch name from prompt
	branchName := b.generateBranchName(prompt)
	
	// Create PR title and description
	prTitle := fmt.Sprintf("feat: %s", prompt)
	if len(prTitle) > 72 {
		prTitle = prTitle[:69] + "..."
	}
	
	prDescription := fmt.Sprintf(`## Changes Made by Claude

**Original Prompt:** %s

**Worklet ID:** %s
**Repository:** %s
**Web Preview:** %s

This pull request contains changes generated by Claude in response to the above prompt.

### Summary
Claude has analyzed the codebase and applied the requested changes. Please review the modifications carefully before merging.

---
*Generated via Slack /flow command*`, prompt, workletObj.ID, workletObj.GitRepo, workletObj.WebURL)

	// Get worklet manager to access git operations through ClaudeClient
	// We'll use the worklet's ClaudeClient which has git integration
	claudeClient := &worklet.ClaudeClient{}
	
	// Create PR using the worklet's repository path
	err := claudeClient.CreatePR(ctx, fmt.Sprintf("/tmp/worklet-repos/%s", workletObj.ID), branchName, prTitle, prDescription)
	if err != nil {
		slog.Error("Failed to create PR for worklet", "error", err, "worklet_id", workletObj.ID)
		_ = b.updateMessage(channelID, threadTS, 
			fmt.Sprintf("❌ Failed to create pull request: %s\n\n🌐 Worklet URL: <%s>", 
				err.Error(), workletObj.WebURL))
		return
	}
	
	// Success! Update message with PR link
	_ = b.updateMessage(channelID, threadTS, 
		fmt.Sprintf(`✅ **Pull Request Created Successfully!**

🔗 **Repository:** %s
🌐 **Worklet Preview:** <%s>
📝 **PR Title:** %s

The changes have been pushed to a new branch and a pull request has been created. You can review and merge the changes on GitHub.

---
*Generated via Slack /flow command*`, workletObj.GitRepo, workletObj.WebURL, prTitle))
}

// generateBranchName creates a git-safe branch name from the prompt
func (b *SlackBot) generateBranchName(prompt string) string {
	// Convert to lowercase and replace non-alphanumeric chars with hyphens
	branchName := strings.ToLower(prompt)
	branchName = regexp.MustCompile(`[^a-zA-Z0-9\s]+`).ReplaceAllString(branchName, "")
	branchName = regexp.MustCompile(`\s+`).ReplaceAllString(branchName, "-")
	branchName = strings.Trim(branchName, "-")
	
	// Limit length
	if len(branchName) > 50 {
		branchName = branchName[:50]
		branchName = strings.TrimSuffix(branchName, "-")
	}
	
	// Add prefix to indicate it's from flow
	return fmt.Sprintf("flow/%s", branchName)
}