package slots

import (
	"testing"
)

func TestSlotTable_TotalCount(t *testing.T) {
	table := AllSlots()
	if len(table) != 50 {
		t.Errorf("expected 50 slots, got %d", len(table))
	}
}

func TestSlotTable_DomainCount(t *testing.T) {
	domains := make(map[string]int)
	for _, s := range AllSlots() {
		domains[s.Domain]++
	}
	if len(domains) != 8 {
		t.Errorf("expected 8 domains, got %d", len(domains))
	}
	// Verify all expected domains exist
	expected := []string{
		"demographics", "glycemic", "renal", "cardiac",
		"lipid", "medications", "lifestyle", "symptoms",
	}
	for _, d := range expected {
		if _, ok := domains[d]; !ok {
			t.Errorf("missing domain: %s", d)
		}
	}
}

func TestSlotTable_RequiredSlots(t *testing.T) {
	required := 0
	for _, s := range AllSlots() {
		if s.Required {
			required++
		}
	}
	// At least demographics + key glycemic + key renal + key cardiac should be required
	if required < 15 {
		t.Errorf("expected at least 15 required slots, got %d", required)
	}
}

func TestSlotTable_LOINCUniqueness(t *testing.T) {
	seen := make(map[string]string)
	for _, s := range AllSlots() {
		if s.LOINCCode == "" {
			continue // some slots (like free-text) may not have LOINC
		}
		if existing, ok := seen[s.LOINCCode]; ok {
			t.Errorf("duplicate LOINC code %s: %s and %s", s.LOINCCode, existing, s.Name)
		}
		seen[s.LOINCCode] = s.Name
	}
}

func TestSlotTable_DataTypes(t *testing.T) {
	validTypes := map[DataType]bool{
		DataTypeNumeric:     true,
		DataTypeBoolean:     true,
		DataTypeCodedChoice: true,
		DataTypeText:        true,
		DataTypeDate:        true,
		DataTypeInteger:     true,
		DataTypeList:        true,
	}
	for _, s := range AllSlots() {
		if !validTypes[s.DataType] {
			t.Errorf("slot %s has invalid data type: %s", s.Name, s.DataType)
		}
	}
}

func TestSlotTable_LookupByName(t *testing.T) {
	s, ok := LookupSlot("fbg")
	if !ok {
		t.Fatal("expected to find slot 'fbg'")
	}
	if s.Domain != "glycemic" {
		t.Errorf("expected fbg domain 'glycemic', got %s", s.Domain)
	}
	if s.LOINCCode != "1558-6" {
		t.Errorf("expected fbg LOINC '1558-6', got %s", s.LOINCCode)
	}
}

func TestSlotTable_LookupByName_NotFound(t *testing.T) {
	_, ok := LookupSlot("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent slot")
	}
}
