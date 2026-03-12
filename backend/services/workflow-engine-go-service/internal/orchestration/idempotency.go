package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// IdempotencyManager provides idempotency control for workflow operations
// Prevents duplicate commits and ensures transaction safety
type IdempotencyManager struct {
	redis  *redis.Client
	logger *zap.Logger
	config IdempotencyConfig
}

// IdempotencyConfig configures idempotency behavior
type IdempotencyConfig struct {
	TokenTTL           time.Duration // How long tokens remain valid
	ResultCacheTTL     time.Duration // How long results are cached
	MaxRetries         int           // Maximum retry attempts
	RetryBackoffBase   time.Duration // Base backoff duration for retries
	CircuitBreakerSize int           // Consecutive failures before circuit breaker opens
}

// IdempotencyToken represents a unique token for an operation
type IdempotencyToken struct {
	Token           string    `json:"token"`
	RequestHash     string    `json:"request_hash"`
	WorkflowID      string    `json:"workflow_id"`
	OperationType   string    `json:"operation_type"`
	GeneratedAt     time.Time `json:"generated_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	Attempts        int       `json:"attempts"`
	LastAttemptAt   time.Time `json:"last_attempt_at"`
	Status          string    `json:"status"` // PENDING, SUCCESS, FAILURE
}

// IdempotencyResult represents the cached result of an idempotent operation
type IdempotencyResult struct {
	Token        string      `json:"token"`
	Success      bool        `json:"success"`
	Result       interface{} `json:"result"`
	Error        string      `json:"error,omitempty"`
	ExecutedAt   time.Time   `json:"executed_at"`
	Duration     time.Duration `json:"duration"`
	Attempts     int         `json:"attempts"`
}

// RetryableError represents an error that can be retried
type RetryableError struct {
	Message   string
	Temporary bool
	Cause     error
}

func (e *RetryableError) Error() string {
	return e.Message
}

// NewIdempotencyManager creates a new idempotency manager
func NewIdempotencyManager(redis *redis.Client, logger *zap.Logger) *IdempotencyManager {
	return &IdempotencyManager{
		redis:  redis,
		logger: logger,
		config: IdempotencyConfig{
			TokenTTL:           1 * time.Hour,
			ResultCacheTTL:     24 * time.Hour,
			MaxRetries:         3,
			RetryBackoffBase:   100 * time.Millisecond,
			CircuitBreakerSize: 5,
		},
	}
}

// GenerateToken generates a unique idempotency token for a request
func (im *IdempotencyManager) GenerateToken(ctx context.Context, request interface{}, workflowID, operationType string) (*IdempotencyToken, error) {
	// Create hash of request for uniqueness
	requestHash, err := im.hashRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to hash request: %w", err)
	}

	// Generate token
	token := im.generateTokenString(workflowID, operationType, requestHash)

	idempotencyToken := &IdempotencyToken{
		Token:         token,
		RequestHash:   requestHash,
		WorkflowID:    workflowID,
		OperationType: operationType,
		GeneratedAt:   time.Now(),
		ExpiresAt:     time.Now().Add(im.config.TokenTTL),
		Attempts:      0,
		Status:        "PENDING",
	}

	// Store token in Redis
	tokenKey := im.getTokenKey(token)
	tokenJSON, err := json.Marshal(idempotencyToken)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token: %w", err)
	}

	err = im.redis.Set(ctx, tokenKey, tokenJSON, im.config.TokenTTL).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	im.logger.Info("Generated idempotency token",
		zap.String("token", token),
		zap.String("workflow_id", workflowID),
		zap.String("operation_type", operationType))

	return idempotencyToken, nil
}

// CheckAndExecute checks if an operation has already been executed and returns cached result,
// or executes the operation if it's the first time
func (im *IdempotencyManager) CheckAndExecute(ctx context.Context, token string, executor func() (interface{}, error)) (*IdempotencyResult, error) {
	// Check if result already exists
	resultKey := im.getResultKey(token)
	resultJSON, err := im.redis.Get(ctx, resultKey).Result()
	if err == nil {
		// Result exists, return cached result
		var cachedResult IdempotencyResult
		if err := json.Unmarshal([]byte(resultJSON), &cachedResult); err == nil {
			im.logger.Info("Returning cached idempotent result",
				zap.String("token", token),
				zap.Bool("success", cachedResult.Success))
			return &cachedResult, nil
		}
	}

	// Get token information
	idempotencyToken, err := im.getToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token: %w", err)
	}

	// Check if token has expired
	if time.Now().After(idempotencyToken.ExpiresAt) {
		return nil, fmt.Errorf("idempotency token has expired")
	}

	// Execute operation with retry logic
	result, err := im.executeWithRetry(ctx, idempotencyToken, executor)
	if err != nil {
		return nil, err
	}

	// Cache result
	if err := im.cacheResult(ctx, result); err != nil {
		im.logger.Error("Failed to cache idempotent result", zap.Error(err))
		// Continue anyway - the operation succeeded
	}

	return result, nil
}

// executeWithRetry executes an operation with exponential backoff retry logic
func (im *IdempotencyManager) executeWithRetry(ctx context.Context, token *IdempotencyToken, executor func() (interface{}, error)) (*IdempotencyResult, error) {
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt < im.config.MaxRetries; attempt++ {
		// Update attempt counter
		token.Attempts = attempt + 1
		token.LastAttemptAt = time.Now()
		if err := im.updateToken(ctx, token); err != nil {
			im.logger.Error("Failed to update token attempt counter", zap.Error(err))
		}

		// Execute operation
		result, err := executor()
		if err == nil {
			// Success
			duration := time.Since(startTime)
			idempotentResult := &IdempotencyResult{
				Token:      token.Token,
				Success:    true,
				Result:     result,
				ExecutedAt: time.Now(),
				Duration:   duration,
				Attempts:   attempt + 1,
			}

			// Update token status
			token.Status = "SUCCESS"
			if updateErr := im.updateToken(ctx, token); updateErr != nil {
				im.logger.Error("Failed to update token status", zap.Error(updateErr))
			}

			im.logger.Info("Idempotent operation succeeded",
				zap.String("token", token.Token),
				zap.Duration("duration", duration),
				zap.Int("attempts", attempt+1))

			return idempotentResult, nil
		}

		lastErr = err

		// Check if error is retryable
		if retryableErr, ok := err.(*RetryableError); ok && !retryableErr.Temporary {
			// Non-temporary error, don't retry
			break
		}

		// Check for circuit breaker
		if attempt >= im.config.CircuitBreakerSize {
			im.logger.Error("Circuit breaker triggered for idempotent operation",
				zap.String("token", token.Token),
				zap.Int("attempts", attempt+1))
			break
		}

		// Calculate backoff delay
		backoffDelay := im.calculateBackoff(attempt)

		im.logger.Warn("Idempotent operation failed, retrying",
			zap.String("token", token.Token),
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", im.config.MaxRetries),
			zap.Duration("backoff", backoffDelay))

		// Wait before retry
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoffDelay):
			continue
		}
	}

	// All retries failed
	duration := time.Since(startTime)
	idempotentResult := &IdempotencyResult{
		Token:      token.Token,
		Success:    false,
		Error:      lastErr.Error(),
		ExecutedAt: time.Now(),
		Duration:   duration,
		Attempts:   im.config.MaxRetries,
	}

	// Update token status
	token.Status = "FAILURE"
	if err := im.updateToken(ctx, token); err != nil {
		im.logger.Error("Failed to update token status", zap.Error(err))
	}

	im.logger.Error("Idempotent operation failed after all retries",
		zap.String("token", token.Token),
		zap.Error(lastErr),
		zap.Duration("total_duration", duration),
		zap.Int("total_attempts", im.config.MaxRetries))

	return idempotentResult, fmt.Errorf("operation failed after %d attempts: %w", im.config.MaxRetries, lastErr)
}

// ValidateToken validates that a token exists and is not expired
func (im *IdempotencyManager) ValidateToken(ctx context.Context, token string) error {
	_, err := im.getToken(ctx, token)
	return err
}

// RevokeToken revokes an idempotency token
func (im *IdempotencyManager) RevokeToken(ctx context.Context, token string) error {
	tokenKey := im.getTokenKey(token)
	resultKey := im.getResultKey(token)

	// Delete both token and cached result
	pipe := im.redis.Pipeline()
	pipe.Del(ctx, tokenKey)
	pipe.Del(ctx, resultKey)
	_, err := pipe.Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	im.logger.Info("Revoked idempotency token", zap.String("token", token))
	return nil
}

// GetOperationStatus returns the current status of an operation
func (im *IdempotencyManager) GetOperationStatus(ctx context.Context, token string) (*IdempotencyToken, *IdempotencyResult, error) {
	// Get token
	idempotencyToken, err := im.getToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	// Check for cached result
	resultKey := im.getResultKey(token)
	resultJSON, err := im.redis.Get(ctx, resultKey).Result()
	if err != nil {
		if err == redis.Nil {
			// No result yet, operation may be in progress
			return idempotencyToken, nil, nil
		}
		return idempotencyToken, nil, fmt.Errorf("failed to get result: %w", err)
	}

	var result IdempotencyResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return idempotencyToken, nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return idempotencyToken, &result, nil
}

// Helper methods

func (im *IdempotencyManager) hashRequest(request interface{}) (string, error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(requestJSON)
	return hex.EncodeToString(hash[:]), nil
}

func (im *IdempotencyManager) generateTokenString(workflowID, operationType, requestHash string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", workflowID, operationType, requestHash, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:32] // Use first 32 characters
}

func (im *IdempotencyManager) getTokenKey(token string) string {
	return fmt.Sprintf("idempotency:token:%s", token)
}

func (im *IdempotencyManager) getResultKey(token string) string {
	return fmt.Sprintf("idempotency:result:%s", token)
}

func (im *IdempotencyManager) getToken(ctx context.Context, token string) (*IdempotencyToken, error) {
	tokenKey := im.getTokenKey(token)
	tokenJSON, err := im.redis.Get(ctx, tokenKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var idempotencyToken IdempotencyToken
	if err := json.Unmarshal([]byte(tokenJSON), &idempotencyToken); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &idempotencyToken, nil
}

func (im *IdempotencyManager) updateToken(ctx context.Context, token *IdempotencyToken) error {
	tokenKey := im.getTokenKey(token.Token)
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	ttl := token.ExpiresAt.Sub(time.Now())
	if ttl <= 0 {
		ttl = 1 * time.Minute // Grace period for expired tokens
	}

	return im.redis.Set(ctx, tokenKey, tokenJSON, ttl).Err()
}

func (im *IdempotencyManager) cacheResult(ctx context.Context, result *IdempotencyResult) error {
	resultKey := im.getResultKey(result.Token)
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return im.redis.Set(ctx, resultKey, resultJSON, im.config.ResultCacheTTL).Err()
}

func (im *IdempotencyManager) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: base * 2^attempt
	backoff := im.config.RetryBackoffBase * (1 << uint(attempt))

	// Cap at 30 seconds
	maxBackoff := 30 * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

// CreateRetryableError creates a retryable error
func CreateRetryableError(message string, temporary bool, cause error) *RetryableError {
	return &RetryableError{
		Message:   message,
		Temporary: temporary,
		Cause:     cause,
	}
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if retryableErr, ok := err.(*RetryableError); ok {
		return retryableErr.Temporary
	}
	return false
}

// CommitExecutor is a wrapper for commit operations that provides idempotency
type CommitExecutor struct {
	idempotencyManager *IdempotencyManager
	logger             *zap.Logger
}

// NewCommitExecutor creates a new commit executor
func NewCommitExecutor(idempotencyManager *IdempotencyManager, logger *zap.Logger) *CommitExecutor {
	return &CommitExecutor{
		idempotencyManager: idempotencyManager,
		logger:             logger,
	}
}

// ExecuteCommit executes a commit operation with idempotency protection
func (ce *CommitExecutor) ExecuteCommit(ctx context.Context, request interface{}, workflowID string, commitFunc func() (interface{}, error)) (interface{}, error) {
	// Generate idempotency token
	token, err := ce.idempotencyManager.GenerateToken(ctx, request, workflowID, "COMMIT")
	if err != nil {
		return nil, fmt.Errorf("failed to generate idempotency token: %w", err)
	}

	// Execute with idempotency protection
	result, err := ce.idempotencyManager.CheckAndExecute(ctx, token.Token, commitFunc)
	if err != nil {
		return nil, fmt.Errorf("commit execution failed: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("commit failed: %s", result.Error)
	}

	ce.logger.Info("Commit executed successfully with idempotency protection",
		zap.String("workflow_id", workflowID),
		zap.String("token", token.Token),
		zap.Duration("duration", result.Duration),
		zap.Int("attempts", result.Attempts))

	return result.Result, nil
}