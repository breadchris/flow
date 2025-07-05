package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
)

// Environment variables
var (
	claudeApiKey    = os.Getenv("CLAUDE_API_KEY")
	dynamoDBTable   = os.Getenv("DYNAMODB_TABLE")
	s3Bucket        = os.Getenv("S3_BUCKET")
	workDirectory   = os.Getenv("WORK_DIRECTORY")
)

// AWS clients
var (
	dynamoClient *dynamodb.DynamoDB
	s3Client     *s3.S3
	s3Uploader   *s3manager.Uploader
)

// ClaudeSession represents a Claude session
type ClaudeSession struct {
	SessionID    string    `json:"session_id" dynamodb:"sessionId"`
	ConnectionID string    `json:"connection_id" dynamodb:"connectionId"`
	UserID       string    `json:"user_id" dynamodb:"userId"`
	WorkDir      string    `json:"work_dir" dynamodb:"workDir"`
	CreatedAt    time.Time `json:"created_at" dynamodb:"createdAt"`
	LastActivity time.Time `json:"last_activity" dynamodb:"lastActivity"`
	Active       bool      `json:"active" dynamodb:"active"`
	Context      string    `json:"context" dynamodb:"context"`
}

// WebSocketMessage represents a message sent via WebSocket
type WebSocketMessage struct {
	Action  string      `json:"action"`
	Data    interface{} `json:"data"`
	Session string      `json:"session"`
}

// ClaudeRequest represents a request to Claude
type ClaudeRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
	WorkDir   string `json:"work_dir"`
}

// ClaudeResponse represents a response from Claude
type ClaudeResponse struct {
	Content   string `json:"content"`
	SessionID string `json:"session_id"`
	ToolUse   []Tool `json:"tool_use,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Tool represents a tool used by Claude
type Tool struct {
	Type   string      `json:"type"`
	Name   string      `json:"name"`
	Input  interface{} `json:"input"`
	Output string      `json:"output"`
}

func init() {
	// Initialize AWS session
	sess := session.Must(session.NewSession())
	dynamoClient = dynamodb.New(sess)
	s3Client = s3.New(sess)
	s3Uploader = s3manager.NewUploader(sess)
}

// handleRequest processes incoming Lambda requests for Claude sessions
func handleRequest(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("WebSocket request: %s %s", request.RequestContext.RouteKey, request.RequestContext.ConnectionID)

	switch request.RequestContext.RouteKey {
	case "$connect":
		return handleConnect(ctx, request)
	case "$disconnect":
		return handleDisconnect(ctx, request)
	case "$default":
		return handleMessage(ctx, request)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Unknown route",
		}, nil
	}
}

// handleConnect handles WebSocket connection
func handleConnect(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("WebSocket connection established: %s", request.RequestContext.ConnectionID)
	
	// Extract user ID from query parameters
	userID := request.QueryStringParameters["user_id"]
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Missing user_id parameter",
		}, nil
	}

	// Create new session
	session := &ClaudeSession{
		SessionID:    uuid.New().String(),
		ConnectionID: request.RequestContext.ConnectionID,
		UserID:       userID,
		WorkDir:      fmt.Sprintf("%s/%s", workDirectory, uuid.New().String()),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Active:       true,
		Context:      "New Claude session started",
	}

	if err := saveClaudeSession(ctx, session); err != nil {
		log.Printf("Failed to save session: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Failed to create session",
		}, nil
	}

	// Send welcome message
	welcomeMsg := WebSocketMessage{
		Action: "session_created",
		Data: map[string]interface{}{
			"session_id": session.SessionID,
			"work_dir":   session.WorkDir,
			"message":    "Claude session created successfully. You can now send messages to Claude.",
		},
		Session: session.SessionID,
	}

	if err := sendWebSocketMessage(ctx, request.RequestContext.ConnectionID, welcomeMsg); err != nil {
		log.Printf("Failed to send welcome message: %v", err)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// handleDisconnect handles WebSocket disconnection
func handleDisconnect(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("WebSocket disconnection: %s", request.RequestContext.ConnectionID)

	// Find and deactivate session
	session, err := getClaudeSessionByConnectionID(ctx, request.RequestContext.ConnectionID)
	if err != nil {
		log.Printf("Failed to find session for connection: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	if session != nil {
		session.Active = false
		session.LastActivity = time.Now()
		
		if err := saveClaudeSession(ctx, session); err != nil {
			log.Printf("Failed to update session: %v", err)
		}

		// Upload session data to S3
		if err := uploadSessionToS3(ctx, session); err != nil {
			log.Printf("Failed to upload session to S3: %v", err)
		}
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// handleMessage handles incoming WebSocket messages
func handleMessage(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("WebSocket message received: %s", request.Body)

	var msg WebSocketMessage
	if err := json.Unmarshal([]byte(request.Body), &msg); err != nil {
		log.Printf("Failed to parse message: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}

	// Get session
	session, err := getClaudeSessionByConnectionID(ctx, request.RequestContext.ConnectionID)
	if err != nil {
		log.Printf("Failed to find session: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 404}, nil
	}

	if session == nil {
		log.Printf("Session not found for connection: %s", request.RequestContext.ConnectionID)
		return events.APIGatewayProxyResponse{StatusCode: 404}, nil
	}

	// Update session activity
	session.LastActivity = time.Now()
	saveClaudeSession(ctx, session)

	// Process message based on action
	switch msg.Action {
	case "send_message":
		return handleClaudeMessage(ctx, request.RequestContext.ConnectionID, session, msg)
	case "get_session_info":
		return handleGetSessionInfo(ctx, request.RequestContext.ConnectionID, session)
	case "upload_session":
		return handleUploadSession(ctx, request.RequestContext.ConnectionID, session)
	default:
		log.Printf("Unknown action: %s", msg.Action)
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}
}

// handleClaudeMessage processes messages to Claude
func handleClaudeMessage(ctx context.Context, connectionID string, session *ClaudeSession, msg WebSocketMessage) (events.APIGatewayProxyResponse, error) {
	// Extract message data
	dataMap, ok := msg.Data.(map[string]interface{})
	if !ok {
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}

	message, ok := dataMap["message"].(string)
	if !ok {
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}

	// Process message with Claude (simulated for now)
	go processClaudeMessage(ctx, connectionID, session, message)

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// processClaudeMessage processes a message with Claude
func processClaudeMessage(ctx context.Context, connectionID string, session *ClaudeSession, message string) {
	// Send "thinking" message
	thinkingMsg := WebSocketMessage{
		Action: "claude_thinking",
		Data: map[string]interface{}{
			"message": "Claude is processing your request...",
		},
		Session: session.SessionID,
	}
	sendWebSocketMessage(ctx, connectionID, thinkingMsg)

	// Simulate Claude processing
	// In a real implementation, you would:
	// 1. Send the message to Claude API
	// 2. Handle streaming responses
	// 3. Process tool usage
	// 4. Manage file operations in the work directory
	// 5. Stream responses back to the client

	response := fmt.Sprintf("🤖 Claude received your message: %s\n\n*This is a demo response. In the full implementation, Claude would process your request, potentially use tools like file operations, code execution, and provide detailed responses.*\n\nWork directory: %s", message, session.WorkDir)

	// Send response
	responseMsg := WebSocketMessage{
		Action: "claude_response",
		Data: map[string]interface{}{
			"content":    response,
			"session_id": session.SessionID,
			"timestamp":  time.Now().Format(time.RFC3339),
		},
		Session: session.SessionID,
	}
	sendWebSocketMessage(ctx, connectionID, responseMsg)

	// Update session context
	session.Context = fmt.Sprintf("Last message: %s", message)
	session.LastActivity = time.Now()
	saveClaudeSession(ctx, session)
}

// handleGetSessionInfo returns session information
func handleGetSessionInfo(ctx context.Context, connectionID string, session *ClaudeSession) (events.APIGatewayProxyResponse, error) {
	infoMsg := WebSocketMessage{
		Action: "session_info",
		Data: map[string]interface{}{
			"session_id":    session.SessionID,
			"work_dir":      session.WorkDir,
			"created_at":    session.CreatedAt.Format(time.RFC3339),
			"last_activity": session.LastActivity.Format(time.RFC3339),
			"active":        session.Active,
			"context":       session.Context,
		},
		Session: session.SessionID,
	}

	sendWebSocketMessage(ctx, connectionID, infoMsg)
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// handleUploadSession uploads session data to S3
func handleUploadSession(ctx context.Context, connectionID string, session *ClaudeSession) (events.APIGatewayProxyResponse, error) {
	if err := uploadSessionToS3(ctx, session); err != nil {
		errorMsg := WebSocketMessage{
			Action: "upload_error",
			Data: map[string]interface{}{
				"error": fmt.Sprintf("Failed to upload session: %v", err),
			},
			Session: session.SessionID,
		}
		sendWebSocketMessage(ctx, connectionID, errorMsg)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	successMsg := WebSocketMessage{
		Action: "upload_success",
		Data: map[string]interface{}{
			"message": fmt.Sprintf("Session uploaded to S3: s3://%s/sessions/%s/", s3Bucket, session.SessionID),
			"s3_path": fmt.Sprintf("s3://%s/sessions/%s/", s3Bucket, session.SessionID),
		},
		Session: session.SessionID,
	}
	sendWebSocketMessage(ctx, connectionID, successMsg)
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// sendWebSocketMessage sends a message via WebSocket
func sendWebSocketMessage(ctx context.Context, connectionID string, message WebSocketMessage) error {
	// This would typically use API Gateway Management API to send messages
	// For now, we'll log the message
	messageJSON, _ := json.Marshal(message)
	log.Printf("Sending WebSocket message to %s: %s", connectionID, string(messageJSON))
	return nil
}

// getClaudeSessionByConnectionID retrieves a session by connection ID
func getClaudeSessionByConnectionID(ctx context.Context, connectionID string) (*ClaudeSession, error) {
	// Query DynamoDB for session with this connection ID
	input := &dynamodb.ScanInput{
		TableName: aws.String(dynamoDBTable),
		FilterExpression: aws.String("connectionId = :conn_id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":conn_id": {
				S: aws.String(connectionID),
			},
		},
	}

	result, err := dynamoClient.ScanWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	var session ClaudeSession
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// saveClaudeSession saves a session to DynamoDB
func saveClaudeSession(ctx context.Context, session *ClaudeSession) error {
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

// uploadSessionToS3 uploads session data to S3
func uploadSessionToS3(ctx context.Context, session *ClaudeSession) error {
	// Create session summary
	sessionData := map[string]interface{}{
		"session_id":    session.SessionID,
		"user_id":       session.UserID,
		"work_dir":      session.WorkDir,
		"created_at":    session.CreatedAt.Format(time.RFC3339),
		"last_activity": session.LastActivity.Format(time.RFC3339),
		"context":       session.Context,
		"active":        session.Active,
	}

	// Convert to JSON
	data, err := json.MarshalIndent(sessionData, "", "  ")
	if err != nil {
		return err
	}

	// Upload to S3
	key := fmt.Sprintf("sessions/%s/session-info.json", session.SessionID)
	input := &s3manager.UploadInput{
		Bucket:      aws.String(s3Bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(string(data)),
		ContentType: aws.String("application/json"),
	}

	_, err = s3Uploader.UploadWithContext(ctx, input)
	if err != nil {
		return err
	}

	log.Printf("Uploaded session to S3: s3://%s/%s", s3Bucket, key)
	return nil
}

func main() {
	lambda.Start(handleRequest)
}