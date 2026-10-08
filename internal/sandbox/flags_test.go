package sandbox_test

import (
	"encoding/json"
	"sige/internal/sandbox"
	"testing"
)

func TestParseFlagsString(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"   ", nil},
		{"-O3", []string{"-O3"}},
		{"-O3 -std=c11 -Wall", []string{"-O3", "-std=c11", "-Wall"}},
		{"  -O2    -Wall  ", []string{"-O2", "-Wall"}},
		{"-DMSG=\"hello world\" -O3", []string{"-DMSG=hello world", "-O3"}},
		{"-I'my/dir with spaces' -Wall", []string{"-Imy/dir with spaces", "-Wall"}},
	}

	for _, tc := range tests {
		got := sandbox.ParseFlagsString(tc.input)
		if len(got) != len(tc.expected) {
			t.Fatalf("ParseFlagsString(%q): len %d != len %d (got %v, expected %v)", tc.input, len(got), len(tc.expected), got, tc.expected)
		}
		for i := range got {
			if got[i] != tc.expected[i] {
				t.Errorf("ParseFlagsString(%q)[%d]: got %q, expected %q", tc.input, i, got[i], tc.expected[i])
			}
		}
	}
}

func TestCompileFlags_UnmarshalJSON(t *testing.T) {
	// 1. Array of strings
	var flags1 sandbox.CompileFlags
	if err := json.Unmarshal([]byte(`["-O3", "-Wall", "-std=c11"]`), &flags1); err != nil {
		t.Fatalf("failed unmarshaling array: %v", err)
	}
	if len(flags1) != 3 || flags1[0] != "-O3" || flags1[1] != "-Wall" || flags1[2] != "-std=c11" {
		t.Errorf("unexpected array result: %v", flags1)
	}

	// 2. String representation
	var flags2 sandbox.CompileFlags
	if err := json.Unmarshal([]byte(`"-O3 -Wall -std=c11"`), &flags2); err != nil {
		t.Fatalf("failed unmarshaling string: %v", err)
	}
	if len(flags2) != 3 || flags2[0] != "-O3" || flags2[1] != "-Wall" || flags2[2] != "-std=c11" {
		t.Errorf("unexpected string result: %v", flags2)
	}

	// 3. Null
	var flags3 sandbox.CompileFlags
	if err := json.Unmarshal([]byte(`null`), &flags3); err != nil {
		t.Fatalf("failed unmarshaling null: %v", err)
	}
	if flags3 != nil {
		t.Errorf("expected nil for null JSON, got: %v", flags3)
	}

	// 4. Invalid type (number)
	var flags4 sandbox.CompileFlags
	if err := json.Unmarshal([]byte(`12345`), &flags4); err == nil {
		t.Fatal("expected error unmarshaling number to CompileFlags, got nil")
	}
}

func TestValidateCompileFlags(t *testing.T) {
	tests := []struct {
		name        string
		language    string
		flags       []string
		expectError bool
	}{
		{
			name:        "Valid C flags",
			language:    "c",
			flags:       []string{"-O3", "-std=c11", "-Wall", "-DDEBUG", "-lm", "-lpthread"},
			expectError: false,
		},
		{
			name:        "Valid C++ flags",
			language:    "cpp",
			flags:       []string{"-std=c++17", "-O2", "-Wextra"},
			expectError: false,
		},
		{
			name:        "Empty flags slice",
			language:    "c",
			flags:       []string{},
			expectError: false,
		},
		{
			name:        "Flags rejected for Python",
			language:    "python",
			flags:       []string{"-O3"},
			expectError: true,
		},
		{
			name:        "Flags rejected for Bash",
			language:    "bash",
			flags:       []string{"-Wall"},
			expectError: true,
		},
		{
			name:        "Output flag -o rejected",
			language:    "c",
			flags:       []string{"-o", "custom_bin"},
			expectError: true,
		},
		{
			name:        "Output flag -ofile rejected",
			language:    "c",
			flags:       []string{"-ocustom_bin"},
			expectError: true,
		},
		{
			name:        "Output flag --output rejected",
			language:    "c",
			flags:       []string{"--output=custom_bin"},
			expectError: true,
		},
		{
			name:        "Compile only flag -c rejected",
			language:    "c",
			flags:       []string{"-c"},
			expectError: true,
		},
		{
			name:        "Assemble only flag -S rejected",
			language:    "c",
			flags:       []string{"-S"},
			expectError: true,
		},
		{
			name:        "Preprocess only flag -E rejected",
			language:    "c",
			flags:       []string{"-E"},
			expectError: true,
		},
		{
			name:        "Control char rejected",
			language:    "c",
			flags:       []string{"-O2\n-evil"},
			expectError: true,
		},
		{
			name:        "Optimization flags -O, -O0, -O2, -O3, -Os, -Ofast allowed",
			language:    "c",
			flags:       []string{"-O0", "-O1", "-O2", "-O3", "-Os", "-Og", "-Ofast"},
			expectError: false,
		},
		{
			name:        "Response file @/etc/passwd rejected",
			language:    "c",
			flags:       []string{"@/etc/passwd"},
			expectError: true,
		},
		{
			name:        "Positional non-flag argument rejected",
			language:    "c",
			flags:       []string{"/etc/passwd"},
			expectError: true,
		},
		{
			name:        "Double dash -- rejected",
			language:    "c",
			flags:       []string{"--"},
			expectError: true,
		},
		{
			name:        "GCC plugin flag -fplugin rejected",
			language:    "c",
			flags:       []string{"-fplugin=/tmp/evil.so"},
			expectError: true,
		},
		{
			name:        "Custom specs flag -specs rejected",
			language:    "c",
			flags:       []string{"-specs=/tmp/evil.spec"},
			expectError: true,
		},
		{
			name:        "Wrapper flag -wrapper rejected",
			language:    "c",
			flags:       []string{"-wrapper", "/bin/sh"},
			expectError: true,
		},
		{
			name:        "Forced include flag -include rejected",
			language:    "c",
			flags:       []string{"-include=/etc/passwd"},
			expectError: true,
		},
		{
			name:        "Dependency file output -MF rejected",
			language:    "c",
			flags:       []string{"-MF=/workspace/solution"},
			expectError: true,
		},
		{
			name:        "Linker output override -Wl,-o rejected",
			language:    "c",
			flags:       []string{"-Wl,-o,/tmp/evil"},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := sandbox.ValidateCompileFlags(tc.language, tc.flags)
			if tc.expectError && err == nil {
				t.Errorf("[%s] expected validation error, got nil", tc.name)
			}
			if !tc.expectError && err != nil {
				t.Errorf("[%s] expected no error, got: %v", tc.name, err)
			}
		})
	}
}
