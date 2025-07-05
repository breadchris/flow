package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// Environment variables
var (
	slackBotToken      = os.Getenv("SLACK_BOT_TOKEN")
	slackSigningSecret = os.Getenv("SLACK_SIGNING_SECRET")
	claudeApiKey       = os.Getenv("CLAUDE_API_KEY")
	dynamoDBTable      = os.Getenv("DYNAMODB_TABLE")
	s3Bucket           = os.Getenv("S3_BUCKET")
	workDirectory      = os.Getenv("WORK_DIRECTORY")
)

// AWS clients
var (
	dynamoClient *dynamodb.DynamoDB
	s3Client     *s3.S3
	slackClient  *slack.Client
)

// SlackSession represents a Claude session stored in DynamoDB
type SlackSession struct {
	SessionID    string    `json:"session_id" dynamodb:"sessionId"`
	ThreadID     string    `json:"thread_id" dynamodb:"threadId"`
	ChannelID    string    `json:"channel_id" dynamodb:"channelId"`
	UserID       string    `json:"user_id" dynamodb:"userId"`
	LastActivity time.Time `json:"last_activity" dynamodb:"lastActivity"`
	Context      string    `json:"context" dynamodb:"context"`
	Active       bool      `json:"active" dynamodb:"active"`
	ProcessID    string    `json:"process_id" dynamodb:"processId"`
}

// SlackEvent represents a parsed Slack event
type SlackEvent struct {
	Type      string      `json:"type"`
	Challenge string      `json:"challenge,omitempty"`
	Event     interface{} `json:"event,omitempty"`
}

func init() {
	// Initialize AWS session
	sess := session.Must(session.NewSession())
	dynamoClient = dynamodb.New(sess)
	s3Client = s3.New(sess)
	slackClient = slack.New(slackBotToken)
}

// handleRequest processes incoming Lambda requests
func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received request: %s %s", request.HTTPMethod, request.Path)

	// Verify Slack request signature
	if !verifySlackSignature(request) {
		log.Printf("Invalid Slack signature")
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       "Unauthorized",
		}, nil
	}

	// Parse request body
	var slackEvent SlackEvent
	if err := json.Unmarshal([]byte(request.Body), &slackEvent); err != nil {
		log.Printf("Failed to parse request body: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	// Handle URL verification challenge
	if slackEvent.Type == "url_verification" {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       slackEvent.Challenge,
		}, nil
	}

	// Handle events
	if slackEvent.Type == "event_callback" {
		eventData, _ := json.Marshal(slackEvent.Event)
		var eventsAPIEvent slackevents.EventsAPIEvent
		if err := json.Unmarshal([]byte(request.Body), &eventsAPIEvent); err != nil {
			log.Printf("Failed to parse events API event: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       "Bad Request",
			}, nil
		}

		// Process the event asynchronously
		go processSlackEvent(ctx, &eventsAPIEvent)
	}

	// Handle slash commands
	if request.HTTPMethod == "POST" && strings.Contains(request.Headers["content-type"], "application/x-www-form-urlencoded") {
		return handleSlashCommand(ctx, request)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "OK",
	}, nil
}

// verifySlackSignature verifies the Slack request signature
func verifySlackSignature(request events.APIGatewayProxyRequest) bool {
	if slackSigningSecret == "" {
		return true // Skip verification if no secret is set
	}

	timestamp := request.Headers["x-slack-request-timestamp"]
	signature := request.Headers["x-slack-signature"]

	if timestamp == "" || signature == "" {
		return false
	}

	// Check timestamp to prevent replay attacks
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	if time.Now().Unix()-ts > 300 { // 5 minutes
		return false
	}

	// Calculate expected signature
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, request.Body)
	h := hmac.New(sha256.New, []byte(slackSigningSecret))
	h.Write([]byte(baseString))
	expectedSignature := "v0=" + hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// handleSlashCommand processes slash commands
func handleSlashCommand(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse form data
	values := make(map[string]string)
	for key, value := range request.MultiValueQueryStringParameters {
		if len(value) > 0 {
			values[key] = value[0]
		}
	}

	// Extract command data
	command := values["command"]
	text := values["text"]
	userID := values["user_id"]
	channelID := values["channel_id"]

	log.Printf("Slash command: %s, text: %s, user: %s, channel: %s", command, text, userID, channelID)

	if command == "/flow" {
		// Process flow command asynchronously
		go processFlowCommand(ctx, userID, channelID, text)

		// Return immediate response
		response := map[string]interface{}{
			"response_type": "in_channel",
			"text":          "🤖 Starting Claude session...",
		}

		responseBody, _ := json.Marshal(response)
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: string(responseBody),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Unknown command",
	}, nil
}

// processSlackEvent processes Slack events
func processSlackEvent(ctx context.Context, event *slackevents.EventsAPIEvent) {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			handleMessageEvent(ctx, ev)
		case *slackevents.AppMentionEvent:
			handleAppMentionEvent(ctx, ev)
		}
	}
}

// handleMessageEvent processes message events
func handleMessageEvent(ctx context.Context, ev *slackevents.MessageEvent) {
	// Ignore bot messages
	if ev.BotID != "" || ev.User == "" {
		return
	}

	// Only handle thread replies
	if ev.ThreadTimeStamp == "" {
		return
	}

	// Check if we have a session for this thread
	session, err := getSession(ctx, ev.ThreadTimeStamp, ev.Channel)
	if err != nil || session == nil {
		return
	}

	// Update session activity
	session.LastActivity = time.Now()
	saveSession(ctx, session)

	// Send message to Claude
	sendToClaudeSession(ctx, session, ev.Text)
}

// handleAppMentionEvent processes app mention events
func handleAppMentionEvent(ctx context.Context, ev *slackevents.AppMentionEvent) {
	// Remove bot mention from text
	text := strings.TrimSpace(ev.Text)
	if strings.HasPrefix(text, "<@") {
		parts := strings.SplitN(text, ">", 2)
		if len(parts) == 2 {
			text = strings.TrimSpace(parts[1])
		}
	}

	if text == "" {
		slackClient.PostMessage(ev.Channel,
			slack.MsgOptionText("👋 Hi! Use `/flow <your prompt>` to start a conversation with Claude.", false),
			slack.MsgOptionTS(ev.ThreadTimeStamp))
		return
	}

	// Create new Claude session
	_, threadTS, err := slackClient.PostMessage(ev.Channel,
		slack.MsgOptionText("🤖 Starting Claude session...", false))
	if err != nil {
		log.Printf("Failed to create thread: %v", err)
		return
	}

	session := &SlackSession{
		SessionID:    generateSessionID(),
		ThreadID:     threadTS,
		ChannelID:    ev.Channel,
		UserID:       ev.User,
		LastActivity: time.Now(),
		Active:       true,
		ProcessID:    generateProcessID(),
	}

	if err := saveSession(ctx, session); err != nil {
		log.Printf("Failed to save session: %v", err)
		return
	}

	// Start Claude interaction
	go sendToClaudeSession(ctx, session, text)
}

// processFlowCommand processes /flow slash commands
func processFlowCommand(ctx context.Context, userID, channelID, text string) {
	// Create thread
	_, threadTS, err := slackClient.PostMessage(channelID,
		slack.MsgOptionText("🤖 Starting Claude session...", false))
	if err != nil {
		log.Printf("Failed to create thread: %v", err)
		return
	}

	// Create session
	session := &SlackSession{
		SessionID:    generateSessionID(),
		ThreadID:     threadTS,
		ChannelID:    channelID,
		UserID:       userID,
		LastActivity: time.Now(),
		Active:       true,
		ProcessID:    generateProcessID(),
	}

	if err := saveSession(ctx, session); err != nil {
		log.Printf("Failed to save session: %v", err)
		return
	}

	// Start Claude interaction
	sendToClaudeSession(ctx, session, text)
}

// sendToClaudeSession sends a message to Claude and streams response back to Slack
func sendToClaudeSession(ctx context.Context, session *SlackSession, message string) {
	// Create working directory
	workDir := fmt.Sprintf("%s/%s", workDirectory, session.SessionID)
	
	// For now, simulate Claude response
	// In a real implementation, you would:
	// 1. Create a Claude session
	// 2. Send the message to Claude
	// 3. Stream the response back to Slack
	// 4. Handle tool usage and file operations
	// 5. Upload results to S3
	
	response := fmt.Sprintf("🤖 Claude received your message: %s\n\n*This is a demo response. In the full implementation, Claude would process your request and provide a detailed response.*", message)
	
	// Update the message in Slack
	slackClient.UpdateMessage(session.ChannelID, session.ThreadID,
		slack.MsgOptionText(response, false))
	
	// Simulate file upload to S3
	if err := uploadToS3(session.SessionID, workDir, "Demo session completed"); err != nil {
		log.Printf("Failed to upload to S3: %v", err)
	}
}

// getSession retrieves a session from DynamoDB
func getSession(ctx context.Context, threadID, channelID string) (*SlackSession, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(dynamoDBTable),
		Key: map[string]*dynamodb.AttributeValue{
			"sessionId": {
				S: aws.String(fmt.Sprintf("%s-%s", channelID, threadID)),
			},
			"threadId": {
				S: aws.String(threadID),
			},
		},
	}

	result, err := dynamoClient.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	var session SlackSession
	if err := dynamodbattribute.UnmarshalMap(result.Item, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// saveSession saves a session to DynamoDB
func saveSession(ctx context.Context, session *SlackSession) error {
	item, err := dynamodbattribute.MarshalMap(session)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(dynamoDBTable),
		Item:      item,
	}

	_, err = dynamoClient.PutItemWithContext(ctx, input)
	return err
}

// uploadToS3 uploads session data to S3
func uploadToS3(sessionID, workDir, content string) error {
	key := fmt.Sprintf("sessions/%s/session.txt", sessionID)
	
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s3Bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(content),
		ContentType: aws.String("text/plain"),
	}

	_, err := s3Client.PutObject(input)
	if err != nil {
		return err
	}

	log.Printf("Uploaded session to S3: s3://%s/%s", s3Bucket, key)
	return nil
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

// generateProcessID generates a unique process ID
func generateProcessID() string {
	return fmt.Sprintf("process-%d", time.Now().UnixNano())
}

func main() {
	lambda.Start(handleRequest)
}