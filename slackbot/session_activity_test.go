package slackbot

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
)

// MockSessionDB implements SessionDB interface for testing
type MockSessionDB struct {
	sessions       map[string]*SlackClaudeSession
	updateCalls    []string
	errors         map[string]error // threadTS -> error to return
	updateCount    int
	shouldFail     bool
	failAfterCalls int
	mu             sync.RWMutex
}

func NewMockSessionDB() *MockSessionDB {
	return &MockSessionDB{
		sessions: make(map[string]*SlackClaudeSession),
		errors:   make(map[string]error),
	}
}

func (m *MockSessionDB) UpdateSessionActivity(threadTS string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.updateCalls = append(m.updateCalls, threadTS)
	m.updateCount++
	
	// Check for specific error for this threadTS
	if err, exists := m.errors[threadTS]; exists {
		return err
	}
	
	// Check for global failure conditions
	if m.shouldFail && m.updateCount > m.failAfterCalls {
		return errors.New("database connection failed")
	}
	
	// Check if session exists and is active
	if session, exists := m.sessions[threadTS]; exists && session.Active {
		session.LastActivity = time.Now()
		return nil
	}
	
	return fmt.Errorf("no active session found for thread %s", threadTS)
}

func (m *MockSessionDB) GetSession(threadTS string) (*SlackClaudeSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if err, exists := m.errors[threadTS+"_get"]; exists {
		return nil, err
	}
	
	if session, exists := m.sessions[threadTS]; exists {
		return session, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockSessionDB) SetSession(session *SlackClaudeSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if err, exists := m.errors[session.ThreadTS+"_set"]; exists {
		return err
	}
	
	m.sessions[session.ThreadTS] = session
	return nil
}

func (m *MockSessionDB) SessionExists(threadTS string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if err, exists := m.errors[threadTS+"_exists"]; exists {
		return false, err
	}
	
	_, exists := m.sessions[threadTS]
	return exists, nil
}

func (m *MockSessionDB) SetError(threadTS string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[threadTS] = err
}

func (m *MockSessionDB) GetUpdateCalls() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]string, len(m.updateCalls))
	copy(calls, m.updateCalls)
	return calls
}

// MockSessionCache implements SessionCache interface for testing
type MockSessionCache struct {
	sessions map[string]*SlackClaudeSession
	mu       sync.RWMutex
}

func NewMockSessionCache() *MockSessionCache {
	return &MockSessionCache{
		sessions: make(map[string]*SlackClaudeSession),
	}
}

func (m *MockSessionCache) GetSession(threadTS string) (*SlackClaudeSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, exists := m.sessions[threadTS]
	return session, exists
}

func (m *MockSessionCache) SetSession(threadTS string, session *SlackClaudeSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[threadTS] = session
}

func (m *MockSessionCache) UpdateSessionActivity(threadTS string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session, exists := m.sessions[threadTS]; exists {
		session.LastActivity = time.Now()
	}
}

// MockTimeProvider implements TimeProvider interface for testing
type MockTimeProvider struct {
	currentTime time.Time
}

func NewMockTimeProvider(t time.Time) *MockTimeProvider {
	return &MockTimeProvider{currentTime: t}
}

func (m *MockTimeProvider) Now() time.Time {
	return m.currentTime
}

func (m *MockTimeProvider) SetTime(t time.Time) {
	m.currentTime = t
}

func TestSessionActivityManager_UpdateActivity_Success(t *testing.T) {
	mockDB := NewMockSessionDB()
	mockCache := NewMockSessionCache()
	
	// Create a session in both cache and database
	session := &SlackClaudeSession{
		ThreadTS:     "1234567890.123456",
		ChannelID:    "C1234567890",
		UserID:       "U1234567890",
		SessionID:    "session-123",
		Active:       true,
		LastActivity: time.Now().Add(-time.Hour),
	}
	
	mockCache.SetSession(session.ThreadTS, session)
	mockDB.sessions[session.ThreadTS] = session
	
	manager := NewSessionActivityManager(mockDB, mockCache, false)
	
	err := manager.UpdateActivity(session.ThreadTS)
	if err != nil {
		t.Errorf("UpdateActivity() failed: %v", err)
	}
	
	// Verify database was called
	calls := mockDB.GetUpdateCalls()
	if len(calls) != 1 || calls[0] != session.ThreadTS {
		t.Errorf("Expected 1 database call for %s, got %v", session.ThreadTS, calls)
	}
}

func TestSessionActivityManager_UpdateActivity_EmptyThreadTS(t *testing.T) {
	mockDB := NewMockSessionDB()
	mockCache := NewMockSessionCache()
	manager := NewSessionActivityManager(mockDB, mockCache, false)
	
	err := manager.UpdateActivity("")
	if err == nil {
		t.Error("UpdateActivity() should fail with empty threadTS")
	}
	
	if !strings.Contains(err.Error(), "threadTS cannot be empty") {
		t.Errorf("Expected error about empty threadTS, got: %v", err)
	}
}

func TestSessionActivityManager_UpdateActivity_RaceCondition(t *testing.T) {
	mockDB := NewMockSessionDB()
	mockCache := NewMockSessionCache()
	
	threadTS := "1234567890.123456"
	
	// Session exists in cache but not in database (race condition)
	session := &SlackClaudeSession{
		ThreadTS:     threadTS,
		ChannelID:    "C1234567890",
		UserID:       "U1234567890",
		SessionID:    "session-123",
		Active:       true,
		LastActivity: time.Now(),
	}
	mockCache.SetSession(threadTS, session)
	
	// Database will initially fail with "no active session found"
	mockDB.SetError(threadTS, fmt.Errorf("no active session found for thread %s", threadTS))
	
	manager := NewSessionActivityManager(mockDB, mockCache, true)
	
	// First attempt should fail, but it should try to create the missing session
	err := manager.UpdateActivity(threadTS)
	
	// The manager should have attempted to create the session
	// After creating, it should exist in the mock database
	if _, exists := mockDB.sessions[threadTS]; !exists {
		t.Error("Manager should have created missing session in database")
	}
	
	// Error might still occur on first attempt, but that's okay for race conditions
	if err != nil {
		// Make sure it's the expected race condition error
		if !manager.isRaceConditionError(err) {
			t.Errorf("Expected race condition error, got: %v", err)
		}
	}
}

func TestSessionActivityManager_UpdateActivity_RetryLogic(t *testing.T) {
	mockDB := NewMockSessionDB()
	mockCache := NewMockSessionCache()
	
	threadTS := "1234567890.123456"
	session := &SlackClaudeSession{
		ThreadTS:     threadTS,
		ChannelID:    "C1234567890",
		UserID:       "U1234567890",
		SessionID:    "session-123",
		Active:       true,
		LastActivity: time.Now(),
	}
	
	mockCache.SetSession(threadTS, session)
	mockDB.sessions[threadTS] = session
	
	// Make database fail with a transient error initially
	mockDB.shouldFail = true
	mockDB.failAfterCalls = 0
	
	manager := NewSessionActivityManager(mockDB, mockCache, true)
	manager.maxRetries = 2
	manager.retryDelay = 1 * time.Millisecond // Speed up test
	
	// Simulate transient failure that succeeds after retries
	go func() {
		time.Sleep(5 * time.Millisecond)
		mockDB.mu.Lock()
		mockDB.shouldFail = false
		mockDB.mu.Unlock()
	}()
	
	err := manager.UpdateActivity(threadTS)
	if err != nil {
		t.Errorf("UpdateActivity() should succeed after retries, got: %v", err)
	}
	
	// Verify multiple attempts were made
	calls := mockDB.GetUpdateCalls()
	if len(calls) < 2 {
		t.Errorf("Expected multiple retry attempts, got %d calls", len(calls))
	}
}

func TestSessionActivityManager_UpdateActivity_NonTransientError(t *testing.T) {
	mockDB := NewMockSessionDB()
	mockCache := NewMockSessionCache()
	
	threadTS := "1234567890.123456"
	
	// Set up a non-transient error
	mockDB.SetError(threadTS, errors.New("constraint violation"))
	
	manager := NewSessionActivityManager(mockDB, mockCache, false)
	
	err := manager.UpdateActivity(threadTS)
	if err == nil {
		t.Error("UpdateActivity() should fail with non-transient error")
	}
	
	// Should only try once for non-transient errors
	calls := mockDB.GetUpdateCalls()
	if len(calls) != 1 {
		t.Errorf("Expected exactly 1 call for non-transient error, got %d", len(calls))
	}
}

func TestSessionActivityManager_UpdateActivity_ConcurrentUpdates(t *testing.T) {
	mockDB := NewMockSessionDB()
	mockCache := NewMockSessionCache()
	
	threadTS := "1234567890.123456"
	session := &SlackClaudeSession{
		ThreadTS:     threadTS,
		ChannelID:    "C1234567890",
		UserID:       "U1234567890",
		SessionID:    "session-123",
		Active:       true,
		LastActivity: time.Now(),
	}
	
	mockCache.SetSession(threadTS, session)
	mockDB.sessions[threadTS] = session
	
	manager := NewSessionActivityManager(mockDB, mockCache, false)
	
	// Run multiple concurrent updates
	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := manager.UpdateActivity(threadTS)
			if err != nil {
				errors <- err
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	
	// Check for any errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}
	
	if len(errorList) > 0 {
		t.Errorf("Concurrent updates failed with errors: %v", errorList)
	}
	
	// All calls should have been made
	calls := mockDB.GetUpdateCalls()
	if len(calls) != numGoroutines {
		t.Errorf("Expected %d calls, got %d", numGoroutines, len(calls))
	}
}

func TestSessionActivityManager_IsRaceConditionError(t *testing.T) {
	manager := NewSessionActivityManager(nil, nil, false)
	
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "race condition error",
			err:      fmt.Errorf("no active session found for thread 123"),
			expected: true,
		},
		{
			name:     "gorm record not found",
			err:      gorm.ErrRecordNotFound,
			expected: true,
		},
		{
			name:     "other database error",
			err:      errors.New("connection failed"),
			expected: false,
		},
		{
			name:     "constraint violation",
			err:      errors.New("UNIQUE constraint failed"),
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.isRaceConditionError(tt.err)
			if result != tt.expected {
				t.Errorf("isRaceConditionError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestSessionActivityManager_IsTransientError(t *testing.T) {
	manager := NewSessionActivityManager(nil, nil, false)
	
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "connection error",
			err:      errors.New("database connection failed"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      errors.New("operation timeout"),
			expected: true,
		},
		{
			name:     "deadlock error",
			err:      errors.New("deadlock detected"),
			expected: true,
		},
		{
			name:     "constraint violation",
			err:      errors.New("UNIQUE constraint failed"),
			expected: false,
		},
		{
			name:     "syntax error",
			err:      errors.New("syntax error in SQL"),
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.isTransientError(tt.err)
			if result != tt.expected {
				t.Errorf("isTransientError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestSessionActivityManager_GetSessionInfo(t *testing.T) {
	mockDB := NewMockSessionDB()
	mockCache := NewMockSessionCache()
	mockTime := NewMockTimeProvider(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	
	threadTS := "1234567890.123456"
	session := &SlackClaudeSession{
		ThreadTS:     threadTS,
		ChannelID:    "C1234567890",
		UserID:       "U1234567890",
		SessionID:    "session-123",
		Active:       true,
		LastActivity: mockTime.Now().Add(-time.Hour),
	}
	
	// Add session to both cache and database
	mockCache.SetSession(threadTS, session)
	mockDB.sessions[threadTS] = session
	
	manager := NewSessionActivityManager(mockDB, mockCache, false)
	manager.timeProvider = mockTime
	
	info := manager.GetSessionInfo(threadTS)
	
	// Verify expected fields
	if info["thread_ts"] != threadTS {
		t.Errorf("Expected thread_ts %s, got %v", threadTS, info["thread_ts"])
	}
	
	if info["cache_exists"] != true {
		t.Errorf("Expected cache_exists true, got %v", info["cache_exists"])
	}
	
	if info["db_exists"] != true {
		t.Errorf("Expected db_exists true, got %v", info["db_exists"])
	}
	
	if info["cache_session_id"] != "session-123" {
		t.Errorf("Expected cache_session_id session-123, got %v", info["cache_session_id"])
	}
	
	if info["db_session_id"] != "session-123" {
		t.Errorf("Expected db_session_id session-123, got %v", info["db_session_id"])
	}
}

func TestSessionActivityManager_TryCreateMissingSession(t *testing.T) {
	mockDB := NewMockSessionDB()
	mockCache := NewMockSessionCache()
	
	threadTS := "1234567890.123456"
	session := &SlackClaudeSession{
		ThreadTS:     threadTS,
		ChannelID:    "C1234567890",
		UserID:       "U1234567890",
		SessionID:    "session-123",
		Active:       false, // Initially inactive
		LastActivity: time.Now().Add(-time.Hour),
	}
	
	// Session exists in cache but not in database
	mockCache.SetSession(threadTS, session)
	
	manager := NewSessionActivityManager(mockDB, mockCache, true)
	
	success := manager.tryCreateMissingSession(threadTS)
	if !success {
		t.Error("tryCreateMissingSession() should succeed")
	}
	
	// Verify session was created in database and marked as active
	if dbSession, exists := mockDB.sessions[threadTS]; exists {
		if !dbSession.Active {
			t.Error("Created session should be marked as active")
		}
	} else {
		t.Error("Session should have been created in database")
	}
}