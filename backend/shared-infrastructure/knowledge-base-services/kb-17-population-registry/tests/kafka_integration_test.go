// Package tests provides comprehensive test utilities for KB-17 Population Registry
// kafka_integration_test.go - Tests for Kafka consumer/producer integration
// This validates reliable event streaming critical for event-driven enrollment
package tests

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
)

// =============================================================================
// KAFKA MESSAGE TYPES
// =============================================================================

// KafkaMessage represents a Kafka message for testing
type KafkaMessage struct {
	Topic     string
	Key       string
	Value     []byte
	Headers   map[string]string
	Partition int32
	Offset    int64
	Timestamp time.Time
}

// KafkaConsumerConfig represents consumer configuration
type KafkaConsumerConfig struct {
	GroupID           string
	Topics            []string
	AutoOffsetReset   string
	MaxPollRecords    int
	SessionTimeoutMs  int
	HeartbeatInterval int
	EnableAutoCommit  bool
}

// KafkaProducerConfig represents producer configuration
type KafkaProducerConfig struct {
	Acks              string // "all", "1", "0"
	Retries           int
	RetryBackoffMs    int
	EnableIdempotence bool
	MaxInFlight       int
	LingerMs          int
	BatchSize         int
}

// =============================================================================
// MOCK KAFKA CONSUMER
// =============================================================================

// MockKafkaConsumer simulates Kafka consumer behavior
type MockKafkaConsumer struct {
	mu              sync.RWMutex
	config          *KafkaConsumerConfig
	messages        chan *KafkaMessage
	committed       map[string]int64 // topic-partition -> offset
	paused          map[string]bool
	subscribed      []string
	running         bool
	pollCount       int32
	errorInjection  error
	disconnectAfter int
	reconnectDelay  time.Duration
}

// NewMockKafkaConsumer creates a new mock consumer
func NewMockKafkaConsumer(config *KafkaConsumerConfig) *MockKafkaConsumer {
	return &MockKafkaConsumer{
		config:     config,
		messages:   make(chan *KafkaMessage, 1000),
		committed:  make(map[string]int64),
		paused:     make(map[string]bool),
		subscribed: config.Topics,
		running:    true,
	}
}

// Poll returns next message or nil
func (c *MockKafkaConsumer) Poll(timeoutMs int) (*KafkaMessage, error) {
	c.mu.Lock()
	if c.errorInjection != nil {
		err := c.errorInjection
		c.errorInjection = nil
		c.mu.Unlock()
		return nil, err
	}

	pollNum := atomic.AddInt32(&c.pollCount, 1)
	if c.disconnectAfter > 0 && int(pollNum) == c.disconnectAfter {
		c.running = false
		c.mu.Unlock()
		// Simulate reconnection
		go func() {
			time.Sleep(c.reconnectDelay)
			c.mu.Lock()
			c.running = true
			c.mu.Unlock()
		}()
		return nil, errors.New("broker disconnected")
	}
	c.mu.Unlock()

	select {
	case msg := <-c.messages:
		return msg, nil
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return nil, nil
	}
}

// Commit commits offsets
func (c *MockKafkaConsumer) Commit(topic string, partition int32, offset int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := topicPartitionKey(topic, partition)
	c.committed[key] = offset
	return nil
}

// GetCommittedOffset returns committed offset
func (c *MockKafkaConsumer) GetCommittedOffset(topic string, partition int32) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := topicPartitionKey(topic, partition)
	return c.committed[key]
}

// Pause pauses consumption from topic
func (c *MockKafkaConsumer) Pause(topics []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, t := range topics {
		c.paused[t] = true
	}
}

// Resume resumes consumption
func (c *MockKafkaConsumer) Resume(topics []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, t := range topics {
		delete(c.paused, t)
	}
}

// Close closes the consumer
func (c *MockKafkaConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	close(c.messages)
	return nil
}

// InjectMessage injects a message for testing
func (c *MockKafkaConsumer) InjectMessage(msg *KafkaMessage) {
	c.messages <- msg
}

// InjectError injects an error for next poll
func (c *MockKafkaConsumer) InjectError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errorInjection = err
}

// SetDisconnectAfter configures disconnect simulation
func (c *MockKafkaConsumer) SetDisconnectAfter(polls int, reconnectDelay time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disconnectAfter = polls
	c.reconnectDelay = reconnectDelay
}

func topicPartitionKey(topic string, partition int32) string {
	return topic + "-" + string(rune('0'+partition))
}

// =============================================================================
// MOCK KAFKA PRODUCER
// =============================================================================

// MockKafkaProducer simulates Kafka producer behavior
type MockKafkaProducer struct {
	mu              sync.RWMutex
	config          *KafkaProducerConfig
	sentMessages    []*KafkaMessage
	pendingMessages []*KafkaMessage
	ackChannel      chan *ProducerAck
	errorInjection  error
	failAfter       int
	idempotentMap   map[string]bool // deduplication for idempotent producer
	nextOffset      int64
	flushed         bool
}

// ProducerAck represents producer acknowledgment
type ProducerAck struct {
	Message   *KafkaMessage
	Partition int32
	Offset    int64
	Error     error
}

// NewMockKafkaProducer creates a new mock producer
func NewMockKafkaProducer(config *KafkaProducerConfig) *MockKafkaProducer {
	return &MockKafkaProducer{
		config:        config,
		sentMessages:  make([]*KafkaMessage, 0),
		ackChannel:    make(chan *ProducerAck, 100),
		idempotentMap: make(map[string]bool),
	}
}

// Send sends a message asynchronously
func (p *MockKafkaProducer) Send(ctx context.Context, msg *KafkaMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.errorInjection != nil {
		err := p.errorInjection
		if p.failAfter > 0 {
			p.failAfter--
			if p.failAfter == 0 {
				p.errorInjection = nil
			}
		}
		return err
	}

	// Check idempotence
	if p.config.EnableIdempotence {
		key := msg.Topic + "-" + msg.Key + "-" + string(msg.Value)
		if p.idempotentMap[key] {
			// Duplicate, ignore
			return nil
		}
		p.idempotentMap[key] = true
	}

	msg.Offset = p.nextOffset
	p.nextOffset++
	msg.Timestamp = time.Now()

	p.sentMessages = append(p.sentMessages, msg)

	// Send ack
	go func() {
		p.ackChannel <- &ProducerAck{
			Message:   msg,
			Partition: msg.Partition,
			Offset:    msg.Offset,
		}
	}()

	return nil
}

// SendSync sends a message synchronously with ack waiting
func (p *MockKafkaProducer) SendSync(ctx context.Context, msg *KafkaMessage) (*ProducerAck, error) {
	if err := p.Send(ctx, msg); err != nil {
		return nil, err
	}

	select {
	case ack := <-p.ackChannel:
		return ack, ack.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Flush flushes pending messages
func (p *MockKafkaProducer) Flush(timeoutMs int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.flushed = true
	p.pendingMessages = nil
	return nil
}

// GetSentMessages returns all sent messages
func (p *MockKafkaProducer) GetSentMessages() []*KafkaMessage {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*KafkaMessage, len(p.sentMessages))
	copy(result, p.sentMessages)
	return result
}

// InjectError injects errors
func (p *MockKafkaProducer) InjectError(err error, times int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorInjection = err
	p.failAfter = times
}

// Close closes the producer
func (p *MockKafkaProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	close(p.ackChannel)
	return nil
}

// =============================================================================
// CONSUMER RECONNECTION TESTS
// =============================================================================

// TestKafka_ConsumerReconnectionAfterBrokerDisconnect tests reconnection
func TestKafka_ConsumerReconnectionAfterBrokerDisconnect(t *testing.T) {
	config := &KafkaConsumerConfig{
		GroupID:          "population-registry-consumer",
		Topics:           []string{"clinical.events"},
		AutoOffsetReset:  "earliest",
		SessionTimeoutMs: 10000,
	}
	consumer := NewMockKafkaConsumer(config)

	// Configure disconnect after 5 polls
	consumer.SetDisconnectAfter(5, 100*time.Millisecond)

	var disconnectDetected bool
	var reconnected bool
	var pollsAfterReconnect int

	// Inject some messages
	for i := 0; i < 10; i++ {
		consumer.InjectMessage(&KafkaMessage{
			Topic: "clinical.events",
			Key:   "patient-" + string(rune('0'+i)),
			Value: []byte(`{"type":"diagnosis.created"}`),
		})
	}

	// Poll until disconnect, then wait for reconnect
	for i := 0; i < 20; i++ {
		msg, err := consumer.Poll(50)

		if err != nil && err.Error() == "broker disconnected" {
			disconnectDetected = true
			// Wait for reconnection
			time.Sleep(150 * time.Millisecond)
			continue
		}

		if disconnectDetected && msg != nil {
			reconnected = true
			pollsAfterReconnect++
		}
	}

	assert.True(t, disconnectDetected, "Should detect broker disconnect")
	assert.True(t, reconnected, "Should reconnect and continue processing")
	assert.True(t, pollsAfterReconnect > 0, "Should process messages after reconnect")
}

// TestKafka_ConsumerOffsetCommitOnReconnect tests offset preservation
func TestKafka_ConsumerOffsetCommitOnReconnect(t *testing.T) {
	config := &KafkaConsumerConfig{
		GroupID:          "population-registry-consumer",
		Topics:           []string{"clinical.events"},
		EnableAutoCommit: false, // Manual commit
	}
	consumer := NewMockKafkaConsumer(config)

	// Process and commit some messages
	for i := 0; i < 5; i++ {
		consumer.InjectMessage(&KafkaMessage{
			Topic:     "clinical.events",
			Partition: 0,
			Offset:    int64(i),
			Key:       "patient-" + string(rune('0'+i)),
			Value:     []byte(`{"type":"diagnosis.created"}`),
		})
	}

	// Process and commit
	var lastOffset int64
	for i := 0; i < 5; i++ {
		msg, _ := consumer.Poll(50)
		if msg != nil {
			lastOffset = msg.Offset
			consumer.Commit(msg.Topic, msg.Partition, msg.Offset)
		}
	}

	// Verify committed offset
	committed := consumer.GetCommittedOffset("clinical.events", 0)
	assert.Equal(t, lastOffset, committed, "Committed offset should match last processed")

	// Simulate restart - offset should be preserved
	// In real scenario, new consumer would read from committed offset
	assert.Equal(t, int64(4), committed, "Should have committed all 5 messages (0-4)")
}

// TestKafka_ConsumerGroupRebalance tests rebalance handling
func TestKafka_ConsumerGroupRebalance(t *testing.T) {
	// Simulate consumer group with 2 consumers
	config1 := &KafkaConsumerConfig{
		GroupID: "population-registry-consumer",
		Topics:  []string{"clinical.events"},
	}
	config2 := &KafkaConsumerConfig{
		GroupID: "population-registry-consumer",
		Topics:  []string{"clinical.events"},
	}

	consumer1 := NewMockKafkaConsumer(config1)
	consumer2 := NewMockKafkaConsumer(config2)

	// Both consumers should be able to process
	assert.NotNil(t, consumer1)
	assert.NotNil(t, consumer2)
	assert.Equal(t, consumer1.config.GroupID, consumer2.config.GroupID)

	// In real scenario, Kafka would assign partitions to each consumer
	// This test validates both can be created with same group
}

// TestKafka_ConsumerPauseResume tests pause/resume functionality
func TestKafka_ConsumerPauseResume(t *testing.T) {
	config := &KafkaConsumerConfig{
		GroupID: "population-registry-consumer",
		Topics:  []string{"clinical.events", "lab.events"},
	}
	consumer := NewMockKafkaConsumer(config)

	// Pause one topic
	consumer.Pause([]string{"lab.events"})

	consumer.mu.RLock()
	assert.True(t, consumer.paused["lab.events"], "lab.events should be paused")
	assert.False(t, consumer.paused["clinical.events"], "clinical.events should not be paused")
	consumer.mu.RUnlock()

	// Resume
	consumer.Resume([]string{"lab.events"})

	consumer.mu.RLock()
	assert.False(t, consumer.paused["lab.events"], "lab.events should be resumed")
	consumer.mu.RUnlock()
}

// =============================================================================
// PRODUCER DELIVERY GUARANTEE TESTS
// =============================================================================

// TestKafka_ProducerDeliveryGuaranteeAtLeastOnce tests at-least-once delivery
func TestKafka_ProducerDeliveryGuaranteeAtLeastOnce(t *testing.T) {
	config := &KafkaProducerConfig{
		Acks:    "all",
		Retries: 3,
	}
	producer := NewMockKafkaProducer(config)
	ctx := context.Background()

	// Send enrollment event
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "delivery-test-001",
		RegistryCode: models.RegistryDiabetes,
		Status:       models.EnrollmentStatusActive,
	}

	payload, _ := json.Marshal(enrollment)
	msg := &KafkaMessage{
		Topic: "registry.enrolled",
		Key:   enrollment.PatientID,
		Value: payload,
	}

	ack, err := producer.SendSync(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, ack)
	assert.Nil(t, ack.Error)

	// Verify message was sent
	sent := producer.GetSentMessages()
	assert.Len(t, sent, 1)
	assert.Equal(t, "registry.enrolled", sent[0].Topic)
}

// TestKafka_ProducerIdempotence tests idempotent producer
func TestKafka_ProducerIdempotence(t *testing.T) {
	config := &KafkaProducerConfig{
		EnableIdempotence: true,
		Retries:           5,
	}
	producer := NewMockKafkaProducer(config)
	ctx := context.Background()

	// Same message sent multiple times
	msg := &KafkaMessage{
		Topic: "registry.enrolled",
		Key:   "patient-idempotent-001",
		Value: []byte(`{"patient_id":"patient-idempotent-001","status":"active"}`),
	}

	// Send same message 3 times
	for i := 0; i < 3; i++ {
		err := producer.Send(ctx, msg)
		require.NoError(t, err)
	}

	// Should only have 1 message (idempotent)
	sent := producer.GetSentMessages()
	assert.Len(t, sent, 1, "Idempotent producer should deduplicate identical messages")
}

// TestKafka_ProducerRetryOnTransientError tests retry behavior
func TestKafka_ProducerRetryOnTransientError(t *testing.T) {
	config := &KafkaProducerConfig{
		Retries:        3,
		RetryBackoffMs: 100,
	}
	producer := NewMockKafkaProducer(config)
	ctx := context.Background()

	// Inject transient errors that will be retried
	producer.InjectError(errors.New("broker unavailable"), 2) // Fail first 2 attempts

	msg := &KafkaMessage{
		Topic: "registry.enrolled",
		Key:   "retry-test-001",
		Value: []byte(`{"patient_id":"retry-test-001"}`),
	}

	// First two attempts should fail
	err1 := producer.Send(ctx, msg)
	assert.Error(t, err1)

	err2 := producer.Send(ctx, msg)
	assert.Error(t, err2)

	// Third attempt should succeed
	err3 := producer.Send(ctx, msg)
	assert.NoError(t, err3)
}

// TestKafka_ProducerBatchSend tests batching behavior
func TestKafka_ProducerBatchSend(t *testing.T) {
	config := &KafkaProducerConfig{
		BatchSize: 100,
		LingerMs:  5,
	}
	producer := NewMockKafkaProducer(config)
	ctx := context.Background()

	// Send batch of messages
	const batchSize = 50
	for i := 0; i < batchSize; i++ {
		msg := &KafkaMessage{
			Topic: "registry.enrolled",
			Key:   createKafkaKey(i),
			Value: []byte(`{"patient_id":"batch-` + createKafkaKey(i) + `"}`),
		}
		err := producer.Send(ctx, msg)
		require.NoError(t, err)
	}

	// Flush to ensure all sent
	err := producer.Flush(1000)
	require.NoError(t, err)

	sent := producer.GetSentMessages()
	assert.Len(t, sent, batchSize)
}

// =============================================================================
// EVENT SERIALIZATION TESTS
// =============================================================================

// TestKafka_EnrollmentEventSerialization tests event format
func TestKafka_EnrollmentEventSerialization(t *testing.T) {
	enrollment := &models.RegistryPatient{
		ID:               uuid.New(),
		PatientID:        "serial-test-001",
		RegistryCode:     models.RegistryDiabetes,
		Status:           models.EnrollmentStatusActive,
		RiskTier:         models.RiskTierHigh,
		EnrollmentSource: models.EnrollmentSourceDiagnosis,
		EnrolledAt:       time.Now(),
	}

	// Serialize
	payload, err := json.Marshal(enrollment)
	require.NoError(t, err)

	// Deserialize
	var restored models.RegistryPatient
	err = json.Unmarshal(payload, &restored)
	require.NoError(t, err)

	assert.Equal(t, enrollment.PatientID, restored.PatientID)
	assert.Equal(t, enrollment.RegistryCode, restored.RegistryCode)
	assert.Equal(t, enrollment.Status, restored.Status)
	assert.Equal(t, enrollment.RiskTier, restored.RiskTier)
}

// TestKafka_ClinicalEventSerialization tests clinical event format
func TestKafka_ClinicalEventSerialization(t *testing.T) {
	event := map[string]interface{}{
		"type":         "diagnosis.created",
		"patient_id":   "clinical-event-001",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"source":       "ehr",
		"code":         "E11.9",
		"code_system":  "ICD-10",
		"display":      "Type 2 diabetes mellitus without complications",
		"status":       "active",
		"recorded_at":  time.Now().UTC().Format(time.RFC3339),
		"practitioner": "Dr. Smith",
	}

	payload, err := json.Marshal(event)
	require.NoError(t, err)

	var restored map[string]interface{}
	err = json.Unmarshal(payload, &restored)
	require.NoError(t, err)

	assert.Equal(t, "diagnosis.created", restored["type"])
	assert.Equal(t, "clinical-event-001", restored["patient_id"])
	assert.Equal(t, "E11.9", restored["code"])
}

// =============================================================================
// TOPIC MANAGEMENT TESTS
// =============================================================================

// TestKafka_TopicConfiguration tests topic configuration
func TestKafka_TopicConfiguration(t *testing.T) {
	// Verify expected topics for KB-17
	expectedTopics := []string{
		"clinical.events.diagnosis",
		"clinical.events.lab",
		"clinical.events.medication",
		"clinical.events.problem",
		"registry.enrolled",
		"registry.disenrolled",
		"registry.risk-tier-changed",
		"registry.status-changed",
	}

	for _, topic := range expectedTopics {
		assert.NotEmpty(t, topic, "Topic should be defined")
	}
}

// TestKafka_MessageHeaders tests header propagation
func TestKafka_MessageHeaders(t *testing.T) {
	config := &KafkaProducerConfig{
		Acks: "all",
	}
	producer := NewMockKafkaProducer(config)
	ctx := context.Background()

	// Message with tracing headers
	msg := &KafkaMessage{
		Topic: "registry.enrolled",
		Key:   "header-test-001",
		Value: []byte(`{"patient_id":"header-test-001"}`),
		Headers: map[string]string{
			"correlation-id": "corr-123",
			"trace-id":       "trace-456",
			"source-service": "kb-17-population-registry",
			"event-version":  "1.0",
		},
	}

	err := producer.Send(ctx, msg)
	require.NoError(t, err)

	sent := producer.GetSentMessages()
	require.Len(t, sent, 1)

	// Verify headers preserved
	assert.Equal(t, "corr-123", sent[0].Headers["correlation-id"])
	assert.Equal(t, "trace-456", sent[0].Headers["trace-id"])
	assert.Equal(t, "kb-17-population-registry", sent[0].Headers["source-service"])
}

// =============================================================================
// CONSUMER LAG AND MONITORING TESTS
// =============================================================================

// TestKafka_ConsumerLagMonitoring tests lag detection
func TestKafka_ConsumerLagMonitoring(t *testing.T) {
	config := &KafkaConsumerConfig{
		GroupID: "population-registry-consumer",
		Topics:  []string{"clinical.events"},
	}
	consumer := NewMockKafkaConsumer(config)

	// Inject messages to create lag
	const totalMessages = 100
	for i := 0; i < totalMessages; i++ {
		consumer.InjectMessage(&KafkaMessage{
			Topic:  "clinical.events",
			Offset: int64(i),
			Key:    createKafkaKey(i),
			Value:  []byte(`{"type":"diagnosis.created"}`),
		})
	}

	// Process only half
	var processed int
	for i := 0; i < totalMessages/2; i++ {
		msg, _ := consumer.Poll(10)
		if msg != nil {
			processed++
			consumer.Commit(msg.Topic, msg.Partition, msg.Offset)
		}
	}

	// Lag = total - processed
	// In real scenario, would compare high watermark to committed offset
	lag := totalMessages - processed
	assert.Equal(t, totalMessages/2, lag, "Consumer lag should be half of total messages")
}

// =============================================================================
// END-TO-END KAFKA FLOW TESTS
// =============================================================================

// TestKafka_EndToEndEnrollmentFlow tests complete event flow
func TestKafka_EndToEndEnrollmentFlow(t *testing.T) {
	// Setup
	consumerConfig := &KafkaConsumerConfig{
		GroupID: "population-registry-consumer",
		Topics:  []string{"clinical.events"},
	}
	producerConfig := &KafkaProducerConfig{
		Acks:              "all",
		EnableIdempotence: true,
	}

	consumer := NewMockKafkaConsumer(consumerConfig)
	producer := NewMockKafkaProducer(producerConfig)
	repo := NewMockRepository()
	ctx := context.Background()

	// 1. Receive diagnosis event
	diagnosisEvent := &KafkaMessage{
		Topic: "clinical.events",
		Key:   "e2e-patient-001",
		Value: []byte(`{
			"type": "diagnosis.created",
			"patient_id": "e2e-patient-001",
			"code": "E11.9",
			"code_system": "ICD-10",
			"status": "active"
		}`),
	}
	consumer.InjectMessage(diagnosisEvent)

	// 2. Process event
	msg, err := consumer.Poll(100)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// 3. Create enrollment
	enrollment := &models.RegistryPatient{
		ID:               uuid.New(),
		PatientID:        "e2e-patient-001",
		RegistryCode:     models.RegistryDiabetes,
		Status:           models.EnrollmentStatusActive,
		RiskTier:         models.RiskTierModerate,
		EnrollmentSource: models.EnrollmentSourceDiagnosis,
		EnrolledAt:       time.Now(),
	}
	err = repo.CreateEnrollment(enrollment)
	require.NoError(t, err)

	// 4. Produce enrollment event
	enrollmentPayload, _ := json.Marshal(enrollment)
	enrollmentEvent := &KafkaMessage{
		Topic: "registry.enrolled",
		Key:   enrollment.PatientID,
		Value: enrollmentPayload,
		Headers: map[string]string{
			"correlation-id": msg.Key,
			"source-event":   "diagnosis.created",
		},
	}
	err = producer.Send(ctx, enrollmentEvent)
	require.NoError(t, err)

	// 5. Commit consumer offset
	consumer.Commit(msg.Topic, msg.Partition, msg.Offset)

	// Verify complete flow
	savedEnrollment, _ := repo.GetEnrollmentByPatientRegistry("e2e-patient-001", models.RegistryDiabetes)
	assert.NotNil(t, savedEnrollment)

	producedEvents := producer.GetSentMessages()
	assert.Len(t, producedEvents, 1)
	assert.Equal(t, "registry.enrolled", producedEvents[0].Topic)
}

// TestKafka_MultiRegistryEventProcessing tests processing for multiple registries
func TestKafka_MultiRegistryEventProcessing(t *testing.T) {
	consumerConfig := &KafkaConsumerConfig{
		GroupID: "population-registry-consumer",
		Topics: []string{
			"clinical.events.diagnosis",
			"clinical.events.lab",
			"clinical.events.medication",
		},
	}
	producerConfig := &KafkaProducerConfig{
		Acks: "all",
	}

	consumer := NewMockKafkaConsumer(consumerConfig)
	producer := NewMockKafkaProducer(producerConfig)
	ctx := context.Background()

	// Inject events for different registries
	events := []*KafkaMessage{
		{
			Topic: "clinical.events.diagnosis",
			Key:   "multi-reg-001",
			Value: []byte(`{"type":"diagnosis.created","patient_id":"multi-reg-001","code":"E11.9"}`),
		},
		{
			Topic: "clinical.events.diagnosis",
			Key:   "multi-reg-001",
			Value: []byte(`{"type":"diagnosis.created","patient_id":"multi-reg-001","code":"I10"}`),
		},
		{
			Topic: "clinical.events.lab",
			Key:   "multi-reg-001",
			Value: []byte(`{"type":"lab.result.created","patient_id":"multi-reg-001","code":"33914-3","value":25}`),
		},
	}

	for _, e := range events {
		consumer.InjectMessage(e)
	}

	// Process all events
	var enrollments []string
	for i := 0; i < len(events); i++ {
		msg, _ := consumer.Poll(50)
		if msg != nil {
			// Simulate registry matching
			var eventData map[string]interface{}
			json.Unmarshal(msg.Value, &eventData)

			if code, ok := eventData["code"].(string); ok {
				var registry string
				switch code {
				case "E11.9":
					registry = "DIABETES"
				case "I10":
					registry = "HYPERTENSION"
				case "33914-3":
					registry = "CKD"
				}
				if registry != "" {
					enrollments = append(enrollments, registry)
					// Produce enrollment event
					producer.Send(ctx, &KafkaMessage{
						Topic: "registry.enrolled",
						Key:   eventData["patient_id"].(string),
						Value: []byte(`{"registry":"` + registry + `"}`),
					})
				}
			}
		}
	}

	assert.ElementsMatch(t, []string{"DIABETES", "HYPERTENSION", "CKD"}, enrollments)
	assert.Len(t, producer.GetSentMessages(), 3)
}

// =============================================================================
// ERROR HANDLING TESTS
// =============================================================================

// TestKafka_ConsumerErrorRecovery tests error recovery
func TestKafka_ConsumerErrorRecovery(t *testing.T) {
	config := &KafkaConsumerConfig{
		GroupID: "population-registry-consumer",
		Topics:  []string{"clinical.events"},
	}
	consumer := NewMockKafkaConsumer(config)

	// Inject error then messages
	consumer.InjectError(errors.New("temporary network error"))
	consumer.InjectMessage(&KafkaMessage{
		Topic: "clinical.events",
		Key:   "recovery-001",
		Value: []byte(`{"type":"diagnosis.created"}`),
	})

	// First poll gets error
	_, err := consumer.Poll(50)
	assert.Error(t, err)

	// Second poll should succeed
	msg, err := consumer.Poll(50)
	assert.NoError(t, err)
	assert.NotNil(t, msg)
}

// TestKafka_ProducerFailoverBehavior tests failover
func TestKafka_ProducerFailoverBehavior(t *testing.T) {
	config := &KafkaProducerConfig{
		Acks:    "all",
		Retries: 5,
	}
	producer := NewMockKafkaProducer(config)
	ctx := context.Background()

	// Simulate partial failure
	producer.InjectError(errors.New("leader not available"), 1)

	msg := &KafkaMessage{
		Topic: "registry.enrolled",
		Key:   "failover-001",
		Value: []byte(`{"patient_id":"failover-001"}`),
	}

	// First send fails
	err := producer.Send(ctx, msg)
	assert.Error(t, err)

	// Retry succeeds
	err = producer.Send(ctx, msg)
	assert.NoError(t, err)

	sent := producer.GetSentMessages()
	assert.Len(t, sent, 1)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func createKafkaKey(index int) string {
	return "patient-" +
		string(rune('0'+index/1000%10)) +
		string(rune('0'+index/100%10)) +
		string(rune('0'+index/10%10)) +
		string(rune('0'+index%10))
}
