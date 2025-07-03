# Flow - Slack Bot with Claude Integration

This repository contains a Slack bot that integrates with Claude AI, allowing users to interact with Claude directly from Slack using the `/flow` command.

## Features

- **Real-time Claude Integration**: Stream Claude responses directly to Slack threads
- **Session Management**: Maintain conversation context within Slack threads
- **Thread Support**: Reply to threads to continue conversations with Claude
- **Automatic Cleanup**: Sessions timeout after inactivity to manage resources

## Prerequisites

1. **Slack App**: Create a Slack app with Socket Mode enabled
2. **Claude CLI**: Install the Claude command-line tool
3. **Go 1.23+**: Required for building the application
4. **Database**: SQLite (default) or PostgreSQL

## Setup

### 1. Slack App Configuration

1. Create a new Slack App at https://api.slack.com/apps
2. Enable Socket Mode in "Socket Mode" settings
3. Add slash command `/flow` pointing to your app
4. Configure OAuth scopes:
   - `app_mentions:read`
   - `channels:read`
   - `chat:write`
   - `commands`
   - `im:read`
   - `users:read`
5. Install app to your workspace

### 2. Environment Variables

```bash
export SLACK_APP_TOKEN=xapp-1-...     # Socket Mode token
export SLACK_BOT_TOKEN=xoxb-...       # Bot User OAuth token
export SLACK_BOT_ENABLED=true         # Enable the bot
export SLACK_BOT_DEBUG=true           # Optional: Enable debug logging
```

### 3. Configuration File

Create `data/config.json`:

```json
{
  "slack_bot_enabled": true,
  "slack_app_token": "xapp-1-...",
  "slack_bot_token": "xoxb-...",
  "session_timeout": "30m",
  "max_sessions": 10,
  "dsn": "sqlite://data/db.sqlite",
  "share_dir": "data"
}
```

### 4. Install Dependencies

```bash
go mod tidy
```

### 5. Build and Run

```bash
go build -o flow
./flow
```

## Usage

### Starting a Claude Session

In any Slack channel where the bot is present:

```
/flow Help me refactor this Go code to be more modular
```

This creates a new thread where Claude will:
1. Acknowledge the request
2. Stream the response in real-time
3. Update the message as new content arrives
4. Show tool usage (file reads, code analysis, etc.)

### Continuing the Conversation

Simply reply to the thread to send additional messages to Claude:

```
Actually, can you also add error handling?
```

### Session Management

- Sessions automatically timeout after 30 minutes of inactivity
- Each thread maintains its own Claude session and context
- Sessions are cleaned up when threads become inactive

## Architecture

### Core Components

- **SlackBot**: Main bot service managing Slack connections and Claude sessions
- **Session Management**: Bridges Slack threads with Claude sessions
- **Message Streaming**: Real-time updates to Slack messages as Claude responds
- **Event Handling**: Processes slash commands and thread replies

### Dependencies

- **Claude Service**: Manages Claude CLI processes and WebSocket communication
- **Database**: Stores session information and conversation history
- **Configuration**: Manages bot settings and credentials

## Development

### Project Structure

```
flow/
├── slackbot/           # Core slack bot implementation
├── coderunner/claude/  # Claude integration service
├── deps/              # Dependency injection
├── models/            # Database models
├── config/            # Configuration management
├── db/                # Database utilities
├── session/           # Session management
├── websocket/         # WebSocket utilities
├── data/             # Runtime data (config, database, sessions)
├── main.go           # Application entry point
└── README.md         # This file
```

### Running Tests

```bash
go test ./...
```

### Debug Mode

Enable debug logging:

```bash
export SLACK_BOT_DEBUG=true
```

This provides detailed logs of:
- Slack event processing
- Claude session management
- Message update operations
- Error conditions and retries

## Troubleshooting

### Bot not responding to commands

- Check token configuration in environment variables or config file
- Verify Socket Mode is enabled in Slack app settings
- Ensure slash command `/flow` is properly configured

### Claude sessions not working

- Verify Claude CLI is installed and available in PATH
- Check Claude configuration and API keys
- Review application logs for WebSocket connection errors

### Message updates not appearing

- Check Slack API rate limits
- Verify bot has `chat:write` permissions
- Review error logs for API failures

## License

This project is licensed under the MIT License.
