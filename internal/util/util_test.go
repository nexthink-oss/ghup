package util

import (
	"slices"
	"testing"

	"github.com/spf13/viper"
)

func TestCoalesce(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "No arguments",
			input:    []string{},
			expected: "",
		},
		{
			name:     "All empty strings",
			input:    []string{"", "", ""},
			expected: "",
		},
		{
			name:     "Mixed empty and non-empty strings",
			input:    []string{"", "first", "", "second"},
			expected: "first",
		},
		{
			name:     "All non-empty strings",
			input:    []string{"first", "second", "third"},
			expected: "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Coalesce(tt.input...)
			if result != tt.expected {
				t.Errorf("Coalesce(%v) = %v; expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEncodeYAML(t *testing.T) {
	tests := []struct {
		name           string
		input          any
		expectedOutput string
	}{
		{
			name:           "Empty struct",
			input:          struct{}{},
			expectedOutput: "{}\n",
		},
		{
			name: "Simple struct",
			input: struct {
				Name  string `yaml:"name"`
				Value int    `yaml:"value"`
			}{
				Name:  "test",
				Value: 42,
			},
			expectedOutput: "name: test\nvalue: 42\n",
		},
		{
			name: "Nested struct",
			input: struct {
				Name  string `yaml:"name"`
				Inner struct {
					Field1 string `yaml:"field1"`
					Field2 int    `yaml:"field2"`
				} `yaml:"inner"`
			}{
				Name: "outer",
				Inner: struct {
					Field1 string `yaml:"field1"`
					Field2 int    `yaml:"field2"`
				}{
					Field1: "inner value",
					Field2: 99,
				},
			},
			expectedOutput: "name: outer\ninner:\n  field1: inner value\n  field2: 99\n",
		},
		{
			name: "Map input",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			expectedOutput: "key1: value1\nkey2: 123\n",
		},
		{
			name:           "Slice input",
			input:          []string{"item1", "item2", "item3"},
			expectedOutput: "- item1\n- item2\n- item3\n",
		},
		{
			name: "Nested struct with expected indentation",
			input: struct {
				Level1 struct {
					Level2 struct {
						Level3 string `yaml:"level3"`
					} `yaml:"level2"`
				} `yaml:"level1"`
			}{
				Level1: struct {
					Level2 struct {
						Level3 string `yaml:"level3"`
					} `yaml:"level2"`
				}{
					Level2: struct {
						Level3 string `yaml:"level3"`
					}{
						Level3: "deep value",
					},
				},
			},
			expectedOutput: "level1:\n  level2:\n    level3: deep value\n",
		},
		{
			name: "Nested list elements",
			input: struct {
				Level1 struct {
					List []struct {
						Item1 string `yaml:"item1"`
						Item2 int    `yaml:"item2"`
					} `yaml:"list"`
				} `yaml:"level1"`
			}{
				Level1: struct {
					List []struct {
						Item1 string `yaml:"item1"`
						Item2 int    `yaml:"item2"`
					} `yaml:"list"`
				}{
					List: []struct {
						Item1 string `yaml:"item1"`
						Item2 int    `yaml:"item2"`
					}{
						{Item1: "value1", Item2: 1},
						{Item1: "value2", Item2: 2},
					},
				},
			},
			expectedOutput: "level1:\n  list:\n    - item1: value1\n      item2: 1\n    - item1: value2\n      item2: 2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeYAML(tt.input)
			if result != tt.expectedOutput {
				t.Errorf("EncodeYAML() = %v; expected %v", result, tt.expectedOutput)
			}
		})
	}
}

func TestIsCommitHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid short hash",
			input:    "1a2b3c4",
			expected: true,
		},
		{
			name:     "Valid long hash",
			input:    "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b",
			expected: true,
		},
		{
			name:     "Invalid hash with non-hex characters",
			input:    "1a2b3c4z",
			expected: false,
		},
		{
			name:     "Invalid hash too short",
			input:    "1a2b3",
			expected: false,
		},
		{
			name:     "Invalid hash too long",
			input:    "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCommitHash(tt.input)
			if result != tt.expected {
				t.Errorf("IsCommitHash(%v) = %v; expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeRefName(t *testing.T) {
	tests := []struct {
		name           string
		refName        string
		defaultPrefix  string
		expectedOutput string
		expectedError  bool
	}{
		{
			name:           "Fully-qualified ref",
			refName:        "refs/heads/main",
			defaultPrefix:  "heads",
			expectedOutput: "heads/main",
		},
		{
			name:           "Fully-qualified ref",
			refName:        "refs/tags/v1.0.0",
			defaultPrefix:  "tags",
			expectedOutput: "tags/v1.0.0",
		},
		{
			name:           "Partially-qualified ref",
			refName:        "heads/main",
			defaultPrefix:  "heads",
			expectedOutput: "heads/main",
		},
		{
			name:           "Partially-qualified ref",
			refName:        "tags/v1.0.0",
			defaultPrefix:  "tags",
			expectedOutput: "tags/v1.0.0",
		},
		{
			name:           "Unqualified ref with heads default #1",
			refName:        "main",
			defaultPrefix:  "heads",
			expectedOutput: "heads/main",
		},
		{
			name:           "Unqualified ref with heads default #2",
			refName:        "feature/branch",
			defaultPrefix:  "heads",
			expectedOutput: "heads/feature/branch",
		},

		{
			name:           "Unqualified ref with tags default",
			refName:        "v1.0.0",
			defaultPrefix:  "tags",
			expectedOutput: "tags/v1.0.0",
		},
		{
			name:           "Partially-qualified ref with mismatched default prefix",
			refName:        "heads/main",
			defaultPrefix:  "tags",
			expectedOutput: "heads/main",
		},
		{
			name:           "Invalid ref name with control characters",
			refName:        "invalid\x7Fref",
			defaultPrefix:  "heads",
			expectedOutput: "",
			expectedError:  true,
		},
		{
			name:           "Invalid ref name with consecutive dots",
			refName:        "invalid..ref",
			defaultPrefix:  "heads",
			expectedOutput: "",
			expectedError:  true,
		},
		{
			name:           "Invalid ref name with trailing dot",
			refName:        "invalidref.",
			defaultPrefix:  "heads",
			expectedOutput: "",
			expectedError:  true,
		},
		{
			name:           "Invalid ref name with leading slash",
			refName:        "/invalidref",
			defaultPrefix:  "heads",
			expectedOutput: "",
			expectedError:  true,
		},
		{
			name:           "Invalid ref name with trailing slash",
			refName:        "invalidref/",
			defaultPrefix:  "heads",
			expectedOutput: "",
			expectedError:  true,
		},
		{
			name:          "Invalid ref name with sequence @{",
			refName:       "invalid@{ref",
			defaultPrefix: "heads",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeRefName(tt.refName, tt.defaultPrefix)
			if (err != nil) != tt.expectedError {
				t.Errorf("NormalizeRefName() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if result != tt.expectedOutput {
				t.Errorf("NormalizeRefName() = %v, expected %v", result, tt.expectedOutput)
			}
		})
	}
}

func TestBuildCommitMessage(t *testing.T) {
	tests := []struct {
		name           string
		viperSettings  map[string]interface{}
		expectedOutput string
	}{
		{
			name:           "No message and no trailers",
			viperSettings:  map[string]interface{}{},
			expectedOutput: "",
		},
		{
			name: "Only message",
			viperSettings: map[string]interface{}{
				"message": "This is a commit message",
			},
			expectedOutput: "This is a commit message",
		},
		{
			name: "Message with long title",
			viperSettings: map[string]interface{}{
				"message": "This is a very long commit message title that exceeds seventy-two characters and should trigger a warning",
			},
			expectedOutput: "This is a very long commit message title that exceeds seventy-two characters and should trigger a warning",
		},
		{
			name: "Message with trailers",
			viperSettings: map[string]interface{}{
				"message":        "This is a commit message",
				"author.trailer": "Co-Authored-By",
				"user.name":      "John Doe",
				"user.email":     "john.doe@example.com",
			},
			expectedOutput: "This is a commit message\n\nCo-Authored-By: John Doe <john.doe@example.com>",
		},
		{
			name: "Message with author trailer disabled",
			viperSettings: map[string]interface{}{
				"message":        "This is a commit message",
				"author.trailer": "",
				"user.name":      "John Doe",
				"user.email":     "john.doe@example.com",
			},
			expectedOutput: "This is a commit message",
		},
		{
			name: "Only trailers",
			viperSettings: map[string]interface{}{
				"author.trailer": "Co-Authored-By",
				"user.name":      "John Doe",
				"user.email":     "john.doe@example.com",
			},
			expectedOutput: "\nCo-Authored-By: John Doe <john.doe@example.com>",
		},
		{
			name: "Multiple trailers",
			viperSettings: map[string]interface{}{
				"message":        "This is a commit message",
				"author.trailer": "Co-Authored-By",
				"user.name":      "John Doe",
				"user.email":     "john.doe@example.com",
				"trailer": map[string]string{
					"Reviewed-By": "Jane Smith",
				},
			},
			expectedOutput: "This is a commit message\n\nCo-Authored-By: John Doe <john.doe@example.com>\nReviewed-By: Jane Smith",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.viperSettings {
				viper.Set(key, value)
			}

			result := BuildCommitMessage()
			if result != tt.expectedOutput {
				t.Errorf("BuildCommitMessage() = %v; expected %v", result, tt.expectedOutput)
			}

			viper.Reset()
		})
	}
}

func TestBuildTrailers(t *testing.T) {
	tests := []struct {
		name           string
		viperSettings  map[string]interface{}
		expectedOutput []string
	}{
		{
			name:           "No trailers",
			viperSettings:  map[string]interface{}{},
			expectedOutput: []string{},
		},
		{
			name: "Author trailer only",
			viperSettings: map[string]interface{}{
				"author.trailer": "Co-Authored-By",
				"user.name":      "John Doe",
				"user.email":     "john.doe@example.com",
			},
			expectedOutput: []string{"Co-Authored-By: John Doe <john.doe@example.com>"},
		},
		{
			name: "Author trailer disabled",
			viperSettings: map[string]interface{}{
				"author.trailer": "",
				"user.name":      "John Doe",
				"user.email":     "john.doe@example.com",
			},
			expectedOutput: []string{},
		},
		{
			name: "Multiple trailers",
			viperSettings: map[string]interface{}{
				"author.trailer": "Co-Authored-By",
				"user.name":      "John Doe",
				"user.email":     "john.doe@example.com",
				"trailer": map[string]string{
					"Reviewed-By": "Jane Smith",
				},
			},
			expectedOutput: []string{
				"Co-Authored-By: John Doe <john.doe@example.com>",
				"Reviewed-By: Jane Smith",
			},
		},
		{
			name: "Only additional trailers",
			viperSettings: map[string]interface{}{
				"trailer": map[string]string{
					"Reviewed-By":   "Jane Smith",
					"Signed-Off-By": "Alice Johnson",
				},
			},
			expectedOutput: []string{
				"Reviewed-By: Jane Smith",
				"Signed-Off-By: Alice Johnson",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.viperSettings {
				viper.Set(key, value)
			}

			result := BuildTrailers()
			if !slices.Equal[[]string](result, tt.expectedOutput) {
				t.Errorf("BuildTrailers() = %v; expected %v", result, tt.expectedOutput)
			}

			viper.Reset()
		})
	}
}
