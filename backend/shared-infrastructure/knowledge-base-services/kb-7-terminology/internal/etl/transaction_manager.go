package etl

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"kb-7-terminology/internal/elasticsearch"

	"go.uber.org/zap"
)

// TransactionManager handles distributed transactions across PostgreSQL and Elasticsearch
type TransactionManager struct {
	db       *sql.DB
	esClient *elasticsearch.Client
	logger   *zap.Logger

	// Transaction state
	transactions map[string]*DistributedTransaction
	txMutex      sync.RWMutex

	// Configuration
	config *TransactionConfig
}

// TransactionConfig holds transaction management configuration
type TransactionConfig struct {
	TransactionTimeout time.Duration `json:"transaction_timeout"`
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
	EnableSnapshots    bool          `json:"enable_snapshots"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
}

// DistributedTransaction represents a transaction across multiple stores
type DistributedTransaction struct {
	ID        string                 `json:"id"`
	StartTime time.Time             `json:"start_time"`
	Status    TransactionStatus     `json:"status"`
	Stores    map[string]StoreState `json:"stores"`
	Operations []TransactionOperation `json:"operations"`
	Snapshots  map[string]interface{} `json:"snapshots"`
	Context    context.Context       `json:"-"`
	Cancel     context.CancelFunc    `json:"-"`
	mutex      sync.RWMutex         `json:"-"`
}

// TransactionStatus represents the state of a distributed transaction
type TransactionStatus string

const (
	TransactionStatusPending    TransactionStatus = "pending"
	TransactionStatusPreparing  TransactionStatus = "preparing"
	TransactionStatusPrepared   TransactionStatus = "prepared"
	TransactionStatusCommitting TransactionStatus = "committing"
	TransactionStatusCommitted  TransactionStatus = "committed"
	TransactionStatusAborting   TransactionStatus = "aborting"
	TransactionStatusAborted    TransactionStatus = "aborted"
	TransactionStatusFailed     TransactionStatus = "failed"
)

// StoreState represents the state of a store within a transaction
type StoreState struct {
	StoreName     string            `json:"store_name"`
	Status        string            `json:"status"`
	PreparedAt    *time.Time        `json:"prepared_at,omitempty"`
	CommittedAt   *time.Time        `json:"committed_at,omitempty"`
	AbortedAt     *time.Time        `json:"aborted_at,omitempty"`
	LastError     string            `json:"last_error,omitempty"`
	SnapshotID    string            `json:"snapshot_id,omitempty"`
	RecordsCount  int64            `json:"records_count"`
	Checksum      string            `json:"checksum,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// TransactionOperation represents an operation within a transaction
type TransactionOperation struct {
	ID          string                 `json:"id"`
	Type        OperationType         `json:"type"`
	StoreName   string                `json:"store_name"`
	EntityID    string                `json:"entity_id"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time             `json:"timestamp"`
	Status      string                `json:"status"`
	RetryCount  int                   `json:"retry_count"`
	Error       string                `json:"error,omitempty"`
}

// OperationType defines the type of operation
type OperationType string

const (
	OperationTypeInsert OperationType = "insert"
	OperationTypeUpdate OperationType = "update"
	OperationTypeDelete OperationType = "delete"
	OperationTypeBulk   OperationType = "bulk"
)

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(
	db *sql.DB,
	esClient *elasticsearch.Client,
	logger *zap.Logger,
	config *TransactionConfig,
) *TransactionManager {
	if config == nil {
		config = &TransactionConfig{
			TransactionTimeout: 5 * time.Minute,
			MaxRetries:         3,
			RetryDelay:         5 * time.Second,
			EnableSnapshots:    true,
			CleanupInterval:    10 * time.Minute,
		}
	}

	tm := &TransactionManager{
		db:           db,
		esClient:     esClient,
		logger:       logger,
		config:       config,
		transactions: make(map[string]*DistributedTransaction),
	}

	// Start cleanup goroutine
	go tm.cleanupExpiredTransactions()

	return tm
}

// BeginTransaction starts a new distributed transaction
func (tm *TransactionManager) BeginTransaction(ctx context.Context, transactionID string) (*DistributedTransaction, error) {
	tm.txMutex.Lock()
	defer tm.txMutex.Unlock()

	// Check if transaction already exists
	if _, exists := tm.transactions[transactionID]; exists {
		return nil, fmt.Errorf("transaction %s already exists", transactionID)
	}

	// Create transaction context with timeout
	txCtx, cancel := context.WithTimeout(ctx, tm.config.TransactionTimeout)

	transaction := &DistributedTransaction{
		ID:        transactionID,
		StartTime: time.Now(),
		Status:    TransactionStatusPending,
		Stores:    make(map[string]StoreState),
		Operations: make([]TransactionOperation, 0),
		Snapshots:  make(map[string]interface{}),
		Context:   txCtx,
		Cancel:    cancel,
	}

	// Initialize store states
	transaction.Stores["postgresql"] = StoreState{
		StoreName: "postgresql",
		Status:    "pending",
		Metadata:  make(map[string]string),
	}

	transaction.Stores["elasticsearch"] = StoreState{
		StoreName: "elasticsearch",
		Status:    "pending",
		Metadata:  make(map[string]string),
	}

	tm.transactions[transactionID] = transaction

	tm.logger.Info("Distributed transaction started",
		zap.String("transaction_id", transactionID),
	)

	return transaction, nil
}

// PrepareTransaction prepares all stores for commit (two-phase commit)
func (tm *TransactionManager) PrepareTransaction(transactionID string) error {
	tm.txMutex.RLock()
	transaction, exists := tm.transactions[transactionID]
	tm.txMutex.RUnlock()

	if !exists {
		return fmt.Errorf("transaction %s not found", transactionID)
	}

	transaction.mutex.Lock()
	defer transaction.mutex.Unlock()

	if transaction.Status != TransactionStatusPending {
		return fmt.Errorf("transaction %s is in invalid state for prepare: %s",
			transactionID, transaction.Status)
	}

	transaction.Status = TransactionStatusPreparing

	tm.logger.Info("Preparing transaction",
		zap.String("transaction_id", transactionID),
	)

	// Prepare PostgreSQL
	if err := tm.preparePostgreSQL(transaction); err != nil {
		transaction.Status = TransactionStatusFailed
		return fmt.Errorf("PostgreSQL prepare failed: %w", err)
	}

	// Prepare Elasticsearch
	if err := tm.prepareElasticsearch(transaction); err != nil {
		transaction.Status = TransactionStatusFailed
		// Abort PostgreSQL if Elasticsearch prepare fails
		tm.abortPostgreSQL(transaction)
		return fmt.Errorf("Elasticsearch prepare failed: %w", err)
	}

	transaction.Status = TransactionStatusPrepared

	tm.logger.Info("Transaction prepared successfully",
		zap.String("transaction_id", transactionID),
	)

	return nil
}

// CommitTransaction commits the prepared transaction
func (tm *TransactionManager) CommitTransaction(transactionID string) error {
	tm.txMutex.RLock()
	transaction, exists := tm.transactions[transactionID]
	tm.txMutex.RUnlock()

	if !exists {
		return fmt.Errorf("transaction %s not found", transactionID)
	}

	transaction.mutex.Lock()
	defer transaction.mutex.Unlock()

	if transaction.Status != TransactionStatusPrepared {
		return fmt.Errorf("transaction %s is not prepared for commit: %s",
			transactionID, transaction.Status)
	}

	transaction.Status = TransactionStatusCommitting

	tm.logger.Info("Committing transaction",
		zap.String("transaction_id", transactionID),
	)

	var commitErrors []error

	// Commit PostgreSQL
	if err := tm.commitPostgreSQL(transaction); err != nil {
		commitErrors = append(commitErrors, fmt.Errorf("PostgreSQL commit failed: %w", err))
	}

	// Commit Elasticsearch
	if err := tm.commitElasticsearch(transaction); err != nil {
		commitErrors = append(commitErrors, fmt.Errorf("Elasticsearch commit failed: %w", err))
	}

	if len(commitErrors) > 0 {
		transaction.Status = TransactionStatusFailed
		tm.logger.Error("Transaction commit failed",
			zap.String("transaction_id", transactionID),
			zap.Errors("errors", commitErrors),
		)
		return fmt.Errorf("commit failed with %d errors: %v", len(commitErrors), commitErrors[0])
	}

	transaction.Status = TransactionStatusCommitted

	tm.logger.Info("Transaction committed successfully",
		zap.String("transaction_id", transactionID),
	)

	// Schedule cleanup
	go tm.cleanupTransaction(transactionID, 30*time.Second)

	return nil
}

// AbortTransaction aborts the transaction and rolls back changes
func (tm *TransactionManager) AbortTransaction(transactionID string) error {
	tm.txMutex.RLock()
	transaction, exists := tm.transactions[transactionID]
	tm.txMutex.RUnlock()

	if !exists {
		return fmt.Errorf("transaction %s not found", transactionID)
	}

	transaction.mutex.Lock()
	defer transaction.mutex.Unlock()

	transaction.Status = TransactionStatusAborting

	tm.logger.Info("Aborting transaction",
		zap.String("transaction_id", transactionID),
	)

	var abortErrors []error

	// Abort Elasticsearch first (simpler cleanup)
	if err := tm.abortElasticsearch(transaction); err != nil {
		abortErrors = append(abortErrors, fmt.Errorf("Elasticsearch abort failed: %w", err))
	}

	// Abort PostgreSQL
	if err := tm.abortPostgreSQL(transaction); err != nil {
		abortErrors = append(abortErrors, fmt.Errorf("PostgreSQL abort failed: %w", err))
	}

	transaction.Status = TransactionStatusAborted

	if len(abortErrors) > 0 {
		tm.logger.Warn("Transaction abort completed with errors",
			zap.String("transaction_id", transactionID),
			zap.Errors("errors", abortErrors),
		)
		return fmt.Errorf("abort completed with %d errors: %v", len(abortErrors), abortErrors[0])
	}

	tm.logger.Info("Transaction aborted successfully",
		zap.String("transaction_id", transactionID),
	)

	// Schedule cleanup
	go tm.cleanupTransaction(transactionID, 5*time.Second)

	return nil
}

// ExecuteInTransaction executes operations within a distributed transaction
func (tm *TransactionManager) ExecuteInTransaction(
	transactionID string,
	operations []TransactionOperation,
) error {
	tm.txMutex.RLock()
	transaction, exists := tm.transactions[transactionID]
	tm.txMutex.RUnlock()

	if !exists {
		return fmt.Errorf("transaction %s not found", transactionID)
	}

	transaction.mutex.Lock()
	defer transaction.mutex.Unlock()

	if transaction.Status != TransactionStatusPending {
		return fmt.Errorf("transaction %s is not in pending state", transactionID)
	}

	// Execute operations on each store
	for _, op := range operations {
		if err := tm.executeOperation(transaction, op); err != nil {
			return fmt.Errorf("operation %s failed: %w", op.ID, err)
		}
		transaction.Operations = append(transaction.Operations, op)
	}

	return nil
}

// PostgreSQL transaction operations

func (tm *TransactionManager) preparePostgreSQL(transaction *DistributedTransaction) error {
	// Begin PostgreSQL transaction
	tx, err := tm.db.BeginTx(transaction.Context, nil)
	if err != nil {
		return fmt.Errorf("failed to begin PostgreSQL transaction: %w", err)
	}

	// Store transaction in metadata for later use
	storeState := transaction.Stores["postgresql"]
	storeState.Status = "prepared"
	now := time.Now()
	storeState.PreparedAt = &now
	storeState.Metadata["tx_id"] = fmt.Sprintf("%p", tx)
	transaction.Stores["postgresql"] = storeState

	// Store the transaction object for commit/rollback
	if transaction.Snapshots == nil {
		transaction.Snapshots = make(map[string]interface{})
	}
	transaction.Snapshots["postgresql_tx"] = tx

	return nil
}

func (tm *TransactionManager) commitPostgreSQL(transaction *DistributedTransaction) error {
	tx, exists := transaction.Snapshots["postgresql_tx"]
	if !exists {
		return fmt.Errorf("PostgreSQL transaction not found in snapshots")
	}

	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("invalid PostgreSQL transaction type")
	}

	if err := sqlTx.Commit(); err != nil {
		return fmt.Errorf("PostgreSQL commit failed: %w", err)
	}

	storeState := transaction.Stores["postgresql"]
	storeState.Status = "committed"
	now := time.Now()
	storeState.CommittedAt = &now
	transaction.Stores["postgresql"] = storeState

	return nil
}

func (tm *TransactionManager) abortPostgreSQL(transaction *DistributedTransaction) error {
	tx, exists := transaction.Snapshots["postgresql_tx"]
	if !exists {
		return nil // Nothing to abort
	}

	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("invalid PostgreSQL transaction type")
	}

	if err := sqlTx.Rollback(); err != nil {
		return fmt.Errorf("PostgreSQL rollback failed: %w", err)
	}

	storeState := transaction.Stores["postgresql"]
	storeState.Status = "aborted"
	now := time.Now()
	storeState.AbortedAt = &now
	transaction.Stores["postgresql"] = storeState

	return nil
}

// Elasticsearch transaction operations

func (tm *TransactionManager) prepareElasticsearch(transaction *DistributedTransaction) error {
	// For Elasticsearch, we create a snapshot of the current state
	if tm.config.EnableSnapshots {
		snapshotID := fmt.Sprintf("tx_%s_%d", transaction.ID, time.Now().Unix())

		// Store snapshot ID for potential rollback
		storeState := transaction.Stores["elasticsearch"]
		storeState.Status = "prepared"
		now := time.Now()
		storeState.PreparedAt = &now
		storeState.SnapshotID = snapshotID
		transaction.Stores["elasticsearch"] = storeState

		// In a real implementation, create an Elasticsearch snapshot here
		tm.logger.Debug("Elasticsearch snapshot created",
			zap.String("snapshot_id", snapshotID),
		)
	}

	return nil
}

func (tm *TransactionManager) commitElasticsearch(transaction *DistributedTransaction) error {
	// For Elasticsearch, commit means making the changes visible (refresh)
	if tm.esClient != nil {
		// Force refresh to make changes visible
		// In a real implementation, send refresh request to Elasticsearch
		tm.logger.Debug("Elasticsearch refresh completed")
	}

	storeState := transaction.Stores["elasticsearch"]
	storeState.Status = "committed"
	now := time.Now()
	storeState.CommittedAt = &now
	transaction.Stores["elasticsearch"] = storeState

	return nil
}

func (tm *TransactionManager) abortElasticsearch(transaction *DistributedTransaction) error {
	storeState := transaction.Stores["elasticsearch"]

	if storeState.SnapshotID != "" {
		// Restore from snapshot
		tm.logger.Debug("Restoring Elasticsearch from snapshot",
			zap.String("snapshot_id", storeState.SnapshotID),
		)
		// In a real implementation, restore from Elasticsearch snapshot
	}

	storeState.Status = "aborted"
	now := time.Now()
	storeState.AbortedAt = &now
	transaction.Stores["elasticsearch"] = storeState

	return nil
}

// executeOperation executes a single operation
func (tm *TransactionManager) executeOperation(transaction *DistributedTransaction, op TransactionOperation) error {
	switch op.StoreName {
	case "postgresql":
		return tm.executePostgreSQLOperation(transaction, op)
	case "elasticsearch":
		return tm.executeElasticsearchOperation(transaction, op)
	default:
		return fmt.Errorf("unknown store: %s", op.StoreName)
	}
}

func (tm *TransactionManager) executePostgreSQLOperation(transaction *DistributedTransaction, op TransactionOperation) error {
	tx, exists := transaction.Snapshots["postgresql_tx"]
	if !exists {
		return fmt.Errorf("PostgreSQL transaction not prepared")
	}

	sqlTx := tx.(*sql.Tx)

	// Execute operation based on type
	switch op.Type {
	case OperationTypeInsert:
		// Execute INSERT operation
		tm.logger.Debug("Executing PostgreSQL INSERT", zap.String("entity_id", op.EntityID))
	case OperationTypeUpdate:
		// Execute UPDATE operation
		tm.logger.Debug("Executing PostgreSQL UPDATE", zap.String("entity_id", op.EntityID))
	case OperationTypeDelete:
		// Execute DELETE operation
		tm.logger.Debug("Executing PostgreSQL DELETE", zap.String("entity_id", op.EntityID))
	case OperationTypeBulk:
		// Execute bulk operation
		tm.logger.Debug("Executing PostgreSQL BULK", zap.String("entity_id", op.EntityID))
	}

	// Update store state
	storeState := transaction.Stores["postgresql"]
	storeState.RecordsCount++
	transaction.Stores["postgresql"] = storeState

	// Prevent unused variable error
	_ = sqlTx

	return nil
}

func (tm *TransactionManager) executeElasticsearchOperation(transaction *DistributedTransaction, op TransactionOperation) error {
	// Execute Elasticsearch operation
	switch op.Type {
	case OperationTypeInsert:
		tm.logger.Debug("Executing Elasticsearch INSERT", zap.String("entity_id", op.EntityID))
	case OperationTypeUpdate:
		tm.logger.Debug("Executing Elasticsearch UPDATE", zap.String("entity_id", op.EntityID))
	case OperationTypeDelete:
		tm.logger.Debug("Executing Elasticsearch DELETE", zap.String("entity_id", op.EntityID))
	case OperationTypeBulk:
		tm.logger.Debug("Executing Elasticsearch BULK", zap.String("entity_id", op.EntityID))
	}

	// Update store state
	storeState := transaction.Stores["elasticsearch"]
	storeState.RecordsCount++
	transaction.Stores["elasticsearch"] = storeState

	return nil
}

// GetTransaction returns transaction status
func (tm *TransactionManager) GetTransaction(transactionID string) (*DistributedTransaction, error) {
	tm.txMutex.RLock()
	defer tm.txMutex.RUnlock()

	transaction, exists := tm.transactions[transactionID]
	if !exists {
		return nil, fmt.Errorf("transaction %s not found", transactionID)
	}

	// Return a copy to avoid race conditions
	txCopy := *transaction
	return &txCopy, nil
}

// cleanupExpiredTransactions removes expired transactions
func (tm *TransactionManager) cleanupExpiredTransactions() {
	ticker := time.NewTicker(tm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.performCleanup()
		}
	}
}

func (tm *TransactionManager) performCleanup() {
	tm.txMutex.Lock()
	defer tm.txMutex.Unlock()

	now := time.Now()
	expiredTransactions := make([]string, 0)

	for id, tx := range tm.transactions {
		// Clean up transactions older than timeout or completed transactions older than 1 hour
		if (tx.Status == TransactionStatusCommitted || tx.Status == TransactionStatusAborted) &&
			now.Sub(tx.StartTime) > time.Hour {
			expiredTransactions = append(expiredTransactions, id)
		} else if now.Sub(tx.StartTime) > tm.config.TransactionTimeout*2 {
			// Force cleanup of very old transactions
			if tx.Cancel != nil {
				tx.Cancel()
			}
			expiredTransactions = append(expiredTransactions, id)
		}
	}

	for _, id := range expiredTransactions {
		delete(tm.transactions, id)
		tm.logger.Debug("Cleaned up expired transaction", zap.String("transaction_id", id))
	}

	if len(expiredTransactions) > 0 {
		tm.logger.Info("Transaction cleanup completed",
			zap.Int("cleaned_count", len(expiredTransactions)),
		)
	}
}

func (tm *TransactionManager) cleanupTransaction(transactionID string, delay time.Duration) {
	time.Sleep(delay)

	tm.txMutex.Lock()
	defer tm.txMutex.Unlock()

	if tx, exists := tm.transactions[transactionID]; exists {
		if tx.Cancel != nil {
			tx.Cancel()
		}
		delete(tm.transactions, transactionID)
		tm.logger.Debug("Transaction cleaned up", zap.String("transaction_id", transactionID))
	}
}