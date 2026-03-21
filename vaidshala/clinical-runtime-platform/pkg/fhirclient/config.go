package fhirclient

import "fmt"

// GoogleFHIRConfig holds connection details for Google Healthcare FHIR Store.
type GoogleFHIRConfig struct {
	Enabled         bool   `json:"enabled"`
	ProjectID       string `json:"project_id"`
	Location        string `json:"location"`
	DatasetID       string `json:"dataset_id"`
	FhirStoreID     string `json:"fhir_store_id"`
	CredentialsPath string `json:"credentials_path"`
}

// BaseURL returns the FHIR Store REST base URL.
func (c GoogleFHIRConfig) BaseURL() string {
	return fmt.Sprintf(
		"https://healthcare.googleapis.com/v1/projects/%s/locations/%s/datasets/%s/fhirStores/%s/fhir",
		c.ProjectID, c.Location, c.DatasetID, c.FhirStoreID,
	)
}
