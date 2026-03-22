package kafka

// Kafka topic constants for the Intake-Onboarding Service.
// Follows naming convention: {service}.{domain} (spec section 6.0).

const (
	// TopicPatientLifecycle carries PATIENT_CREATED, PATIENT_ENROLLED events.
	// Consumer: KB-20.
	TopicPatientLifecycle = "intake.patient-lifecycle"

	// TopicSlotEvents carries slot fill events with safety results.
	// Consumers: KB-20, KB-22.
	TopicSlotEvents = "intake.slot-events"

	// TopicSafetyAlerts carries HARD_STOP triggers (urgent physician card).
	// Consumers: KB-23, Notifications.
	TopicSafetyAlerts = "intake.safety-alerts"

	// TopicSafetyFlags carries SOFT_FLAG triggers (pharmacist awareness).
	// Consumer: Review Queue.
	TopicSafetyFlags = "intake.safety-flags"

	// TopicCompletions carries intake-complete events ready for pharmacist review.
	// Consumer: Review Queue.
	TopicCompletions = "intake.completions"

	// TopicCheckinEvents carries biweekly check-in and trajectory signals.
	// Consumers: M4, KB-20, KB-21.
	TopicCheckinEvents = "intake.checkin-events"

	// TopicSessionLifecycle carries ABANDONED, PAUSED session events.
	// Consumer: Admin Dashboard.
	TopicSessionLifecycle = "intake.session-lifecycle"

	// TopicLabOrders carries missing baseline lab requests.
	// Consumer: Lab Integration.
	TopicLabOrders = "intake.lab-orders"
)

// AllTopics returns all 8 intake Kafka topics.
func AllTopics() []string {
	return []string{
		TopicPatientLifecycle,
		TopicSlotEvents,
		TopicSafetyAlerts,
		TopicSafetyFlags,
		TopicCompletions,
		TopicCheckinEvents,
		TopicSessionLifecycle,
		TopicLabOrders,
	}
}
