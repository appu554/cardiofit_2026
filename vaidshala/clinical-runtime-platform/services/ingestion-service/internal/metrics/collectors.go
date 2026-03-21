package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Ingestion Prometheus metrics -- 10 metrics from spec section 7.3.

var (
	// MessagesReceived counts total messages received by source type.
	MessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_messages_received_total",
			Help: "Total messages received by the ingestion service",
		},
		[]string{"source_type", "source_id", "tenant_id"},
	)

	// MessagesProcessed counts messages processed by stage and status.
	MessagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_messages_processed_total",
			Help: "Total messages processed by pipeline stage and status",
		},
		[]string{"source_type", "stage", "status"},
	)

	// PipelineDuration tracks the duration of each pipeline stage.
	PipelineDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ingestion_pipeline_duration_seconds",
			Help:    "Duration of each pipeline stage in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source_type", "stage"},
	)

	// CriticalValues counts critical values detected.
	CriticalValues = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_critical_values_total",
			Help: "Total critical values detected during validation",
		},
		[]string{"observation_type", "tenant_id"},
	)

	// DLQMessages counts messages sent to the DLQ.
	DLQMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_dlq_messages_total",
			Help: "Total messages sent to the dead letter queue",
		},
		[]string{"error_class", "source_type"},
	)

	// WALMessagesPending tracks messages waiting in the write-ahead log.
	WALMessagesPending = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ingestion_wal_messages_pending",
			Help: "Number of messages pending in the Kafka WAL failover buffer",
		},
	)

	// PatientResolutionPending tracks unresolved patient identifiers.
	PatientResolutionPending = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ingestion_patient_resolution_pending",
			Help: "Number of observations pending patient resolution",
		},
		[]string{"tenant_id"},
	)

	// ABDMConsentOperations counts ABDM consent operations.
	ABDMConsentOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_abdm_consent_operations_total",
			Help: "Total ABDM consent operations by type and status",
		},
		[]string{"operation", "status"},
	)

	// FHIRValidationFailures counts FHIR validation failures.
	FHIRValidationFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_fhir_validation_failures_total",
			Help: "Total FHIR validation failures by profile and violation type",
		},
		[]string{"profile", "violation_type"},
	)

	// SourceFreshness tracks the freshness of data from each source.
	SourceFreshness = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ingestion_source_freshness_seconds",
			Help: "Seconds since last message from each data source",
		},
		[]string{"source_type", "source_id"},
	)
)
