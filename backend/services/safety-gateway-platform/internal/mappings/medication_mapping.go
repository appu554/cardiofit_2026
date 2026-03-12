package mappings

// MedicationMapping provides UUID to medication name mapping
type MedicationMapping struct {
	uuidToName map[string]string
	nameToUUID map[string]string
}

// NewMedicationMapping creates a new medication mapping instance
func NewMedicationMapping() *MedicationMapping {
	mapping := &MedicationMapping{
		uuidToName: make(map[string]string),
		nameToUUID: make(map[string]string),
	}
	
	// Initialize with test mappings
	mapping.initializeTestMappings()
	
	return mapping
}

// initializeTestMappings sets up test medication mappings
func (m *MedicationMapping) initializeTestMappings() {
	medications := map[string]string{
		"550e8400-e29b-41d4-a716-446655440001": "warfarin",
		"550e8400-e29b-41d4-a716-446655440002": "aspirin",
		"550e8400-e29b-41d4-a716-446655440003": "ibuprofen",
		"550e8400-e29b-41d4-a716-446655440004": "penicillin",
		"550e8400-e29b-41d4-a716-446655440005": "amoxicillin",
		"550e8400-e29b-41d4-a716-446655440006": "metformin",
		"550e8400-e29b-41d4-a716-446655440007": "lisinopril",
		"550e8400-e29b-41d4-a716-446655440008": "metoprolol",
		"550e8400-e29b-41d4-a716-446655440009": "atorvastatin",
		"550e8400-e29b-41d4-a716-446655440010": "acetaminophen",
	}
	
	for uuid, name := range medications {
		m.uuidToName[uuid] = name
		m.nameToUUID[name] = uuid
	}
}

// UUIDToName converts a medication UUID to its name
func (m *MedicationMapping) UUIDToName(uuid string) (string, bool) {
	name, exists := m.uuidToName[uuid]
	return name, exists
}

// NameToUUID converts a medication name to its UUID
func (m *MedicationMapping) NameToUUID(name string) (string, bool) {
	uuid, exists := m.nameToUUID[name]
	return uuid, exists
}

// UUIDsToNames converts a slice of UUIDs to medication names
func (m *MedicationMapping) UUIDsToNames(uuids []string) []string {
	names := make([]string, 0, len(uuids))
	for _, uuid := range uuids {
		if name, exists := m.uuidToName[uuid]; exists {
			names = append(names, name)
		} else {
			// If no mapping exists, use the UUID as-is (fallback)
			names = append(names, uuid)
		}
	}
	return names
}

// NamesToUUIDs converts a slice of medication names to UUIDs
func (m *MedicationMapping) NamesToUUIDs(names []string) []string {
	uuids := make([]string, 0, len(names))
	for _, name := range names {
		if uuid, exists := m.nameToUUID[name]; exists {
			uuids = append(uuids, uuid)
		} else {
			// If no mapping exists, use the name as-is (fallback)
			uuids = append(uuids, name)
		}
	}
	return uuids
}
