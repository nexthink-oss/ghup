package local

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetLocalFileContent(t *testing.T) {
	testFilePath := filepath.Join("testdata", "testfile.txt")
	testFileContent := []byte("test content\n")
	tests := []struct {
		name        string
		arg         string
		separator   string
		wantTarget  string
		wantContent []byte
		wantErr     bool
	}{
		{
			name:        "Single file",
			arg:         testFilePath,
			separator:   ":",
			wantTarget:  testFilePath,
			wantContent: testFileContent,
		},
		{
			name:        "Source and target",
			arg:         strings.Join([]string{testFilePath, "destfile.txt"}, ":"),
			separator:   ":",
			wantTarget:  "destfile.txt",
			wantContent: testFileContent,
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
			name:        "Alternate separator",
			arg:         strings.Join([]string{testFilePath, "destfile.txt"}, "=>"),
			separator:   "=>",
			wantTarget:  "destfile.txt",
			wantContent: testFileContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, gotContent, err := GetLocalFileContent(tt.arg, tt.separator)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLocalFileContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("GetLocalFileContent() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
			if !bytes.Equal(gotContent, tt.wantContent) {
				t.Errorf("GetLocalFileContent() gotContent = %v, want %v", gotContent, tt.wantContent)
			}
		})
	}
}
