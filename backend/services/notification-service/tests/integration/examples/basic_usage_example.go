package examples

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cardiofit/notification-service/internal/delivery"
	"github.com/cardiofit/notification-service/internal/fatigue"
	"github.com/cardiofit/notification-service/internal/models"
	"github.com/cardiofit/notification-service/internal/routing"
	"github.com/cardiofit/notification-service/internal/users"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// ExampleBasicNotification demonstrates how to send a simple notification
func ExampleBasicNotification() {
	ctx := context.Background()

	// 1. Connect to PostgreSQL
	dbURL := "postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// 2. Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer redisClient.Close()

	// 3. Initialize services
	userService := users.NewUserPreferenceService(pool, redisClient)
	fatigueTracker := fatigue.NewAlertFatigueTracker(redisClient)

	// 4. Configure delivery service (with real credentials)
	deliveryConfig := &delivery.Config{
		TwilioAccountSID: "your_twilio_account_sid",
		TwilioAuthToken:  "your_twilio_auth_token",
		TwilioFromNumber: "+1234567890",
		SendGridAPIKey:   "your_sendgrid_api_key",
		SendGridFromEmail: "notifications@cardiofit.com",
		FirebaseCredentialsPath: "path/to/firebase-credentials.json",
	}

	deliveryService, err := delivery.NewDeliveryService(deliveryConfig)
	if err != nil {
		log.Fatalf("Failed to create delivery service: %v", err)
	}

	// 5. Create alert router
	alertRouter := routing.NewAlertRouter(userService, fatigueTracker, deliveryService)

	// 6. Create a critical alert
	alert := &models.Alert{
		ID:          "alert_001",
		PatientID:   "patient_12345",
		Severity:    "CRITICAL",
		Type:        "CARDIAC_ARREST",
		Title:       "Code Blue - Cardiac Arrest",
		Message:     "Patient experiencing cardiac arrest. Immediate response required.",
		Timestamp:   time.Now(),
		TargetRoles: []string{"attending_physician", "charge_nurse"},
		Metadata: map[string]interface{}{
			"room_number":  "ICU-204",
			"patient_name": "John Doe",
			"vital_signs": map[string]interface{}{
				"heart_rate":       0,
				"blood_pressure":   "0/0",
				"oxygen_saturation": 0,
			},
		},
	}

	// 7. Route the alert through the notification system
	err = alertRouter.RouteAlert(ctx, alert)
	if err != nil {
		log.Fatalf("Failed to route alert: %v", err)
	}

	fmt.Println("✅ Alert routed successfully")
}

// ExampleSetupUserPreferences demonstrates how to configure user notification preferences
func ExampleSetupUserPreferences() {
	ctx := context.Background()

	// Connect to database
	dbURL := "postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Connect to Redis for caching
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer redisClient.Close()

	// Create user service
	userService := users.NewUserPreferenceService(pool, redisClient)

	// Configure preferences for an attending physician
	prefs := &models.UserPreferences{
		UserID: "dr_smith_001",

		// Channel preferences - which channels this user wants to receive notifications on
		ChannelPreferences: map[string]bool{
			"SMS":   true,  // SMS enabled
			"EMAIL": true,  // Email enabled
			"PUSH":  true,  // Push notifications enabled
			"PAGER": true,  // Pager enabled
		},

		// Severity-specific channel routing
		SeverityChannels: map[string][]string{
			"CRITICAL": {"PAGER", "SMS"},           // Critical alerts go to pager and SMS
			"HIGH":     {"SMS", "PUSH"},            // High alerts go to SMS and push
			"MEDIUM":   {"PUSH", "EMAIL"},          // Medium alerts go to push and email
			"LOW":      {"EMAIL"},                  // Low alerts only go to email
		},

		// Quiet hours configuration
		QuietHoursEnabled: true,
		QuietHoursStart:   22, // 10 PM
		QuietHoursEnd:     6,  // 6 AM
		// During quiet hours, only CRITICAL alerts will be sent via pager/SMS

		// Alert fatigue prevention
		MaxAlertsPerHour: 20, // Maximum 20 non-critical alerts per hour

		// Contact information
		PhoneNumber:  "+1234567890",
		Email:        "dr.smith@cardiofit.com",
		PagerNumber:  "+1234567891",
		FCMToken:     "firebase_device_token_here",
	}

	// Save preferences
	err = userService.SavePreferences(ctx, prefs)
	if err != nil {
		log.Fatalf("Failed to save preferences: %v", err)
	}

	fmt.Println("✅ User preferences saved successfully")

	// Retrieve preferences later
	retrievedPrefs, err := userService.GetPreferences(ctx, "dr_smith_001")
	if err != nil {
		log.Fatalf("Failed to retrieve preferences: %v", err)
	}

	fmt.Printf("✅ Retrieved preferences for user: %s\n", retrievedPrefs.UserID)
	fmt.Printf("   Quiet hours enabled: %v\n", retrievedPrefs.QuietHoursEnabled)
	fmt.Printf("   Max alerts per hour: %d\n", retrievedPrefs.MaxAlertsPerHour)
}

// ExampleSendDirectNotification demonstrates sending a notification directly to a specific user
func ExampleSendDirectNotification() {
	ctx := context.Background()

	// Configure delivery service
	deliveryConfig := &delivery.Config{
		TwilioAccountSID: "your_twilio_account_sid",
		TwilioAuthToken:  "your_twilio_auth_token",
		TwilioFromNumber: "+1234567890",
		SendGridAPIKey:   "your_sendgrid_api_key",
		SendGridFromEmail: "notifications@cardiofit.com",
	}

	deliveryService, err := delivery.NewDeliveryService(deliveryConfig)
	if err != nil {
		log.Fatalf("Failed to create delivery service: %v", err)
	}

	// Create notification request
	notificationReq := &models.NotificationRequest{
		NotificationID: "notif_001",
		UserID:         "dr_smith_001",
		Channel:        "SMS",
		Priority:       "HIGH",
		Subject:        "Lab Result Alert",
		Message:        "Patient John Doe has abnormal potassium level: 5.8 mmol/L. Please review.",
		Recipient:      "+1234567890",
		Metadata: map[string]interface{}{
			"patient_id":  "patient_12345",
			"lab_test":    "Potassium",
			"result":      "5.8 mmol/L",
			"normal_range": "3.5-5.0 mmol/L",
		},
	}

	// Send SMS notification
	err = deliveryService.SendSMS(ctx, notificationReq)
	if err != nil {
		log.Fatalf("Failed to send SMS: %v", err)
	}

	fmt.Println("✅ SMS notification sent successfully")

	// Check delivery status
	time.Sleep(2 * time.Second) // Wait for delivery
	status, err := deliveryService.GetDeliveryStatus(ctx, "notif_001")
	if err != nil {
		log.Fatalf("Failed to get delivery status: %v", err)
	}

	fmt.Printf("✅ Notification status: %s\n", status.Status)
	fmt.Printf("   Delivery timestamp: %v\n", status.Timestamp)
}

// ExampleCheckAlertFatigue demonstrates how to use the alert fatigue tracker
func ExampleCheckAlertFatigue() {
	ctx := context.Background()

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer redisClient.Close()

	// Create fatigue tracker
	tracker := fatigue.NewAlertFatigueTracker(redisClient)

	userID := "dr_smith_001"

	// Check if we should send an alert
	shouldSend, err := tracker.ShouldSendAlert(ctx, userID, "HIGH")
	if err != nil {
		log.Fatalf("Failed to check alert fatigue: %v", err)
	}

	if shouldSend {
		fmt.Println("✅ OK to send alert - user is not fatigued")

		// Record the alert
		err = tracker.RecordAlert(ctx, userID, "HIGH", "alert_123")
		if err != nil {
			log.Fatalf("Failed to record alert: %v", err)
		}

		fmt.Println("✅ Alert recorded in fatigue tracker")
	} else {
		fmt.Println("⚠️  Alert suppressed - user has exceeded alert limit")
	}

	// Get current alert count for user
	count, err := tracker.GetAlertCount(ctx, userID, 1*time.Hour)
	if err != nil {
		log.Fatalf("Failed to get alert count: %v", err)
	}

	fmt.Printf("📊 User has received %d alerts in the last hour\n", count)

	// Get recent alert history
	recentAlerts, err := tracker.GetRecentAlerts(ctx, userID, 10)
	if err != nil {
		log.Fatalf("Failed to get recent alerts: %v", err)
	}

	fmt.Printf("📜 Recent alert history (%d alerts):\n", len(recentAlerts))
	for i, alertID := range recentAlerts {
		fmt.Printf("   %d. %s\n", i+1, alertID)
	}
}

// ExampleGetUsersByRole demonstrates how to query users by their role
func ExampleGetUsersByRole() {
	ctx := context.Background()

	// Connect to database
	dbURL := "postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer redisClient.Close()

	// Create user service
	userService := users.NewUserPreferenceService(pool, redisClient)

	// Get all attending physicians for a department
	attendingPhysicians, err := userService.GetAttendingPhysician(ctx, "cardiology")
	if err != nil {
		log.Fatalf("Failed to get attending physicians: %v", err)
	}

	fmt.Printf("👨‍⚕️ Found %d attending physicians in cardiology:\n", len(attendingPhysicians))
	for _, user := range attendingPhysicians {
		fmt.Printf("   - %s (User ID: %s)\n", user.Email, user.UserID)
		fmt.Printf("     Phone: %s, Pager: %s\n", user.PhoneNumber, user.PagerNumber)
	}

	// Get charge nurse
	chargeNurses, err := userService.GetChargeNurse(ctx, "cardiology")
	if err != nil {
		log.Fatalf("Failed to get charge nurses: %v", err)
	}

	fmt.Printf("👩‍⚕️ Found %d charge nurses:\n", len(chargeNurses))
	for _, user := range chargeNurses {
		fmt.Printf("   - %s (User ID: %s)\n", user.Email, user.UserID)
	}

	// Get residents
	residents, err := userService.GetResident(ctx, "cardiology")
	if err != nil {
		log.Fatalf("Failed to get residents: %v", err)
	}

	fmt.Printf("🎓 Found %d residents:\n", len(residents))
	for _, user := range residents {
		fmt.Printf("   - %s (User ID: %s)\n", user.Email, user.UserID)
	}
}

// ExampleAlertWithEscalation demonstrates creating an alert with escalation policy
func ExampleAlertWithEscalation() {
	ctx := context.Background()

	// Setup services (abbreviated for clarity)
	dbURL := "postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"
	pool, _ := pgxpool.New(ctx, dbURL)
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer redisClient.Close()

	userService := users.NewUserPreferenceService(pool, redisClient)
	fatigueTracker := fatigue.NewAlertFatigueTracker(redisClient)

	deliveryConfig := &delivery.Config{
		TwilioAccountSID: "your_twilio_account_sid",
		TwilioAuthToken:  "your_twilio_auth_token",
		TwilioFromNumber: "+1234567890",
	}
	deliveryService, _ := delivery.NewDeliveryService(deliveryConfig)

	alertRouter := routing.NewAlertRouter(userService, fatigueTracker, deliveryService)

	// Create alert with escalation metadata
	alert := &models.Alert{
		ID:          "alert_escalation_001",
		PatientID:   "patient_12345",
		Severity:    "HIGH",
		Type:        "ABNORMAL_LAB_RESULT",
		Title:       "Critical Lab Result - Requires Acknowledgment",
		Message:     "Patient potassium level dangerously high: 6.2 mmol/L",
		Timestamp:   time.Now(),
		TargetRoles: []string{"attending_physician"},
		Metadata: map[string]interface{}{
			"escalation_policy": map[string]interface{}{
				"enabled":              true,
				"initial_timeout":      5 * 60,  // 5 minutes
				"escalation_levels": []map[string]interface{}{
					{
						"level":       1,
						"timeout":     5 * 60, // 5 minutes
						"roles":       []string{"attending_physician"},
					},
					{
						"level":       2,
						"timeout":     3 * 60, // 3 minutes
						"roles":       []string{"attending_physician", "charge_nurse"},
					},
					{
						"level":       3,
						"timeout":     2 * 60, // 2 minutes
						"roles":       []string{"attending_physician", "charge_nurse", "clinical_director"},
					},
				},
			},
			"requires_acknowledgment": true,
			"auto_escalate":           true,
		},
	}

	// Route the alert
	err := alertRouter.RouteAlert(ctx, alert)
	if err != nil {
		log.Fatalf("Failed to route alert with escalation: %v", err)
	}

	fmt.Println("✅ Alert with escalation policy created successfully")
	fmt.Println("   - Initial notification sent to attending physician")
	fmt.Println("   - Will escalate if not acknowledged within 5 minutes")
	fmt.Println("   - Escalation Level 1 → 2 → 3 until acknowledged")
}
