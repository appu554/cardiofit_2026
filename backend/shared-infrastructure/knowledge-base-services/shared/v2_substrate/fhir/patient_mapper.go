package fhir

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// ResidentToAUPatient translates a v2 substrate Resident to a JSON-shape
// AU FHIR Patient resource (HL7 AU Base v6.0.0). Returns a generic
// map[string]interface{} representing the FHIR JSON structure, suitable
// for json.Marshal to produce wire format.
//
// AU-specific extensions used:
//   - identifier[].system http://ns.electronichealth.net.au/id/hi/ihi/1.0
//   - extension[indigenous-status]
//   - extension[care-intensity] (Vaidshala-internal)
//
// Lossy fields (NOT round-tripped):
//   - SDMs       — DROPPED on egress to AU FHIR Patient. AU Patient profile
//                  has no clean Vaidshala-internal-aware home for SDM
//                  references; preserved only in canonical kb-20 storage.
//                  Adapters that need SDM round-trip must read from kb-20
//                  directly.
//   - CreatedAt / UpdatedAt — managed by canonical store, not in FHIR
//   - FacilityID — DROPPED on egress. Vaidshala internal facility UUIDs do
//                  not map cleanly to FHIR Organization references without
//                  an Organization resource lookup. Preserved only in
//                  canonical kb-20 storage.
func ResidentToAUPatient(r models.Resident) (map[string]interface{}, error) {
	if err := validation.ValidateResident(r); err != nil {
		return nil, err
	}
	p := map[string]interface{}{
		"resourceType": "Patient",
		"id":           r.ID.String(),
		"name": []map[string]interface{}{
			{"use": "official", "given": []string{r.GivenName}, "family": r.FamilyName},
		},
		"gender": r.Sex,
		// FHIR birthDate is a timezone-naive civil date; normalise to UTC for stable formatting.
		"birthDate": r.DOB.UTC().Format("2006-01-02"),
		"active":    r.Status == models.ResidentStatusActive,
	}

	// Identifier (only if IHI present)
	if r.IHI != "" {
		p["identifier"] = []map[string]interface{}{
			{
				"system": SystemIHI,
				"value":  r.IHI,
			},
		}
	}

	// Extensions
	exts := []map[string]interface{}{}
	if r.IndigenousStatus != "" {
		exts = append(exts, map[string]interface{}{
			"url":         ExtIndigenousStatus,
			"valueString": r.IndigenousStatus,
		})
	}
	if r.CareIntensity != "" {
		exts = append(exts, map[string]interface{}{
			"url":         ExtCareIntensity,
			"valueString": r.CareIntensity,
		})
	}
	if r.AdmissionDate != nil {
		exts = append(exts, map[string]interface{}{
			"url":           ExtAdmissionDate,
			"valueDateTime": r.AdmissionDate.Format(time.RFC3339),
		})
	}
	if len(exts) > 0 {
		p["extension"] = exts
	}

	return p, nil
}

// AUPatientToResident is the reverse mapper. Lossy fields documented
// above are NOT reconstructed; callers must set them separately.
func AUPatientToResident(p map[string]interface{}) (models.Resident, error) {
	r := models.Resident{}

	if rt, _ := p["resourceType"].(string); rt != "Patient" {
		return r, fmt.Errorf("expected resourceType=Patient, got %q", rt)
	}

	if id, _ := p["id"].(string); id != "" {
		// optional — if absent or non-UUID (external system identifier),
		// caller must assign.
		if parsed, err := uuid.Parse(id); err == nil {
			r.ID = parsed
		}
	}

	// Name
	if names, ok := p["name"].([]interface{}); ok && len(names) > 0 {
		if first, ok := names[0].(map[string]interface{}); ok {
			if givens, ok := first["given"].([]interface{}); ok && len(givens) > 0 {
				r.GivenName, _ = givens[0].(string)
			}
			r.FamilyName, _ = first["family"].(string)
		}
	}

	r.Sex, _ = p["gender"].(string)
	if dob, _ := p["birthDate"].(string); dob != "" {
		// Go's time.Parse defaults to UTC for date-only formats, so the
		// parsed DOB is already in UTC — matches the egress normalisation.
		if t, err := time.Parse("2006-01-02", dob); err == nil {
			r.DOB = t
		}
	}

	if active, _ := p["active"].(bool); active {
		r.Status = models.ResidentStatusActive
	} else {
		// FHIR active=false maps to Discharged by design. This is intentional
		// one-way lossiness: Vaidshala's Deceased and Transferred lifecycle
		// states are domain-specific and not modelled by the FHIR active flag.
		// Round-tripping those states requires reading from the canonical
		// kb-20 store, not from FHIR.
		r.Status = models.ResidentStatusDischarged
	}

	// Identifier → IHI
	if ids, ok := p["identifier"].([]interface{}); ok {
		for _, idAny := range ids {
			id, ok := idAny.(map[string]interface{})
			if !ok {
				continue
			}
			if id["system"] == SystemIHI {
				r.IHI, _ = id["value"].(string)
			}
		}
	}

	// Extensions
	if exts, ok := p["extension"].([]interface{}); ok {
		for _, extAny := range exts {
			ext, ok := extAny.(map[string]interface{})
			if !ok {
				continue
			}
			url, _ := ext["url"].(string)
			switch url {
			case ExtIndigenousStatus:
				r.IndigenousStatus, _ = ext["valueString"].(string)
			case ExtCareIntensity:
				r.CareIntensity, _ = ext["valueString"].(string)
			case ExtAdmissionDate:
				if s, _ := ext["valueDateTime"].(string); s != "" {
					if t, err := time.Parse(time.RFC3339, s); err == nil {
						r.AdmissionDate = &t
					}
				}
			}
		}
	}

	// Defence-in-depth: validate the reconstructed Resident before returning
	// to callers. External Layer 1B adapters may send malformed FHIR; we want
	// to reject early rather than poison kb-20 storage.
	if err := validation.ValidateResident(r); err != nil {
		return r, fmt.Errorf("ingress validation: %w", err)
	}
	return r, nil
}
