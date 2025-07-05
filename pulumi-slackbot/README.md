# Slackbot Lambda Deployment

This Pulumi project deploys a Slackbot as AWS Lambda functions with API Gateway, WebSocket support, and Claude integration.

## Architecture

### Components
- **Lambda Functions**: Serverless execution for Slackbot and Claude sessions
- **API Gateway**: HTTP endpoints for Slack webhook events
- **WebSocket API Gateway**: Real-time communication for Claude sessions
- **DynamoDB**: Session storage and persistence
- **S3**: File storage for Claude session outputs
- **IAM Roles**: Secure access management

### Key Features
- **Event-driven**: Responds to Slack events via webhooks (no persistent connections)
- **Stateless**: Session state stored in DynamoDB
- **Scalable**: Auto-scaling Lambda functions
- **Cost-effective**: Pay-per-use pricing model
- **Secure**: IAM-based access control

## Prerequisites

1. **AWS CLI** configured with appropriate credentials
2. **Pulumi CLI** installed
3. **Go 1.23** or later
4. **Slack App** configured with:
   - Bot Token (`xoxb-...`)
   - Signing Secret
   - Event subscriptions enabled
   - Slash commands configured

## Setup

### 1. Configure Slack App

1. Create a new Slack App at https://api.slack.com/apps
2. Enable the following bot token scopes:
   - `chat:write`
   - `commands`
   - `app_mentions:read`
   - `channels:read`
   - `groups:read`
   - `im:read`
   - `mpim:read`
3. Enable Event Subscriptions and subscribe to:
   - `app_mention`
   - `message.channels`
   - `message.groups`
   - `message.im`
   - `message.mpim`
4. Create a `/flow` slash command (URL will be provided after deployment)

### 2. Deploy Infrastructure

```bash
# Initialize Pulumi stack
pulumi stack init dev

# Set required configuration
pulumi config set aws:region us-east-1
pulumi config set slackbot:slackBotToken "xoxb-your-bot-token" --secret
pulumi config set slackbot:slackSigningSecret "your-signing-secret" --secret
pulumi config set slackbot:claudeApiKey "your-claude-api-key" --secret

# Optional configuration
pulumi config set slackbot:s3Bucket "my-slackbot-sessions"
pulumi config set slackbot:workDirectory "/tmp/claude-sessions"

# Build Lambda functions
chmod +x build.sh
./build.sh

# Deploy infrastructure
pulumi up
```

### 3. Configure Slack Webhooks

After deployment, update your Slack app configuration:

1. Get the API Gateway URL from Pulumi output: `slackApiUrl`
2. Set this URL as:
   - Event subscriptions request URL: `{slackApiUrl}/slack`
   - Slash command request URL: `{slackApiUrl}/slack`

## Usage

### Slack Commands

#### `/flow` Command
Start a Claude session directly from Slack:

```
/flow Help me debug this Go code
/flow https://github.com/user/repo.git Add dark mode support
/flow Explain how to optimize this database query
```

#### App Mentions
Mention the bot in any channel:

```
@slackbot Help me understand this error message
@slackbot Can you review this code snippet?
```

### WebSocket Claude Sessions

For advanced use cases, you can connect directly to the Claude session WebSocket:

```javascript
const ws = new WebSocket('wss://your-websocket-url/prod?user_id=your-user-id');

// Send message to Claude
ws.send(JSON.stringify({
    action: 'send_message',
    data: {
        message: 'Hello Claude!'
    }
}));

// Handle responses
ws.onmessage = (event) => {
    const response = JSON.parse(event.data);
    console.log('Claude response:', response);
};
```

## Configuration

### Environment Variables

The Lambda functions use the following environment variables:

- `SLACK_BOT_TOKEN`: Slack bot token
- `SLACK_SIGNING_SECRET`: Slack signing secret
- `CLAUDE_API_KEY`: Claude API key
- `DYNAMODB_TABLE`: DynamoDB table name
- `S3_BUCKET`: S3 bucket name
- `WORK_DIRECTORY`: Working directory for Claude sessions

### Pulumi Configuration

```yaml
# Pulumi.dev.yaml
config:
  aws:region: us-east-1
  slackbot:slackBotToken: "xoxb-your-bot-token"
  slackbot:slackSigningSecret: "your-signing-secret"
  slackbot:claudeApiKey: "your-claude-api-key"
  slackbot:s3Bucket: "my-slackbot-sessions"
  slackbot:workDirectory: "/tmp/claude-sessions"
```

## Development

### Building Lambda Functions

```bash
# Build all Lambda functions
./build.sh

# Build individual functions
cd lambda
GOOS=linux GOARCH=amd64 go build -o slackbot-lambda main.go

cd claude-session
GOOS=linux GOARCH=amd64 go build -o claude-session-lambda main.go
```

### Local Testing

```bash
# Test Lambda functions locally (requires AWS SAM CLI)
sam local start-api --template-file template.yaml

# Test WebSocket functions
sam local start-lambda --template-file template.yaml
```

### Debugging

1. **CloudWatch Logs**: Check Lambda function logs in CloudWatch
2. **DynamoDB**: Monitor session storage in DynamoDB console
3. **S3**: Check uploaded session files in S3 console
4. **API Gateway**: Monitor API usage and errors

## Security

### IAM Permissions

The Lambda functions have minimal required permissions:
- DynamoDB: Read/write access to session table
- S3: Read/write access to session bucket
- CloudWatch: Write access for logging

### Slack Security

- Request signature verification enabled
- Bot token scope restrictions
- Channel whitelist support (if needed)

## Monitoring

### CloudWatch Metrics

- Lambda execution duration
- API Gateway request count
- DynamoDB read/write capacity
- S3 storage usage

### Alarms

Set up CloudWatch alarms for:
- Lambda function errors
- API Gateway 5xx errors
- DynamoDB throttling
- High S3 storage usage

## Cost Optimization

### Lambda
- Use appropriate memory allocation (256MB default)
- Optimize function timeout (30s for Slackbot, 15min for Claude)
- Consider Reserved Concurrency for cost control

### DynamoDB
- Use On-Demand billing for variable workloads
- Implement TTL for automatic session cleanup
- Monitor read/write capacity usage

### S3
- Use Standard storage class for recent sessions
- Implement lifecycle policies for archival
- Enable S3 versioning for data protection

## Troubleshooting

### Common Issues

1. **Slack verification failed**: Check signing secret configuration
2. **Lambda timeout**: Increase timeout or optimize function
3. **DynamoDB throttling**: Increase capacity or use On-Demand
4. **S3 access denied**: Check IAM permissions

### Error Messages

```bash
# Check Lambda logs
aws logs tail /aws/lambda/slackbot-lambda --follow

# Check DynamoDB metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name UserErrors \
  --dimensions Name=TableName,Value=slackbot-sessions \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-01T23:59:59Z \
  --period 300 \
  --statistics Sum
```

## Deployment

### Production Deployment

```bash
# Create production stack
pulumi stack init prod

# Set production configuration
pulumi config set aws:region us-east-1
pulumi config set slackbot:slackBotToken "xoxb-prod-token" --secret
pulumi config set slackbot:slackSigningSecret "prod-signing-secret" --secret
pulumi config set slackbot:claudeApiKey "prod-claude-api-key" --secret

# Deploy to production
pulumi up --stack prod
```

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
name: Deploy Slackbot
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: 1.23
      - name: Build Lambda functions
        run: ./build.sh
      - name: Deploy with Pulumi
        uses: pulumi/actions@v3
        with:
          command: up
          stack-name: prod
        env:
          PULUMI_ACCESS_TOKEN: ${{ secrets.PULUMI_ACCESS_TOKEN }}
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.