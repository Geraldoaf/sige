package sandbox

import (
	"sige/internal/constants"
	"testing"
)

func TestGetMemoryBytes(t *testing.T) {
	tests := []struct {
		mb       int64
		expected int64
	}{
		{0, 0},
		{1, constants.BytesInMB},
		{50, 50 * constants.BytesInMB},
	}

	for _, tc := range tests {
		cfg := Config{MemoryMB: tc.mb}
		got := cfg.GetMemoryBytes()
		if got != tc.expected {
			t.Errorf("GetMemoryBytes(%d) = %d; expected %d", tc.mb, got, tc.expected)
		}
	}
}

func TestGetFormattedCPU(t *testing.T) {
	tests := []struct {
		cpu      string
		expected string
	}{
		{"", ""},
		{"invalid", ""},
		{"50%", "50000 100000"},
		{"100%", "100000 100000"},
		{"10 100000", "10 100000"},
	}

	for _, tc := range tests {
		cfg := Config{CPU: tc.cpu}
		got := cfg.GetFormattedCPU()
		if got != tc.expected {
			t.Errorf("GetFormattedCPU(%q) = %q; expected %q", tc.cpu, got, tc.expected)
		}
	}
}
