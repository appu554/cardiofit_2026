package services

import "fmt"

type MRIMessageType string

const (
	MRIMessageImproved MRIMessageType = "IMPROVED"
	MRIMessageStable   MRIMessageType = "STABLE"
	MRIMessageWorsened MRIMessageType = "WORSENED"
	MRIMessageModerate MRIMessageType = "CROSSED_MODERATE"
	MRIMessageHigh     MRIMessageType = "CROSSED_HIGH"
)

type MRIPatientMessage struct {
	Type      MRIMessageType `json:"type"`
	MessageEN string         `json:"message_en"`
	MessageHI string         `json:"message_hi"`
	Tone      string         `json:"tone"`
	Score     float64        `json:"score"`
	Delta     float64        `json:"delta"`
	TopDriver string         `json:"top_driver,omitempty"`
}

// GenerateMRIMessage produces a patient-facing message based on MRI score change.
// Spec §6.2, Table 7 — 5 message templates.
func GenerateMRIMessage(currentScore, previousScore float64, currentCategory, topDriver string) *MRIPatientMessage {
	delta := currentScore - previousScore

	// HIGH_DETERIORATION: no patient message, physician-gated only
	if currentCategory == "HIGH_DETERIORATION" {
		return &MRIPatientMessage{
			Type:  MRIMessageHigh,
			Tone:  "N/A",
			Score: currentScore,
			Delta: delta,
		}
	}

	// Crossed into MODERATE (≥51)
	if currentCategory == "MODERATE_DETERIORATION" && previousScore <= 50 {
		return &MRIPatientMessage{
			Type:      MRIMessageModerate,
			MessageEN: "Your metabolic health needs attention. We are sharing this update with your doctor for review. In the meantime, resuming your walking routine is the most important step.",
			MessageHI: "आपके चयापचय स्वास्थ्य पर ध्यान देने की जरूरत है। हम इस अपडेट को आपके डॉक्टर के साथ साझा कर रहे हैं। इस बीच, अपनी वॉकिंग रूटीन फिर से शुरू करना सबसे महत्वपूर्ण कदम है।",
			Tone:      "Vaidya",
			Score:     currentScore,
			Delta:     delta,
		}
	}

	// Improved ≥5 points
	if delta <= -5 {
		return &MRIPatientMessage{
			Type:      MRIMessageImproved,
			MessageEN: fmt.Sprintf("Great news! Your metabolic health score improved from %.0f to %.0f this month. Your walking and diet changes are making a real difference. Keep it up!", previousScore, currentScore),
			MessageHI: fmt.Sprintf("बढ़िया खबर! आपका चयापचय स्वास्थ्य स्कोर इस महीने %.0f से %.0f हो गया। आपकी वॉकिंग और डाइट में बदलाव वास्तव में फर्क ला रहे हैं। ऐसे ही जारी रखें!", previousScore, currentScore),
			Tone:      "Sathi",
			Score:     currentScore,
			Delta:     delta,
		}
	}

	// Worsened ≥5 points
	if delta >= 5 {
		return &MRIPatientMessage{
			Type:      MRIMessageWorsened,
			MessageEN: fmt.Sprintf("Your metabolic health score has increased from %.0f to %.0f. The main reason is reduced activity this week. Would you like to try shorter 5-minute walks instead?", previousScore, currentScore),
			MessageHI: fmt.Sprintf("आपका चयापचय स्वास्थ्य स्कोर %.0f से %.0f हो गया है। इस सप्ताह कम गतिविधि मुख्य कारण है। क्या आप इसके बजाय छोटी 5-मिनट की वॉक करना चाहेंगे?", previousScore, currentScore),
			Tone:      "Kinship",
			Score:     currentScore,
			Delta:     delta,
			TopDriver: topDriver,
		}
	}

	// Stable (±2 points)
	return &MRIPatientMessage{
		Type:      MRIMessageStable,
		MessageEN: fmt.Sprintf("Your metabolic health score is holding steady at %.0f. You're maintaining your improvements. This week, can you add one more post-meal walk per day?", currentScore),
		MessageHI: fmt.Sprintf("आपका चयापचय स्वास्थ्य स्कोर %.0f पर स्थिर है। आप अपने सुधारों को बनाए रख रहे हैं। इस सप्ताह, क्या आप प्रति दिन एक और भोजन के बाद की वॉक जोड़ सकते हैं?", currentScore),
		Tone:      "Kinship",
		Score:     currentScore,
		Delta:     delta,
	}
}
