package mcpserver

import (
	"reflect"
	"strings"
	"testing"
)

func TestSanitizeRgArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   []string
		want    []string
		wantErr string
	}{
		{
			name:  "allows safe flags",
			input: []string{" -i ", "--glob", " *.kt ", "--max-count=5", "-F"},
			want:  []string{"-i", "--glob", "*.kt", "--max-count=5", "-F"},
		},
		{
			name:    "rejects output flag",
			input:   []string{"--heading"},
			wantErr: "unsupported rg arg",
		},
		{
			name:    "rejects positional arg",
			input:   []string{"src/main"},
			wantErr: "unsupported rg arg",
		},
		{
			name:    "rejects missing value",
			input:   []string{"--glob"},
			wantErr: "requires a value",
		},
		{
			name:    "rejects flag as value",
			input:   []string{"-m", "--json"},
			wantErr: "requires a non-flag value",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := sanitizeRgArgs(testCase.input)
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("expected error containing %q, got %v", testCase.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("sanitizeRgArgs error: %v", err)
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("unexpected args:\nwant: %#v\n got: %#v", testCase.want, got)
			}
		})
	}
}
