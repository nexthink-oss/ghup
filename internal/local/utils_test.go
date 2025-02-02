package local

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitUpdateSpec(t *testing.T) {
	testFilePath := filepath.Join("testdata", "testfile.txt")
	tests := []struct {
		name       string
		arg        string
		separator  string
		wantSource string
		wantTarget string
		wantErr    bool
	}{
		{
			name:       "Single file",
			arg:        testFilePath,
			separator:  ":",
			wantSource: testFilePath,
			wantTarget: testFilePath,
		},
		{
			name:       "Source and target",
			arg:        strings.Join([]string{testFilePath, "destfile.txt"}, ":"),
			separator:  ":",
			wantSource: testFilePath,
			wantTarget: "destfile.txt",
		},
		{
			name:      "Empty parameter",
			arg:       "",
			separator: ":",
			wantErr:   true,
		},
		{
			name:      "Missing source",
			arg:       ":destfile.txt",
			separator: ":",
			wantErr:   true,
		},
		{
			name:      "Missing target",
			arg:       strings.Join([]string{testFilePath, ""}, ":"),
			separator: ":",
			wantErr:   true,
		},
		{
			name:       "Alternate separator",
			arg:        strings.Join([]string{testFilePath, "destfile.txt"}, "=>"),
			separator:  "=>",
			wantSource: testFilePath,
			wantTarget: "destfile.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSource, gotTarget, err := ParseUpdateSpec(tt.arg, tt.separator)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitUpdateSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotSource != tt.wantSource {
				t.Errorf("SplitUpdateSpec() gotSource = %v, want %v", gotSource, tt.wantSource)
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("SplitUpdateSpec() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
		})
	}
}
