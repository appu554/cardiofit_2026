package config

// SourceConfig describes a registered data source with its expected format and freshness SLA.
type SourceConfig struct {
	SourceID       string `json:"source_id"`
	SourceType     string `json:"source_type"` // EHR, LAB, DEVICE, WEARABLE, ABDM, PATIENT_REPORTED
	Name           string `json:"name"`
	Format         string `json:"format"`          // FHIR, HL7V2, CSV, JSON, OPENMHEALTH
	FreshnessHours int    `json:"freshness_hours"` // Alert if no data received within this window
	Enabled        bool   `json:"enabled"`
}

// DefaultSourceRegistry returns the built-in source configurations.
// In production, these can be overridden from PostgreSQL or environment variables.
func DefaultSourceRegistry() []SourceConfig {
	return []SourceConfig{
		{SourceID: "fhir_direct", SourceType: "EHR", Name: "Direct FHIR", Format: "FHIR", FreshnessHours: 24, Enabled: true},
		{SourceID: "hl7v2_mllp", SourceType: "EHR", Name: "HL7v2 MLLP", Format: "HL7V2", FreshnessHours: 24, Enabled: true},
		{SourceID: "ehr_fhir_rest", SourceType: "EHR", Name: "EHR FHIR REST", Format: "FHIR", FreshnessHours: 24, Enabled: true},
		{SourceID: "thyrocare", SourceType: "LAB", Name: "Thyrocare", Format: "CSV", FreshnessHours: 48, Enabled: true},
		{SourceID: "redcliffe", SourceType: "LAB", Name: "Redcliffe", Format: "JSON", FreshnessHours: 48, Enabled: true},
		{SourceID: "srl_agilus", SourceType: "LAB", Name: "SRL Agilus", Format: "CSV", FreshnessHours: 48, Enabled: true},
		{SourceID: "dr_lal", SourceType: "LAB", Name: "Dr. Lal Pathlabs", Format: "CSV", FreshnessHours: 48, Enabled: true},
		{SourceID: "metropolis", SourceType: "LAB", Name: "Metropolis", Format: "JSON", FreshnessHours: 48, Enabled: true},
		{SourceID: "orange_health", SourceType: "LAB", Name: "Orange Health", Format: "JSON", FreshnessHours: 48, Enabled: true},
		{SourceID: "ble_device", SourceType: "DEVICE", Name: "BLE Device (App Relay)", Format: "JSON", FreshnessHours: 4, Enabled: true},
		{SourceID: "health_connect", SourceType: "WEARABLE", Name: "Health Connect", Format: "OPENMHEALTH", FreshnessHours: 12, Enabled: true},
		{SourceID: "ultrahuman", SourceType: "WEARABLE", Name: "Ultrahuman CGM", Format: "JSON", FreshnessHours: 4, Enabled: true},
		{SourceID: "apple_health", SourceType: "WEARABLE", Name: "Apple Health", Format: "OPENMHEALTH", FreshnessHours: 12, Enabled: true},
		{SourceID: "abdm_hiu", SourceType: "ABDM", Name: "ABDM HIU", Format: "FHIR", FreshnessHours: 168, Enabled: true},
		{SourceID: "app_checkin", SourceType: "PATIENT_REPORTED", Name: "Flutter App Checkin", Format: "JSON", FreshnessHours: 24, Enabled: true},
		{SourceID: "whatsapp", SourceType: "PATIENT_REPORTED", Name: "WhatsApp NLU", Format: "JSON", FreshnessHours: 24, Enabled: true},
	}
}

// LookupSource finds a source config by ID. Returns false if not found.
func LookupSource(registry []SourceConfig, sourceID string) (SourceConfig, bool) {
	for _, s := range registry {
		if s.SourceID == sourceID {
			return s, true
		}
	}
	return SourceConfig{}, false
}
