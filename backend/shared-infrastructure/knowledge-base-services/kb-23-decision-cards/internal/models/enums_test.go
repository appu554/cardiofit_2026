package models

import "testing"

func TestMCUGate_Level(t *testing.T) {
	tests := []struct {
		gate MCUGate
		want int
	}{
		{GateSafe, 0},
		{GateModify, 1},
		{GatePause, 2},
		{GateHalt, 3},
	}
	for _, tt := range tests {
		t.Run(string(tt.gate), func(t *testing.T) {
			got := tt.gate.Level()
			if got != tt.want {
				t.Errorf("%q.Level() = %d, want %d", tt.gate, got, tt.want)
			}
		})
	}
}

func TestMostRestrictive(t *testing.T) {
	tests := []struct {
		name string
		a, b MCUGate
		want MCUGate
	}{
		{"SAFE vs HALT", GateSafe, GateHalt, GateHalt},
		{"PAUSE vs MODIFY", GatePause, GateModify, GatePause},
		{"SAFE vs SAFE", GateSafe, GateSafe, GateSafe},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MostRestrictive(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("MostRestrictive(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
