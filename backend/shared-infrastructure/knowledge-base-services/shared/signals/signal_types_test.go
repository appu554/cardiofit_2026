package signals

import "testing"

func TestSignalTypeConstants(t *testing.T) {
	tests := []struct {
		signal   SignalType
		expected string
	}{
		{SignalFBG, "FBG"},
		{SignalPPBG, "PPBG"},
		{SignalHbA1c, "HBA1C"},
		{SignalSBP, "SBP"},
		{SignalDBP, "DBP"},
		{SignalWeight, "WEIGHT"},
		{SignalAdherence, "ADHERENCE"},
	}
	for _, tt := range tests {
		if string(tt.signal) != tt.expected {
			t.Errorf("signal %s != %s", tt.signal, tt.expected)
		}
	}
}

func TestSignalSource(t *testing.T) {
	if string(SourceAppManual) != "APP_MANUAL" {
		t.Error("SourceAppManual mismatch")
	}
	if string(SourceBLEDevice) != "BLE_DEVICE" {
		t.Error("SourceBLEDevice mismatch")
	}
}
