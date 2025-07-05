package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	Debug    bool
	DebugDir string
	Tools    []string
}

type Service struct {
	config   Config
	sessions map[string]*Process
	mu       sync.RWMutex
}

type Process struct {
	sessionID     string
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	stdoutScanner *bufio.Scanner
	stderrScanner *bufio.Scanner
	ctx           context.Context
	cancel        context.CancelFunc
	startTime     time.Time
	correlationID string
	debugDir      string
	stdinLogFile  *os.File
	stdoutLogFile *os.File
	stderrLogFile *os.File
	isHealthy     bool
	lastHeartbeat time.Time
	inputChan     chan Input   // Channel for sending messages to Claude
	outputChan    chan Message // Channel for receiving messages from Claude
	initComplete  chan bool    // Signal when initialization is complete
	errorChan     chan Message // Channel for forwarding stderr errors
}

// Message represents a message from Claude CLI
type Message struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	Message   json.RawMessage `json:"message,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	ParentID  string          `json:"parent_tool_use_id,omitempty"`
	Result    string          `json:"result,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type Input struct {
	Type    string       `json:"type"`
	Message InputMessage `json:"message"`
}

type InputMessage struct {
	Role    string                `json:"role"`
	Content []InputMessageContent `json:"content"`
}

type InputMessageContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewService(config Config) *Service {
	// Set default values if not provided
	if len(config.Tools) == 0 {
		config.Tools = []string{"Read", "Write", "Bash"}
	}
	if config.DebugDir == "" {
		config.DebugDir = "/tmp/worklet"
	}

	return &Service{
		config:   config,
		sessions: make(map[string]*Process),
	}
}

// createDebugDirectory creates debug logging directory if debug mode is enabled
func (s *Service) createDebugDirectory(correlationID string) (string, error) {
	if !s.config.Debug {
		return "", nil
	}

	debugDir := filepath.Join(s.config.DebugDir, correlationID)
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create debug directory: %w", err)
	}
	return debugDir, nil
}

// formatUserError converts Claude CLI stderr messages into user-friendly error messages
func (s *Service) formatUserError(stderrLine string) string {
	lowerLine := strings.ToLower(stderrLine)

	// Handle JSON parsing errors
	if strings.Contains(lowerLine, "syntaxerror") && strings.Contains(lowerLine, "json") {
		if strings.Contains(stderrLine, "\"asdf\"") || strings.Contains(stderrLine, "'asdf'") {
			return "Invalid input format. Please provide a valid question or command instead of random text."
		}
		return "Invalid input format. Please ensure your input is properly formatted text or valid JSON."
	}

	// Handle other common errors
	if strings.Contains(lowerLine, "parsing") && strings.Contains(lowerLine, "error") {
		return "Unable to process your input. Please check the format and try again."
	}

	if strings.Contains(lowerLine, "timeout") {
		return "Request timed out. Please try again or simplify your request."
	}

	if strings.Contains(lowerLine, "failed") {
		return "Command failed to execute. Please check your input and try again."
	}

	// Default error message for unrecognized errors
	return "An error occurred while processing your request. Please try again."
}

// openDebugFiles opens debug log files for stdin, stdout, and stderr
func (s *Service) openDebugFiles(debugDir string) (*os.File, *os.File, *os.File, error) {
	if debugDir == "" {
		return nil, nil, nil, nil
	}

	stdinFile, err := os.Create(filepath.Join(debugDir, "stdin.log"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdin log file: %w", err)
	}

	stdoutFile, err := os.Create(filepath.Join(debugDir, "stdout.log"))
	if err != nil {
		stdinFile.Close()
		return nil, nil, nil, fmt.Errorf("failed to create stdout log file: %w", err)
	}

	stderrFile, err := os.Create(filepath.Join(debugDir, "stderr.log"))
	if err != nil {
		stdinFile.Close()
		stdoutFile.Close()
		return nil, nil, nil, fmt.Errorf("failed to create stderr log file: %w", err)
	}

	return stdinFile, stdoutFile, stderrFile, nil
}

// closeDebugFiles safely closes all debug files
func (process *Process) closeDebugFiles() {
	if process.stdinLogFile != nil {
		process.stdinLogFile.Close()
	}
	if process.stdoutLogFile != nil {
		process.stdoutLogFile.Close()
	}
	if process.stderrLogFile != nil {
		process.stderrLogFile.Close()
	}
}

// logToDebugFile writes data to a debug file if it exists
func (process *Process) logToDebugFile(file *os.File, prefix string, data []byte) {
	if file != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		line := fmt.Sprintf("[%s] %s: %s\n", timestamp, prefix, string(data))
		file.WriteString(line)
		file.Sync() // Ensure data is written immediately
	}
}

// validateProcessHealth checks if the Claude process is still healthy
func (process *Process) validateProcessHealth() bool {
	if process.cmd == nil || process.cmd.Process == nil {
		return false
	}

	// Check if process is still running
	if err := process.cmd.Process.Signal(os.Signal(nil)); err != nil {
		return false
	}

	process.lastHeartbeat = time.Now()
	process.isHealthy = true
	return true
}

// monitorStderr monitors stderr output from the Claude process
func (s *Service) monitorStderr(process *Process) {
	slog.Debug("Starting stderr monitoring",
		"correlation_id", process.correlationID,
		"session_id", process.sessionID,
		"action", "stderr_monitor_start",
	)

	stderrLineCount := 0
	for process.stderrScanner.Scan() {
		line := process.stderrScanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		stderrLineCount++

		// Log to debug file if enabled
		process.logToDebugFile(process.stderrLogFile, "STDERR", []byte(line))

		// Log stderr messages with high priority since they often indicate issues
		slog.Warn("Claude stderr output",
			"correlation_id", process.correlationID,
			"session_id", process.sessionID,
			"stderr_line", line,
			"stderr_line_count", stderrLineCount,
			"action", "stderr_received",
		)

		// Check for specific error patterns that might indicate process health issues
		if strings.Contains(strings.ToLower(line), "error") ||
			strings.Contains(strings.ToLower(line), "failed") ||
			strings.Contains(strings.ToLower(line), "timeout") {

			slog.Error("Critical error detected in Claude stderr",
				"correlation_id", process.correlationID,
				"session_id", process.sessionID,
				"error_line", line,
				"action", "stderr_critical_error",
			)

			// Create user-friendly error message
			userError := s.formatUserError(line)
			errorMsg := Message{
				Type:      "error",
				Subtype:   "process_error",
				SessionID: process.sessionID,
				Message: json.RawMessage(fmt.Sprintf(`{"error": "%s", "source": "claude_process", "timestamp": "%s", "details": "%s"}`,
					userError, time.Now().Format(time.RFC3339), strings.ReplaceAll(line, `"`, `\"`))),
				IsError: true,
			}

			// Send error to error channel (non-blocking)
			select {
			case process.errorChan <- errorMsg:
				slog.Debug("Error message sent to error channel",
					"correlation_id", process.correlationID,
					"session_id", process.sessionID,
					"action", "error_forwarded",
				)
			default:
				slog.Warn("Error channel full, dropping error message",
					"correlation_id", process.correlationID,
					"session_id", process.sessionID,
					"action", "error_dropped",
				)
			}

			process.isHealthy = false
		}
	}

	if err := process.stderrScanner.Err(); err != nil {
		slog.Error("Claude stderr scanner error",
			"correlation_id", process.correlationID,
			"session_id", process.sessionID,
			"error", err,
			"lines_processed", stderrLineCount,
			"action", "stderr_scanner_error",
		)
	} else {
		slog.Debug("Claude stderr monitoring completed",
			"correlation_id", process.correlationID,
			"session_id", process.sessionID,
			"lines_processed", stderrLineCount,
			"action", "stderr_monitor_completed",
		)
	}
}

func (s *Service) CreateSession() (*Process, error) {
	return s.CreateSessionWithOptions("")
}

func (s *Service) CreateSessionWithOptions(workingDir string) (*Process, error) {
	startTime := time.Now()
	correlationID := uuid.New().String()

	slog.Info("Creating new Claude CLI session",
		"correlation_id", correlationID,
		"debug_mode", s.config.Debug,
		"action", "claude_process_start",
	)

	// Create debug directory if debug mode is enabled
	debugDir, err := s.createDebugDirectory(correlationID)
	if err != nil {
		slog.Error("Failed to create debug directory",
			"correlation_id", correlationID,
			"error", err,
			"action", "debug_dir_failed",
		)
		return nil, fmt.Errorf("failed to create debug directory: %w", err)
	}

	// Open debug files if debug mode is enabled
	stdinLogFile, stdoutLogFile, stderrLogFile, err := s.openDebugFiles(debugDir)
	if err != nil {
		slog.Error("Failed to open debug files",
			"correlation_id", correlationID,
			"error", err,
			"action", "debug_files_failed",
		)
		return nil, fmt.Errorf("failed to open debug files: %w", err)
	}

	if debugDir != "" {
		slog.Info("Debug mode enabled",
			"correlation_id", correlationID,
			"debug_dir", debugDir,
			"action", "debug_enabled",
		)
	}

	ctx, cancel := context.WithCancel(context.Background())

	args := []string{
		"--print",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--allowedTools", strings.Join(s.config.Tools, ","),
	}
	
	if workingDir != "" {
		args = append(args, "--add-dir", workingDir)
	}

	slog.Debug("Claude CLI command prepared",
		"correlation_id", correlationID,
		"command", "claude",
		"args", strings.Join(args, " "),
		"action", "claude_cmd_prepared",
	)

	cmd := exec.CommandContext(ctx, "claude", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		if stdinLogFile != nil {
			stdinLogFile.Close()
		}
		if stdoutLogFile != nil {
			stdoutLogFile.Close()
		}
		if stderrLogFile != nil {
			stderrLogFile.Close()
		}
		slog.Error("Failed to create Claude stdin pipe",
			"correlation_id", correlationID,
			"error", err,
			"action", "claude_stdin_failed",
		)
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		if stdinLogFile != nil {
			stdinLogFile.Close()
		}
		if stdoutLogFile != nil {
			stdoutLogFile.Close()
		}
		if stderrLogFile != nil {
			stderrLogFile.Close()
		}
		slog.Error("Failed to create Claude stdout pipe",
			"correlation_id", correlationID,
			"error", err,
			"action", "claude_stdout_failed",
		)
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		if stdinLogFile != nil {
			stdinLogFile.Close()
		}
		if stdoutLogFile != nil {
			stdoutLogFile.Close()
		}
		if stderrLogFile != nil {
			stderrLogFile.Close()
		}
		slog.Error("Failed to create Claude stderr pipe",
			"correlation_id", correlationID,
			"error", err,
			"action", "claude_stderr_failed",
		)
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	slog.Debug("Starting Claude CLI process",
		"correlation_id", correlationID,
		"action", "claude_process_starting",
	)

	if err := cmd.Start(); err != nil {
		cancel()
		if stdinLogFile != nil {
			stdinLogFile.Close()
		}
		if stdoutLogFile != nil {
			stdoutLogFile.Close()
		}
		if stderrLogFile != nil {
			stderrLogFile.Close()
		}
		slog.Error("Failed to start Claude CLI process",
			"correlation_id", correlationID,
			"error", err,
			"command", "claude",
			"args", strings.Join(args, " "),
			"action", "claude_process_start_failed",
		)
		return nil, fmt.Errorf("failed to start claude process: %w", err)
	}

	processStartDuration := time.Since(startTime)
	slog.Info("Claude CLI process started successfully",
		"correlation_id", correlationID,
		"pid", cmd.Process.Pid,
		"start_duration_ms", processStartDuration.Milliseconds(),
		"action", "claude_process_started",
	)

	process := &Process{
		cmd:           cmd,
		stdin:         stdin,
		stdout:        stdout,
		stderr:        stderr,
		stdoutScanner: bufio.NewScanner(stdout),
		stderrScanner: bufio.NewScanner(stderr),
		ctx:           ctx,
		cancel:        cancel,
		startTime:     startTime,
		correlationID: correlationID,
		debugDir:      debugDir,
		stdinLogFile:  stdinLogFile,
		stdoutLogFile: stdoutLogFile,
		stderrLogFile: stderrLogFile,
		isHealthy:     true,
		lastHeartbeat: time.Now(),
		inputChan:     make(chan Input, 10),   // Buffered channel for input
		outputChan:    make(chan Message, 10), // Buffered channel for output
		initComplete:  make(chan bool, 1),     // Signal channel for init
		errorChan:     make(chan Message, 10), // Buffered channel for errors
	}

	// Start stderr monitoring in background
	go s.monitorStderr(process)

	// Start message handlers
	go s.handleStdout(process)
	go s.handleStdin(process)

	initialMessage := Input{
		Type: "user",
		Message: InputMessage{
			Role: "user",
			Content: []InputMessageContent{
				{
					Type: "text",
					Text: "Hello, Claude! Initializing session.",
				},
			},
		},
	}
	select {
	case process.inputChan <- initialMessage:
		slog.Debug("Sent initial message to trigger Claude init",
			"correlation_id", correlationID,
			"action", "init_trigger_sent",
		)
	case <-time.After(5 * time.Second):
		cancel()
		process.closeDebugFiles()
		return nil, fmt.Errorf("timeout sending initial message")
	}

	// Wait for initialization to complete
	select {
	case <-process.initComplete:
		slog.Info("Claude session initialized successfully",
			"correlation_id", correlationID,
			"session_id", process.sessionID,
			"pid", cmd.Process.Pid,
			"total_duration_ms", time.Since(startTime).Milliseconds(),
			"action", "session_initialized",
		)
	case <-time.After(10 * time.Second):
		cancel()
		process.closeDebugFiles()
		return nil, fmt.Errorf("timeout waiting for Claude initialization")
	case <-ctx.Done():
		process.closeDebugFiles()
		return nil, fmt.Errorf("context cancelled during initialization")
	}

	// Add to active sessions
	s.mu.Lock()
	s.sessions[process.sessionID] = process
	s.mu.Unlock()

	return process, nil
}

// handleStdout reads messages from Claude's stdout and processes them
func (s *Service) handleStdout(process *Process) {
	defer close(process.outputChan)
	defer close(process.initComplete)

	slog.Debug("Starting stdout handler",
		"correlation_id", process.correlationID,
		"action", "stdout_handler_start",
	)

	messageCount := 0
	for process.stdoutScanner.Scan() {
		line := process.stdoutScanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		messageCount++

		// Log to debug file if enabled
		process.logToDebugFile(process.stdoutLogFile, "STDOUT", []byte(line))

		slog.Debug("Claude stdout line received",
			"correlation_id", process.correlationID,
			"line_length", len(line),
			"message_count", messageCount,
			"action", "stdout_line_received",
		)

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			slog.Error("Failed to parse Claude message",
				"correlation_id", process.correlationID,
				"error", err,
				"raw_line", line,
				"action", "message_parse_failed",
			)
			continue
		}

		// Handle initialization message
		if msg.Type == "system" && msg.Subtype == "init" && process.sessionID == "" {
			process.sessionID = msg.SessionID

			slog.Info("Received Claude init message",
				"correlation_id", process.correlationID,
				"session_id", msg.SessionID,
				"action", "init_message_received",
			)

			// Signal initialization complete
			select {
			case process.initComplete <- true:
			default:
			}
			continue
		}

		// Send to output channel
		select {
		case process.outputChan <- msg:
		case <-process.ctx.Done():
			slog.Debug("Context cancelled, stopping stdout handler",
				"correlation_id", process.correlationID,
				"action", "stdout_handler_cancelled",
			)
			return
		}
	}

	if err := process.stdoutScanner.Err(); err != nil {
		slog.Error("Stdout scanner error",
			"correlation_id", process.correlationID,
			"error", err,
			"messages_processed", messageCount,
			"action", "stdout_scanner_error",
		)
	}

	slog.Debug("Stdout handler completed",
		"correlation_id", process.correlationID,
		"messages_processed", messageCount,
		"action", "stdout_handler_completed",
	)
}

// handleStdin writes messages from the input channel to Claude's stdin
func (s *Service) handleStdin(process *Process) {
	slog.Debug("Starting stdin handler",
		"correlation_id", process.correlationID,
		"action", "stdin_handler_start",
	)

	messageCount := 0
	for {
		select {
		case message, ok := <-process.inputChan:
			if !ok {
				slog.Debug("Input channel closed, stopping stdin handler",
					"correlation_id", process.correlationID,
					"messages_sent", messageCount,
					"action", "stdin_handler_stopped",
				)
				return
			}

			messageCount++

			var (
				m   []byte
				err error
			)
			if m, err = json.Marshal(message); err != nil {
				slog.Error("Failed to marshal Claude input message",
					"correlation_id", process.correlationID,
					"error", err,
					"message_count", messageCount,
					"action", "stdin_message_marshal_failed",
				)
				continue
			}

			// Log to debug file if enabled
			process.logToDebugFile(process.stdinLogFile, "STDIN", m)

			// Write to Claude's stdin
			if _, err := fmt.Fprintln(process.stdin, string(m)); err != nil {
				slog.Error("Failed to write to Claude stdin",
					"correlation_id", process.correlationID,
					"error", err,
					"action", "stdin_write_failed",
				)
				return
			}

			slog.Debug("Sent message to Claude",
				"correlation_id", process.correlationID,
				"message_count", messageCount,
				"message", string(m),
				"action", "stdin_message_sent",
			)

		case <-process.ctx.Done():
			slog.Debug("Context cancelled, stopping stdin handler",
				"correlation_id", process.correlationID,
				"messages_sent", messageCount,
				"action", "stdin_handler_cancelled",
			)
			return
		}
	}
}

func (s *Service) SendMessage(process *Process, text string) error {
	message := Input{
		Type: "user",
		Message: InputMessage{
			Role: "user",
			Content: []InputMessageContent{
				{
					Type: "text",
					Text: text,
				},
			},
		},
	}

	select {
	case process.inputChan <- message:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending message")
	case <-process.ctx.Done():
		return fmt.Errorf("session cancelled")
	}
}

func (s *Service) ReceiveMessages(process *Process) <-chan Message {
	return process.outputChan
}

func (s *Service) StopSession(sessionID string) {
	startTime := time.Now()

	slog.Info("Stopping Claude session",
		"session_id", sessionID,
		"action", "session_stop_start",
	)

	s.mu.Lock()
	process, exists := s.sessions[sessionID]
	if exists {
		delete(s.sessions, sessionID)
	}
	s.mu.Unlock()

	if exists {
		correlationID := process.correlationID
		sessionUptime := time.Since(process.startTime)

		slog.Info("Found active session to stop",
			"correlation_id", correlationID,
			"session_id", sessionID,
			"session_uptime_ms", sessionUptime.Milliseconds(),
			"action", "session_found_for_stop",
		)

		// Clean up process
		slog.Debug("Cleaning up Claude process",
			"correlation_id", correlationID,
			"session_id", sessionID,
			"pid", func() int {
				if process.cmd != nil && process.cmd.Process != nil {
					return process.cmd.Process.Pid
				}
				return 0
			}(),
			"action", "process_cleanup_start",
		)

		// Close debug files
		process.closeDebugFiles()

		process.cancel()

		// Close channels to signal goroutines to stop
		if process.inputChan != nil {
			close(process.inputChan)
		}
		if process.errorChan != nil {
			close(process.errorChan)
		}
		// Note: outputChan and initComplete are closed by the handleStdout goroutine

		if process.stdin != nil {
			process.stdin.Close()
		}
		if process.stdout != nil {
			process.stdout.Close()
		}
		if process.stderr != nil {
			process.stderr.Close()
		}

		if process.cmd != nil {
			if err := process.cmd.Wait(); err != nil {
				slog.Warn("Claude process exited with error",
					"correlation_id", correlationID,
					"session_id", sessionID,
					"error", err,
					"action", "process_wait_error",
				)
			} else {
				slog.Debug("Claude process exited cleanly",
					"correlation_id", correlationID,
					"session_id", sessionID,
					"action", "process_exited_clean",
				)
			}
		}

		totalStopDuration := time.Since(startTime)
		slog.Info("Claude session stopped successfully",
			"correlation_id", correlationID,
			"session_id", sessionID,
			"session_uptime_ms", sessionUptime.Milliseconds(),
			"stop_duration_ms", totalStopDuration.Milliseconds(),
			"action", "session_stopped",
		)
	} else {
		slog.Warn("Attempted to stop non-existent session",
			"session_id", sessionID,
			"action", "session_not_found_for_stop",
		)
	}
}