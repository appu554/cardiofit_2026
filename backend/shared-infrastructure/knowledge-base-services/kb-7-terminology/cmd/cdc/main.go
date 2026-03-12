// CDC Pipeline Runner for KB-7 Terminology Service
//
// This command runs the complete CDC pipeline:
//   - GraphDB Producer: Detects changes in GraphDB, publishes to Kafka
//   - Neo4j Consumer: Reads from Kafka, syncs to Neo4j
//
// Usage:
//   go run ./cmd/cdc --mode=both           # Run both producer and consumer
//   go run ./cmd/cdc --mode=producer       # Run only GraphDB producer
//   go run ./cmd/cdc --mode=consumer       # Run only Neo4j consumer
//
// Environment Variables:
//   KAFKA_BROKERS       - Kafka broker addresses (default: localhost:9092)
//   KAFKA_TOPIC         - CDC topic name (default: kb7.graphdb.changes)
//   GRAPHDB_URL         - GraphDB URL (default: http://localhost:7200)
//   GRAPHDB_REPOSITORY  - GraphDB repository (default: kb7-terminology)
//   NEO4J_URL           - Neo4j bolt URL (default: bolt://localhost:7687)
//   NEO4J_USERNAME      - Neo4j username (default: neo4j)
//   NEO4J_PASSWORD      - Neo4j password (required)
//   NEO4J_DATABASE      - Neo4j database (default: kb7-au)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kb-7-terminology/internal/cdc"

	"github.com/sirupsen/logrus"
)

func main() {
	// Parse flags
	mode := flag.String("mode", "both", "Run mode: producer, consumer, or both")
	pollInterval := flag.Duration("poll-interval", 5*time.Second, "Producer poll interval")
	batchSize := flag.Int("batch-size", 100, "Batch size for processing")
	workerCount := flag.Int("workers", 4, "Number of consumer workers")
	flag.Parse()

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	logger.WithFields(logrus.Fields{
		"mode":          *mode,
		"poll_interval": *pollInterval,
		"batch_size":    *batchSize,
		"workers":       *workerCount,
	}).Info("Starting KB-7 CDC Pipeline")

	// Load configuration from environment
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "kb7.graphdb.changes")
	graphDBURL := getEnv("GRAPHDB_URL", "http://localhost:7200")
	graphDBRepo := getEnv("GRAPHDB_REPOSITORY", "kb7-terminology")
	graphDBUser := getEnv("GRAPHDB_USERNAME", "")
	graphDBPass := getEnv("GRAPHDB_PASSWORD", "")
	neo4jURL := getEnv("NEO4J_URL", "bolt://localhost:7687")
	neo4jUser := getEnv("NEO4J_USERNAME", "neo4j")
	neo4jPass := getEnv("NEO4J_PASSWORD", "")
	neo4jDB := getEnv("NEO4J_DATABASE", "kb7-au")

	// Validate required config
	if neo4jPass == "" && (*mode == "consumer" || *mode == "both") {
		logger.Fatal("NEO4J_PASSWORD environment variable is required")
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Error channel for goroutines
	errChan := make(chan error, 2)

	brokers := strings.Split(kafkaBrokers, ",")

	// Start producer if requested
	var producer *cdc.GraphDBProducer
	if *mode == "producer" || *mode == "both" {
		producerConfig := &cdc.ProducerConfig{
			GraphDBURL:        graphDBURL,
			GraphDBRepository: graphDBRepo,
			GraphDBUsername:   graphDBUser,
			GraphDBPassword:   graphDBPass,
			KafkaBrokers:      brokers,
			Topic:             kafkaTopic,
			PollInterval:      *pollInterval,
			BatchSize:         *batchSize,
		}

		var err error
		producer, err = cdc.NewGraphDBProducer(producerConfig, logger)
		if err != nil {
			logger.WithError(err).Fatal("Failed to create GraphDB producer")
		}

		go func() {
			logger.Info("Starting GraphDB CDC producer...")
			if err := producer.Start(ctx); err != nil {
				errChan <- fmt.Errorf("producer error: %w", err)
			}
		}()

		logger.WithFields(logrus.Fields{
			"graphdb_url": graphDBURL,
			"repository":  graphDBRepo,
			"topic":       kafkaTopic,
		}).Info("GraphDB CDC producer started")
	}

	// Start consumer if requested
	var consumer *cdc.Neo4jConsumer
	if *mode == "consumer" || *mode == "both" {
		consumerConfig := &cdc.ConsumerConfig{
			KafkaBrokers:   brokers,
			Topic:          kafkaTopic,
			GroupID:        "kb7-neo4j-sync",
			Neo4jURL:       neo4jURL,
			Neo4jUsername:  neo4jUser,
			Neo4jPassword:  neo4jPass,
			Neo4jDatabase:  neo4jDB,
			BatchSize:      *batchSize,
			BatchTimeout:   5 * time.Second,
			MaxRetries:     3,
			RetryBackoff:   time.Second,
			CommitInterval: 10 * time.Second,
			WorkerCount:    *workerCount,
		}

		var err error
		consumer, err = cdc.NewNeo4jConsumer(consumerConfig, logger)
		if err != nil {
			logger.WithError(err).Fatal("Failed to create Neo4j consumer")
		}

		go func() {
			logger.Info("Starting Neo4j CDC consumer...")
			if err := consumer.Start(ctx); err != nil {
				errChan <- fmt.Errorf("consumer error: %w", err)
			}
		}()

		logger.WithFields(logrus.Fields{
			"neo4j_url": neo4jURL,
			"database":  neo4jDB,
			"topic":     kafkaTopic,
			"group_id":  "kb7-neo4j-sync",
		}).Info("Neo4j CDC consumer started")
	}

	// Start stats reporter
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if producer != nil {
					stats := producer.Stats()
					logger.WithFields(logrus.Fields{
						"component":         "producer",
						"changes_detected":  stats.ChangesDetected,
						"messages_produced": stats.MessagesProduced,
						"errors":            stats.Errors,
					}).Info("CDC producer stats")
				}
				if consumer != nil {
					stats := consumer.Stats()
					logger.WithFields(logrus.Fields{
						"component":          "consumer",
						"messages_received":  stats.MessagesReceived,
						"messages_processed": stats.MessagesProcessed,
						"messages_failed":    stats.MessagesFailed,
						"batches_committed":  stats.BatchesCommitted,
					}).Info("CDC consumer stats")
				}
			}
		}
	}()

	// Print startup banner
	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Println("    🔄 KB-7 CDC PIPELINE RUNNING")
	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Printf("    Mode: %s\n", *mode)
	fmt.Printf("    Topic: %s\n", kafkaTopic)
	if producer != nil {
		fmt.Printf("    Producer: GraphDB (%s/%s) → Kafka\n", graphDBURL, graphDBRepo)
	}
	if consumer != nil {
		fmt.Printf("    Consumer: Kafka → Neo4j (%s/%s)\n", neo4jURL, neo4jDB)
	}
	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Println("    Press Ctrl+C to stop")
	fmt.Println("═══════════════════════════════════════════════════════════════════")

	// Wait for shutdown or error
	select {
	case sig := <-sigChan:
		logger.WithField("signal", sig).Info("Received shutdown signal")
	case err := <-errChan:
		logger.WithError(err).Error("CDC component error")
	}

	// Graceful shutdown
	logger.Info("Shutting down CDC pipeline...")
	cancel()

	if producer != nil {
		if err := producer.Stop(); err != nil {
			logger.WithError(err).Error("Error stopping producer")
		}
	}
	if consumer != nil {
		if err := consumer.Stop(); err != nil {
			logger.WithError(err).Error("Error stopping consumer")
		}
	}

	logger.Info("CDC pipeline stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
