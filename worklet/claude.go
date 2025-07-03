package worklet

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/breadchris/flow/deps"
	"github.com/breadchris/flow/coderunner/claude"
)

type ClaudeClient struct {
	deps          *deps.Dependencies
	claudeService *claude.ClaudeService
}

func NewClaudeClient(deps *deps.Dependencies) *ClaudeClient {
	return &ClaudeClient{
		deps:          deps,
		claudeService: claude.NewClaudeService(deps),
	}
}

func (c *ClaudeClient) ApplyPrompt(ctx context.Context, repoPath, prompt, sessionID string) error {
	if prompt == "" {
		return nil
	}
	
	slog.Info("Applying prompt to worklet", "sessionID", sessionID, "repoPath", repoPath)
	
	session, err := c.claudeService.CreateSession(ctx, sessionID, "", repoPath)
	if err != nil {
		return fmt.Errorf("failed to create Claude session: %w", err)
	}
	
	_, err = c.claudeService.SendMessage(ctx, sessionID, prompt)
	if err != nil {
		return fmt.Errorf("failed to send message to Claude: %w", err)
	}
	
	return nil
}

func (c *ClaudeClient) ProcessPrompt(ctx context.Context, repoPath, prompt, sessionID string) (string, error) {
	slog.Info("Processing prompt for worklet", "sessionID", sessionID, "repoPath", repoPath)
	
	session, err := c.claudeService.GetSession(sessionID)
	if err != nil {
		session, err = c.claudeService.CreateSession(ctx, sessionID, "", repoPath)
		if err != nil {
			return "", fmt.Errorf("failed to create Claude session: %w", err)
		}
	}
	
	response, err := c.claudeService.SendMessage(ctx, sessionID, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to send message to Claude: %w", err)
	}
	
	if err := c.waitForCompletion(ctx, sessionID); err != nil {
		return "", fmt.Errorf("failed to wait for completion: %w", err)
	}
	
	return response, nil
}

func (c *ClaudeClient) waitForCompletion(ctx context.Context, sessionID string) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for Claude response")
		case <-ticker.C:
			session, err := c.claudeService.GetSession(sessionID)
			if err != nil {
				return fmt.Errorf("failed to get session: %w", err)
			}
			
			if session.IsHealthy() {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *ClaudeClient) CreatePR(ctx context.Context, repoPath, branchName, title, description string) error {
	slog.Info("Creating PR for worklet", "repoPath", repoPath, "branch", branchName)
	
	if !c.isGitRepo(repoPath) {
		return fmt.Errorf("not a git repository")
	}
	
	if err := c.createBranch(repoPath, branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}
	
	if err := c.commitChanges(repoPath, title); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	
	if err := c.pushBranch(repoPath, branchName); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}
	
	if err := c.createGitHubPR(repoPath, branchName, title, description); err != nil {
		return fmt.Errorf("failed to create GitHub PR: %w", err)
	}
	
	return nil
}

func (c *ClaudeClient) isGitRepo(repoPath string) bool {
	_, err := os.Stat(filepath.Join(repoPath, ".git"))
	return err == nil
}

func (c *ClaudeClient) createBranch(repoPath, branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = repoPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create branch: %s", string(output))
	}
	
	return nil
}

func (c *ClaudeClient) commitChanges(repoPath, message string) error {
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = repoPath
	
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add changes: %s", string(output))
	}
	
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoPath
	
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}
	
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		slog.Info("No changes to commit")
		return nil
	}
	
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = repoPath
	
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit changes: %s", string(output))
	}
	
	return nil
}

func (c *ClaudeClient) pushBranch(repoPath, branchName string) error {
	cmd := exec.Command("git", "push", "-u", "origin", branchName)
	cmd.Dir = repoPath
	
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", token))
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push branch: %s", string(output))
	}
	
	return nil
}

func (c *ClaudeClient) createGitHubPR(repoPath, branchName, title, description string) error {
	if !c.isGitHubCLIAvailable() {
		return fmt.Errorf("GitHub CLI (gh) is not available")
	}
	
	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", description, "--head", branchName)
	cmd.Dir = repoPath
	
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", token))
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create PR: %s", string(output))
	}
	
	slog.Info("Created PR successfully", "output", string(output))
	
	return nil
}

func (c *ClaudeClient) isGitHubCLIAvailable() bool {
	cmd := exec.Command("gh", "--version")
	return cmd.Run() == nil
}

func (c *ClaudeClient) GetSessionStatus(sessionID string) (string, error) {
	session, err := c.claudeService.GetSession(sessionID)
	if err != nil {
		return "not_found", err
	}
	
	if session.IsHealthy() {
		return "healthy", nil
	}
	
	return "unhealthy", nil
}

func (c *ClaudeClient) CloseSession(sessionID string) error {
	return c.claudeService.CloseSession(sessionID)
}

func (c *ClaudeClient) ListSessions() []string {
	return c.claudeService.ListSessions()
}

func (c *ClaudeClient) CleanupOldSessions(maxAge time.Duration) error {
	sessions := c.claudeService.ListSessions()
	
	for _, sessionID := range sessions {
		session, err := c.claudeService.GetSession(sessionID)
		if err != nil {
			continue
		}
		
		if time.Since(session.StartTime()) > maxAge {
			slog.Info("Cleaning up old Claude session", "sessionID", sessionID)
			if err := c.claudeService.CloseSession(sessionID); err != nil {
				slog.Error("Failed to close old session", "error", err, "sessionID", sessionID)
			}
		}
	}
	
	return nil
}