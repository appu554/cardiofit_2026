package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/cardiofit/notification-service/internal/models"
	"go.uber.org/zap"
)

// SendGridClient manages SendGrid API interactions for email
type SendGridClient struct {
	apiKey     string
	fromEmail  string
	httpClient *http.Client
	logger     *zap.Logger
	baseURL    string
}

// SendGridEmailRequest represents SendGrid API v3 email request
type SendGridEmailRequest struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmailAddress      `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []SendGridContent         `json:"content"`
}

// SendGridPersonalization represents email recipients
type SendGridPersonalization struct {
	To      []SendGridEmailAddress `json:"to"`
	Subject string                 `json:"subject,omitempty"`
}

// SendGridEmailAddress represents an email address
type SendGridEmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// SendGridContent represents email content
type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// SendGridResponse represents SendGrid API response
type SendGridResponse struct {
	MessageID string            `json:"message_id,omitempty"`
	Errors    []SendGridError   `json:"errors,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// SendGridError represents an error from SendGrid
type SendGridError struct {
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Help    string `json:"help,omitempty"`
}

// NewSendGridClient creates a new SendGrid client
func NewSendGridClient(apiKey, fromEmail string, logger *zap.Logger) *SendGridClient {
	return &SendGridClient{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:  logger,
		baseURL: "https://api.sendgrid.com/v3",
	}
}

// SendEmail sends an HTML email via SendGrid
func (s *SendGridClient) SendEmail(ctx context.Context, to, subject, htmlBody string) (messageID string, err error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("sendgrid API key not configured")
	}

	if to == "" {
		return "", fmt.Errorf("recipient email is required")
	}

	if subject == "" {
		return "", fmt.Errorf("email subject is required")
	}

	if htmlBody == "" {
		return "", fmt.Errorf("email body is required")
	}

	// Build email request
	emailRequest := SendGridEmailRequest{
		Personalizations: []SendGridPersonalization{
			{
				To: []SendGridEmailAddress{
					{Email: to},
				},
			},
		},
		From: SendGridEmailAddress{
			Email: s.fromEmail,
			Name:  "CardioFit Clinical Alerts",
		},
		Subject: subject,
		Content: []SendGridContent{
			{
				Type:  "text/html",
				Value: htmlBody,
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(emailRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal email request: %w", err)
	}

	// Build request URL
	apiURL := fmt.Sprintf("%s/mail/send", s.baseURL)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	// Execute request
	s.logger.Debug("Sending email via SendGrid",
		zap.String("to", to),
		zap.String("subject", subject),
	)

	startTime := time.Now()
	resp, err := s.httpClient.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		s.logger.Error("SendGrid API request failed",
			zap.Error(err),
			zap.String("to", to),
			zap.Duration("latency", latency),
		)
		return "", fmt.Errorf("sendgrid API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.Error("SendGrid API returned error status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("to", to),
		)
		return "", fmt.Errorf("sendgrid API returned status %d", resp.StatusCode)
	}

	// Extract message ID from X-Message-Id header
	msgID := resp.Header.Get("X-Message-Id")
	if msgID == "" {
		// Fallback to generated ID
		msgID = fmt.Sprintf("SG%d", time.Now().Unix())
	}

	s.logger.Info("Email sent successfully via SendGrid",
		zap.String("message_id", msgID),
		zap.String("to", to),
		zap.Duration("latency", latency),
		zap.Int("status_code", resp.StatusCode),
	)

	return msgID, nil
}

// BuildAlertEmailHTML builds HTML email body for clinical alerts
func (s *SendGridClient) BuildAlertEmailHTML(alert *models.Alert, user *models.User) string {
	tmplText := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 8px 8px 0 0;
        }
        .alert-badge {
            display: inline-block;
            padding: 5px 15px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: bold;
            text-transform: uppercase;
        }
        .critical { background-color: #dc3545; }
        .high { background-color: #fd7e14; }
        .moderate { background-color: #ffc107; color: #000; }
        .low { background-color: #28a745; }
        .content {
            background: #fff;
            padding: 30px;
            border: 1px solid #dee2e6;
            border-top: none;
        }
        .patient-info {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 8px;
            margin: 20px 0;
        }
        .vital-signs {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 15px;
            margin: 20px 0;
        }
        .vital-item {
            background: #e9ecef;
            padding: 10px;
            border-radius: 5px;
        }
        .vital-label {
            font-size: 12px;
            color: #6c757d;
            text-transform: uppercase;
        }
        .vital-value {
            font-size: 24px;
            font-weight: bold;
            color: #212529;
        }
        .recommendations {
            background: #d1ecf1;
            border-left: 4px solid #0c5460;
            padding: 15px;
            margin: 20px 0;
        }
        .recommendations h3 {
            margin-top: 0;
            color: #0c5460;
        }
        .recommendations ul {
            margin: 10px 0;
            padding-left: 20px;
        }
        .action-button {
            display: inline-block;
            background: #667eea;
            color: white;
            padding: 12px 30px;
            text-decoration: none;
            border-radius: 5px;
            margin: 20px 0;
            font-weight: bold;
        }
        .footer {
            text-align: center;
            padding: 20px;
            color: #6c757d;
            font-size: 12px;
            border-top: 1px solid #dee2e6;
        }
        .timestamp {
            color: #6c757d;
            font-size: 14px;
            margin-top: 10px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1 style="margin: 0;">Clinical Alert</h1>
        <div style="margin-top: 10px;">
            <span class="alert-badge {{.SeverityClass}}">{{.Severity}}</span>
            <span style="margin-left: 10px;">{{.AlertType}}</span>
        </div>
    </div>

    <div class="content">
        <div class="patient-info">
            <h2 style="margin-top: 0;">Patient Information</h2>
            <p><strong>Patient ID:</strong> {{.PatientID}}</p>
            <p><strong>Location:</strong> {{.Location}}</p>
            <p><strong>Hospital:</strong> {{.HospitalID}}</p>
            <p><strong>Department:</strong> {{.DepartmentID}}</p>
        </div>

        <h2>Alert Details</h2>
        <p><strong>Message:</strong> {{.Message}}</p>
        {{if .RiskScore}}
        <p><strong>Risk Score:</strong> {{.RiskScore}}%</p>
        {{end}}
        {{if .Confidence}}
        <p><strong>Confidence:</strong> {{.Confidence}}%</p>
        {{end}}

        {{if .VitalSigns}}
        <h3>Current Vital Signs</h3>
        <div class="vital-signs">
            <div class="vital-item">
                <div class="vital-label">Heart Rate</div>
                <div class="vital-value">{{.VitalSigns.HeartRate}} <span style="font-size: 14px;">bpm</span></div>
            </div>
            <div class="vital-item">
                <div class="vital-label">Blood Pressure</div>
                <div class="vital-value">{{.VitalSigns.BloodPressureSystolic}}/{{.VitalSigns.BloodPressureDiastolic}}</div>
            </div>
            <div class="vital-item">
                <div class="vital-label">Temperature</div>
                <div class="vital-value">{{.VitalSigns.Temperature}} <span style="font-size: 14px;">°F</span></div>
            </div>
            {{if .VitalSigns.OxygenSaturation}}
            <div class="vital-item">
                <div class="vital-label">SpO2</div>
                <div class="vital-value">{{.VitalSigns.OxygenSaturation}} <span style="font-size: 14px;">%</span></div>
            </div>
            {{end}}
        </div>
        {{end}}

        {{if .Recommendations}}
        <div class="recommendations">
            <h3>Recommended Actions</h3>
            <ul>
            {{range .Recommendations}}
                <li>{{.}}</li>
            {{end}}
            </ul>
        </div>
        {{end}}

        <a href="{{.DeepLink}}" class="action-button">View Patient Details</a>

        <div class="timestamp">
            <p><strong>Alert Time:</strong> {{.FormattedTimestamp}}</p>
            <p><strong>Alert ID:</strong> {{.AlertID}}</p>
        </div>
    </div>

    <div class="footer">
        <p>This is an automated clinical alert from CardioFit Clinical Synthesis Hub.</p>
        <p>Please do not reply to this email. For support, contact your system administrator.</p>
        <p>&copy; 2025 CardioFit. All rights reserved.</p>
    </div>
</body>
</html>
`

	// Prepare template data
	data := map[string]interface{}{
		"AlertID":            alert.AlertID,
		"PatientID":          alert.PatientID,
		"HospitalID":         alert.HospitalID,
		"DepartmentID":       alert.DepartmentID,
		"AlertType":          alert.AlertType,
		"Severity":           alert.Severity,
		"SeverityClass":      s.getSeverityClass(alert.Severity),
		"Message":            alert.Message,
		"RiskScore":          alert.RiskScore,
		"Confidence":         alert.Confidence * 100, // Convert to percentage
		"Location":           fmt.Sprintf("Room %s, Bed %s", alert.PatientLocation.Room, alert.PatientLocation.Bed),
		"VitalSigns":         alert.VitalSigns,
		"Recommendations":    alert.Recommendations,
		"DeepLink":           fmt.Sprintf("https://cardiofit.app/patient/%s/alert/%s", alert.PatientID, alert.AlertID),
		"FormattedTimestamp": time.Unix(alert.Timestamp, 0).Format("2006-01-02 15:04:05 MST"),
	}

	// Parse and execute template
	tmpl, err := template.New("email").Parse(tmplText)
	if err != nil {
		s.logger.Error("Failed to parse email template", zap.Error(err))
		return s.buildFallbackEmail(alert)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		s.logger.Error("Failed to execute email template", zap.Error(err))
		return s.buildFallbackEmail(alert)
	}

	return buf.String()
}

// getSeverityClass returns CSS class for severity badge
func (s *SendGridClient) getSeverityClass(severity models.AlertSeverity) string {
	switch severity {
	case models.SeverityCritical:
		return "critical"
	case models.SeverityHigh:
		return "high"
	case models.SeverityModerate:
		return "moderate"
	case models.SeverityLow:
		return "low"
	default:
		return "moderate"
	}
}

// buildFallbackEmail builds a simple text-based email as fallback
func (s *SendGridClient) buildFallbackEmail(alert *models.Alert) string {
	return fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; padding: 20px;">
	<h2 style="color: #dc3545;">Clinical Alert: %s</h2>
	<p><strong>Patient ID:</strong> %s</p>
	<p><strong>Severity:</strong> %s</p>
	<p><strong>Message:</strong> %s</p>
	<p><strong>Location:</strong> Room %s, Bed %s</p>
	<p><strong>Time:</strong> %s</p>
	<a href="https://cardiofit.app/patient/%s/alert/%s" style="display: inline-block; padding: 10px 20px; background: #667eea; color: white; text-decoration: none; border-radius: 5px; margin-top: 20px;">View Details</a>
</body>
</html>
	`,
		alert.AlertType,
		alert.PatientID,
		alert.Severity,
		alert.Message,
		alert.PatientLocation.Room,
		alert.PatientLocation.Bed,
		time.Unix(alert.Timestamp, 0).Format("2006-01-02 15:04:05"),
		alert.PatientID,
		alert.AlertID,
	)
}

// ValidateAPIKey validates SendGrid API key by making a test request
func (s *SendGridClient) ValidateAPIKey(ctx context.Context) error {
	if s.apiKey == "" {
		return fmt.Errorf("sendgrid API key not configured")
	}

	// Test endpoint
	apiURL := fmt.Sprintf("%s/scopes", s.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("invalid sendgrid API key")
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid validation returned status %d", resp.StatusCode)
	}

	s.logger.Info("SendGrid API key validated successfully")
	return nil
}

// Close cleans up resources
func (s *SendGridClient) Close() error {
	// Close HTTP client connections
	s.httpClient.CloseIdleConnections()
	return nil
}
