package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
)

// KafkaIntegration manages Kafka operations for learning events
type KafkaIntegration struct {
	config       *KafkaConfig
	logger       *logger.Logger
	producer     KafkaProducerClient
	consumer     KafkaConsumerClient
	eventStore   OverrideEventStore
	isRunning    bool
	stopChannel  chan bool
	wg           sync.WaitGroup
}

// KafkaConfig contains Kafka configuration
type KafkaConfig struct {
	BootstrapServers    []string          `yaml:"bootstrap_servers"`
	SecurityProtocol    string            `yaml:"security_protocol"`
	SASLMechanism      string            `yaml:"sasl_mechanism"`
	SASLUsername       string            `yaml:"sasl_username"`
	SASLPassword       string            `yaml:"sasl_password"`
	ProducerConfig     ProducerConfig    `yaml:"producer"`
	ConsumerConfig     ConsumerConfig    `yaml:"consumer"`
	TopicConfig        TopicConfig       `yaml:"topics"`
	RetryConfig        RetryConfig       `yaml:"retry"`
}

// ProducerConfig contains Kafka producer configuration
type ProducerConfig struct {
	Acks                 string        `yaml:"acks"`
	RetryBackoffMs       int           `yaml:"retry_backoff_ms"`
	BatchSize           int           `yaml:"batch_size"`
	LingerMs            int           `yaml:"linger_ms"`
	CompressionType     string        `yaml:"compression_type"`
	MaxRequestSize      int           `yaml:"max_request_size"`
	EnableIdempotence   bool          `yaml:"enable_idempotence"`
	RequestTimeoutMs    int           `yaml:"request_timeout_ms"`
}

// ConsumerConfig contains Kafka consumer configuration
type ConsumerConfig struct {
	GroupID              string `yaml:"group_id"`
	AutoOffsetReset     string `yaml:"auto_offset_reset"`
	EnableAutoCommit    bool   `yaml:"enable_auto_commit"`
	AutoCommitIntervalMs int   `yaml:"auto_commit_interval_ms"`
	SessionTimeoutMs    int    `yaml:"session_timeout_ms"`
	HeartbeatIntervalMs int    `yaml:"heartbeat_interval_ms"`
	MaxPollRecords      int    `yaml:"max_poll_records"`
}

// TopicConfig contains topic configuration
type TopicConfig struct {
	SafetyDecisions     string `yaml:"safety_decisions"`
	ClinicalOverrides   string `yaml:"clinical_overrides"`
	ClinicalOutcomes    string `yaml:"clinical_outcomes"`
	PerformanceAnalysis string `yaml:"performance_analysis"`
	ReplicationFactor   int    `yaml:"replication_factor"`
	NumPartitions       int    `yaml:"num_partitions"`
}

// RetryConfig contains retry configuration
type RetryConfig struct {
	MaxRetries      int           `yaml:"max_retries"`
	InitialBackoff  time.Duration `yaml:"initial_backoff"`
	MaxBackoff      time.Duration `yaml:"max_backoff"`
	BackoffMultiplier float64     `yaml:"backoff_multiplier"`
}

// Kafka client interfaces (to be implemented with actual Kafka library)
type KafkaProducerClient interface {
	Produce(topic string, key []byte, value []byte) error
	ProduceAsync(topic string, key []byte, value []byte, callback func(error))
	Flush(timeout time.Duration) error
	Close() error
}

type KafkaConsumerClient interface {
	Subscribe(topics []string) error
	Poll(timeout time.Duration) ([]KafkaMessage, error)
	Commit(message KafkaMessage) error
	Close() error
}

// KafkaMessage represents a Kafka message
type KafkaMessage struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Timestamp time.Time
}

// NewKafkaIntegration creates a new Kafka integration
func NewKafkaIntegration(
	config *KafkaConfig,
	eventStore OverrideEventStore,
	logger *logger.Logger,
) *KafkaIntegration {
	if config == nil {
		config = getDefaultKafkaConfig()
	}

	return &KafkaIntegration{
		config:      config,
		logger:      logger,
		eventStore:  eventStore,
		stopChannel: make(chan bool),
	}
}

// Initialize initializes the Kafka integration
func (k *KafkaIntegration) Initialize() error {
	k.logger.Info("Initializing Kafka integration")

	// Initialize producer
	producer, err := k.createProducer()
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	k.producer = producer

	// Initialize consumer
	consumer, err := k.createConsumer()
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}
	k.consumer = consumer

	// Subscribe to topics
	topics := []string{
		k.config.TopicConfig.ClinicalOutcomes,
		k.config.TopicConfig.PerformanceAnalysis,
	}

	if err := k.consumer.Subscribe(topics); err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	k.logger.Info("Kafka integration initialized successfully",
		zap.Strings("subscribed_topics", topics),
	)

	return nil
}

// Start starts the Kafka integration
func (k *KafkaIntegration) Start(ctx context.Context) error {
	if k.isRunning {
		return fmt.Errorf("Kafka integration is already running")
	}

	k.logger.Info("Starting Kafka integration")
	k.isRunning = true

	// Start consumer goroutine
	k.wg.Add(1)
	go k.consumeMessages(ctx)

	return nil
}

// Stop stops the Kafka integration
func (k *KafkaIntegration) Stop() error {
	if !k.isRunning {
		return nil
	}

	k.logger.Info("Stopping Kafka integration")
	k.isRunning = false

	// Send stop signal
	close(k.stopChannel)

	// Wait for goroutines to finish
	k.wg.Wait()

	// Close Kafka clients
	if k.producer != nil {
		if err := k.producer.Close(); err != nil {
			k.logger.Error("Failed to close Kafka producer", zap.Error(err))
		}
	}

	if k.consumer != nil {
		if err := k.consumer.Close(); err != nil {
			k.logger.Error("Failed to close Kafka consumer", zap.Error(err))
		}
	}

	k.logger.Info("Kafka integration stopped")
	return nil
}

// Produce produces a message to Kafka with retry logic
func (k *KafkaIntegration) Produce(topic string, message []byte) error {
	return k.produceWithRetry(topic, nil, message)
}

// ProduceBatch produces multiple messages in a batch
func (k *KafkaIntegration) ProduceBatch(topic string, messages [][]byte) error {
	k.logger.Debug("Producing message batch",
		zap.String("topic", topic),
		zap.Int("message_count", len(messages)),
	)

	for _, message := range messages {
		if err := k.Produce(topic, message); err != nil {
			return fmt.Errorf("failed to produce message in batch: %w", err)
		}
	}

	// Flush to ensure all messages are sent
	return k.producer.Flush(10 * time.Second)
}

// produceWithRetry produces a message with retry logic
func (k *KafkaIntegration) produceWithRetry(topic string, key []byte, value []byte) error {
	backoff := k.config.RetryConfig.InitialBackoff
	
	for attempt := 0; attempt < k.config.RetryConfig.MaxRetries; attempt++ {
		err := k.producer.Produce(topic, key, value)
		if err == nil {
			if attempt > 0 {
				k.logger.Info("Message produced successfully after retries",
					zap.String("topic", topic),
					zap.Int("attempts", attempt+1),
				)
			}
			return nil
		}

		k.logger.Warn("Failed to produce message, retrying",
			zap.String("topic", topic),
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", k.config.RetryConfig.MaxRetries),
			zap.Error(err),
			zap.Duration("backoff", backoff),
		)

		if attempt < k.config.RetryConfig.MaxRetries-1 {
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * k.config.RetryConfig.BackoffMultiplier)
			if backoff > k.config.RetryConfig.MaxBackoff {
				backoff = k.config.RetryConfig.MaxBackoff
			}
		}
	}

	return fmt.Errorf("failed to produce message after %d attempts", k.config.RetryConfig.MaxRetries)
}

// consumeMessages consumes messages from Kafka
func (k *KafkaIntegration) consumeMessages(ctx context.Context) {
	defer k.wg.Done()

	k.logger.Info("Started Kafka message consumer")

	for k.isRunning {
		select {
		case <-k.stopChannel:
			k.logger.Info("Received stop signal, stopping message consumer")
			return
		case <-ctx.Done():
			k.logger.Info("Context cancelled, stopping message consumer")
			return
		default:
			messages, err := k.consumer.Poll(5 * time.Second)
			if err != nil {
				k.logger.Error("Failed to poll messages", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}

			for _, message := range messages {
				if err := k.processMessage(message); err != nil {
					k.logger.Error("Failed to process message",
						zap.String("topic", message.Topic),
						zap.Int32("partition", message.Partition),
						zap.Int64("offset", message.Offset),
						zap.Error(err),
					)
					continue
				}

				// Commit message
				if err := k.consumer.Commit(message); err != nil {
					k.logger.Error("Failed to commit message",
						zap.String("topic", message.Topic),
						zap.Int64("offset", message.Offset),
						zap.Error(err),
					)
				}
			}
		}
	}
}

// processMessage processes a single Kafka message
func (k *KafkaIntegration) processMessage(message KafkaMessage) error {
	k.logger.Debug("Processing Kafka message",
		zap.String("topic", message.Topic),
		zap.Int32("partition", message.Partition),
		zap.Int64("offset", message.Offset),
	)

	switch message.Topic {
	case k.config.TopicConfig.ClinicalOutcomes:
		return k.processClinicalOutcomeMessage(message)
	case k.config.TopicConfig.PerformanceAnalysis:
		return k.processPerformanceAnalysisMessage(message)
	default:
		k.logger.Warn("Received message from unknown topic",
			zap.String("topic", message.Topic),
		)
		return nil
	}
}

// processClinicalOutcomeMessage processes clinical outcome messages
func (k *KafkaIntegration) processClinicalOutcomeMessage(message KafkaMessage) error {
	var outcomeEvent ClinicalOutcomeEvent
	if err := json.Unmarshal(message.Value, &outcomeEvent); err != nil {
		return fmt.Errorf("failed to unmarshal clinical outcome event: %w", err)
	}

	// Store the outcome event
	if err := k.eventStore.StoreOutcomeEvent(&outcomeEvent); err != nil {
		return fmt.Errorf("failed to store outcome event: %w", err)
	}

	k.logger.Debug("Processed clinical outcome event",
		zap.String("event_id", outcomeEvent.EventID),
		zap.String("patient_id", outcomeEvent.PatientID),
		zap.String("outcome_type", outcomeEvent.OutcomeType),
	)

	return nil
}

// processPerformanceAnalysisMessage processes performance analysis messages
func (k *KafkaIntegration) processPerformanceAnalysisMessage(message KafkaMessage) error {
	var performanceEvent PerformanceAnalysisEvent
	if err := json.Unmarshal(message.Value, &performanceEvent); err != nil {
		return fmt.Errorf("failed to unmarshal performance analysis event: %w", err)
	}

	k.logger.Debug("Processed performance analysis event",
		zap.String("event_id", performanceEvent.EventID),
		zap.String("analysis_type", performanceEvent.AnalysisType),
		zap.String("time_window", performanceEvent.TimeWindow),
	)

	// Here you could trigger additional analysis or alerts based on performance events
	
	return nil
}

// createProducer creates a Kafka producer (placeholder implementation)
func (k *KafkaIntegration) createProducer() (KafkaProducerClient, error) {
	// This would create an actual Kafka producer using a library like confluent-kafka-go
	// For now, returning a mock producer
	return &MockKafkaProducer{
		logger: k.logger,
		config: k.config,
	}, nil
}

// createConsumer creates a Kafka consumer (placeholder implementation)
func (k *KafkaIntegration) createConsumer() (KafkaConsumerClient, error) {
	// This would create an actual Kafka consumer using a library like confluent-kafka-go
	// For now, returning a mock consumer
	return &MockKafkaConsumer{
		logger: k.logger,
		config: k.config,
	}, nil
}

// CreateTopics creates necessary Kafka topics
func (k *KafkaIntegration) CreateTopics() error {
	k.logger.Info("Creating Kafka topics")

	topics := map[string]TopicSpec{
		k.config.TopicConfig.SafetyDecisions: {
			Name:              k.config.TopicConfig.SafetyDecisions,
			NumPartitions:     k.config.TopicConfig.NumPartitions,
			ReplicationFactor: k.config.TopicConfig.ReplicationFactor,
		},
		k.config.TopicConfig.ClinicalOverrides: {
			Name:              k.config.TopicConfig.ClinicalOverrides,
			NumPartitions:     k.config.TopicConfig.NumPartitions,
			ReplicationFactor: k.config.TopicConfig.ReplicationFactor,
		},
		k.config.TopicConfig.ClinicalOutcomes: {
			Name:              k.config.TopicConfig.ClinicalOutcomes,
			NumPartitions:     k.config.TopicConfig.NumPartitions,
			ReplicationFactor: k.config.TopicConfig.ReplicationFactor,
		},
		k.config.TopicConfig.PerformanceAnalysis: {
			Name:              k.config.TopicConfig.PerformanceAnalysis,
			NumPartitions:     k.config.TopicConfig.NumPartitions,
			ReplicationFactor: k.config.TopicConfig.ReplicationFactor,
		},
	}

	for _, topic := range topics {
		k.logger.Info("Creating topic",
			zap.String("topic_name", topic.Name),
			zap.Int("partitions", topic.NumPartitions),
			zap.Int("replication_factor", topic.ReplicationFactor),
		)
		// Topic creation logic would go here using Kafka admin client
	}

	return nil
}

// GetIntegrationMetrics returns Kafka integration metrics
func (k *KafkaIntegration) GetIntegrationMetrics() map[string]interface{} {
	return map[string]interface{}{
		"integration_version": "1.0.0",
		"is_running":          k.isRunning,
		"bootstrap_servers":   k.config.BootstrapServers,
		"security_protocol":   k.config.SecurityProtocol,
		"producer_config":     k.config.ProducerConfig,
		"consumer_config":     k.config.ConsumerConfig,
		"topic_config":        k.config.TopicConfig,
	}
}

// TopicSpec represents a Kafka topic specification
type TopicSpec struct {
	Name              string
	NumPartitions     int
	ReplicationFactor int
}

// getDefaultKafkaConfig returns default Kafka configuration
func getDefaultKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		BootstrapServers: []string{"localhost:9092"},
		SecurityProtocol: "PLAINTEXT",
		ProducerConfig: ProducerConfig{
			Acks:               "all",
			RetryBackoffMs:     100,
			BatchSize:          16384,
			LingerMs:           1,
			CompressionType:    "snappy",
			MaxRequestSize:     1048576,
			EnableIdempotence:  true,
			RequestTimeoutMs:   30000,
		},
		ConsumerConfig: ConsumerConfig{
			GroupID:              "clinical-learning-consumer",
			AutoOffsetReset:      "earliest",
			EnableAutoCommit:     false,
			AutoCommitIntervalMs: 5000,
			SessionTimeoutMs:     10000,
			HeartbeatIntervalMs:  3000,
			MaxPollRecords:       500,
		},
		TopicConfig: TopicConfig{
			SafetyDecisions:     "clinical-learning-safety-decisions",
			ClinicalOverrides:   "clinical-learning-clinical-overrides",
			ClinicalOutcomes:    "clinical-learning-clinical-outcomes",
			PerformanceAnalysis: "clinical-learning-performance-analysis",
			ReplicationFactor:   3,
			NumPartitions:       6,
		},
		RetryConfig: RetryConfig{
			MaxRetries:        3,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        5 * time.Second,
			BackoffMultiplier: 2.0,
		},
	}
}

// Mock implementations for testing/demo purposes

// MockKafkaProducer is a mock Kafka producer
type MockKafkaProducer struct {
	logger *logger.Logger
	config *KafkaConfig
}

func (m *MockKafkaProducer) Produce(topic string, key []byte, value []byte) error {
	m.logger.Debug("Mock: Producing message",
		zap.String("topic", topic),
		zap.Int("value_size", len(value)),
	)
	// Simulate successful production
	return nil
}

func (m *MockKafkaProducer) ProduceAsync(topic string, key []byte, value []byte, callback func(error)) {
	m.logger.Debug("Mock: Producing message async",
		zap.String("topic", topic),
		zap.Int("value_size", len(value)),
	)
	// Simulate successful production
	go func() {
		time.Sleep(10 * time.Millisecond) // Simulate network delay
		callback(nil)
	}()
}

func (m *MockKafkaProducer) Flush(timeout time.Duration) error {
	m.logger.Debug("Mock: Flushing producer", zap.Duration("timeout", timeout))
	return nil
}

func (m *MockKafkaProducer) Close() error {
	m.logger.Debug("Mock: Closing producer")
	return nil
}

// MockKafkaConsumer is a mock Kafka consumer
type MockKafkaConsumer struct {
	logger *logger.Logger
	config *KafkaConfig
	topics []string
}

func (m *MockKafkaConsumer) Subscribe(topics []string) error {
	m.logger.Debug("Mock: Subscribing to topics", zap.Strings("topics", topics))
	m.topics = topics
	return nil
}

func (m *MockKafkaConsumer) Poll(timeout time.Duration) ([]KafkaMessage, error) {
	// For mock purposes, return empty messages
	// In real implementation, this would poll for actual messages
	time.Sleep(100 * time.Millisecond) // Simulate polling delay
	return []KafkaMessage{}, nil
}

func (m *MockKafkaConsumer) Commit(message KafkaMessage) error {
	m.logger.Debug("Mock: Committing message",
		zap.String("topic", message.Topic),
		zap.Int64("offset", message.Offset),
	)
	return nil
}

func (m *MockKafkaConsumer) Close() error {
	m.logger.Debug("Mock: Closing consumer")
	return nil
}

// ValidateKafkaConnection validates the connection to Kafka
func (k *KafkaIntegration) ValidateConnection() error {
	k.logger.Info("Validating Kafka connection")

	// Test producer connection
	testMessage := []byte(`{"test": "connection_validation", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`)
	testTopic := strings.TrimSuffix(k.config.TopicConfig.SafetyDecisions, "-safety-decisions") + "-connection-test"

	if err := k.Produce(testTopic, testMessage); err != nil {
		return fmt.Errorf("failed to produce test message: %w", err)
	}

	k.logger.Info("Kafka connection validated successfully")
	return nil
}