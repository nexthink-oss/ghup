package util

import (
	"os"
	"slices"
	"testing"

	"github.com/spf13/viper"
)

func TestGithubActionsBranch(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedBranch string
	}{
		{
			name: "PR context",
			envVars: map[string]string{
				"GITHUB_REF_TYPE": "branch",
				"GITHUB_HEAD_REF": "feature-branch",
				"GITHUB_REF_NAME": "7/merge",
			},
			expectedBranch: "feature-branch",
		},
		{
			name: "Other context",
			envVars: map[string]string{
				"GITHUB_REF_TYPE": "branch",
				"GITHUB_HEAD_REF": "",
				"GITHUB_REF_NAME": "main",
			},
			expectedBranch: "main",
		},
		{
			name: "Not a branch",
			envVars: map[string]string{
				"GITHUB_REF_TYPE": "tag",
				"GITHUB_REF_NAME": "v1.0.0",
			},
			expectedBranch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.envVars {
				if err := os.Setenv(key, value); err != nil {
					t.Fatalf("os.Setenv(%v, %v) = %v", key, value, err)
				}
			}

			result := GithubActionsBranch()
			if result != tt.expectedBranch {
				t.Errorf("GithubActionsBranch() = %v; expected %v", result, tt.expectedBranch)
			}

			for key := range tt.envVars {
				if err := os.Unsetenv(key); err != nil {
					t.Fatalf("os.Unsetenv(%v) = %v", key, err)
				}
			}
		})
	}
}

func TestGithubActionsContext(t *testing.T) {
	tests := []struct {
		name            string
		envVars         map[string]string
		expectedContext *RepositoryContext
	}{
		{
			name: "Valid context",
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "owner/repo",
				"GITHUB_REF_TYPE":   "branch",
				"GITHUB_REF_NAME":   "main",
			},
			expectedContext: &RepositoryContext{
				Owner:  "owner",
				Name:   "repo",
				Branch: "main",
			},
		},
		{
			name: "PR context",
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "owner/repo",
				"GITHUB_REF_TYPE":   "branch",
				"GITHUB_HEAD_REF":   "feature-branch",
				"GITHUB_REF_NAME":   "1/merge",
			},
			expectedContext: &RepositoryContext{
				Owner:  "owner",
				Name:   "repo",
				Branch: "feature-branch",
			},
		},
		{
			name: "Tag context",
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "owner/repo",
				"GITHUB_REF_TYPE":   "tag",
				"GITHUB_HEAD_REF":   "",
				"GITHUB_REF_NAME":   "v1.0.0",
			},
			expectedContext: &RepositoryContext{
				Owner:  "owner",
				Name:   "repo",
				Branch: "",
			},
		},
		{
			name: "No repository",
			envVars: map[string]string{
				"GITHUB_REF_TYPE": "branch",
				"GITHUB_REF_NAME": "main",
			},
			expectedContext: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.envVars {
				if err := os.Setenv(key, value); err != nil {
					t.Fatalf("os.Setenv(%v, %v) = %v", key, value, err)
				}
			}

			result := GithubActionsContext()
			if (result == nil && tt.expectedContext != nil) || (result != nil && tt.expectedContext == nil) {
				t.Errorf("GithubActionsContext() = %v; expected %v", result, tt.expectedContext)
			} else if result != nil && tt.expectedContext != nil {
				if result.Owner != tt.expectedContext.Owner || result.Name != tt.expectedContext.Name || result.Branch != tt.expectedContext.Branch {
					t.Errorf("GithubActionsContext() = %v; expected %v", result, tt.expectedContext)
				}
			}

			for key := range tt.envVars {
				if err := os.Unsetenv(key); err != nil {
					t.Fatalf("os.Unsetenv(%v) = %v", key, err)
				}
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

func TestQualifiedRefName(t *testing.T) {
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
			expectedOutput: "refs/heads/main",
		},
		{
			name:           "Fully-qualified ref",
			refName:        "refs/tags/v1.0.0",
			expectedOutput: "refs/tags/v1.0.0",
		},
		{
			name:           "Partially-qualified ref",
			refName:        "heads/main",
			expectedOutput: "refs/heads/main",
		},
		{
			name:           "Partially-qualified ref",
			refName:        "tags/v1.0.0",
			expectedOutput: "refs/tags/v1.0.0",
		},
		{
			name:           "Unqualified ref with heads default #1",
			refName:        "main",
			defaultPrefix:  "heads",
			expectedOutput: "refs/heads/main",
		},
		{
			name:           "Unqualified ref with heads default #2",
			refName:        "feature/branch",
			defaultPrefix:  "heads",
			expectedOutput: "refs/heads/feature/branch",
		},

		{
			name:           "Unqualified ref with tags default",
			refName:        "v1.0.0",
			defaultPrefix:  "tags",
			expectedOutput: "refs/tags/v1.0.0",
		},
		{
			name:           "Partially-qualified ref with mismatched default prefix",
			refName:        "heads/main",
			defaultPrefix:  "tags",
			expectedOutput: "refs/heads/main",
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
			result, err := QualifiedRefName(tt.refName, tt.defaultPrefix)
			if (err != nil) != tt.expectedError {
				t.Errorf("QualifiedRefName() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if result != tt.expectedOutput {
				t.Errorf("QualifiedRefName() = %v, expected %v", result, tt.expectedOutput)
			}
		})
	}
}

func TestBuildCommitMessage(t *testing.T) {
	tests := []struct {
		name           string
		viperSettings  map[string]any
		expectedOutput string
	}{
		{
			name:           "No message and no trailers",
			viperSettings:  map[string]any{},
			expectedOutput: "",
		},
		{
			name: "Only message",
			viperSettings: map[string]any{
				"message": "This is a commit message",
			},
			expectedOutput: "This is a commit message",
		},
		{
			name: "Message with long title",
			viperSettings: map[string]any{
				"message": "This is a very long commit message title that exceeds seventy-two characters and should trigger a warning",
			},
			expectedOutput: "This is a very long commit message title that exceeds seventy-two characters and should trigger a warning",
		},
		{
			name: "Message with trailers",
			viperSettings: map[string]any{
				"message":      "This is a commit message",
				"user-trailer": "Co-Authored-By",
				"user-name":    "John Doe",
				"user-email":   "john.doe@example.com",
			},
			expectedOutput: "This is a commit message\n\nCo-Authored-By: John Doe <john.doe@example.com>",
		},
		{
			name: "Message with author trailer disabled",
			viperSettings: map[string]any{
				"message":      "This is a commit message",
				"user-trailer": "",
				"user-name":    "John Doe",
				"user-email":   "john.doe@example.com",
			},
			expectedOutput: "This is a commit message",
		},
		{
			name: "Only trailers",
			viperSettings: map[string]any{
				"user-trailer": "Co-Authored-By",
				"user-name":    "John Doe",
				"user-email":   "john.doe@example.com",
			},
			expectedOutput: "\nCo-Authored-By: John Doe <john.doe@example.com>",
		},
		{
			name: "Multiple trailers",
			viperSettings: map[string]any{
				"message":      "This is a commit message",
				"user-trailer": "Co-Authored-By",
				"user-name":    "John Doe",
				"user-email":   "john.doe@example.com",
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
		viperSettings  map[string]any
		expectedOutput []string
	}{
		{
			name:           "No trailers",
			viperSettings:  map[string]any{},
			expectedOutput: []string{},
		},
		{
			name: "Author trailer only",
			viperSettings: map[string]any{
				"user-trailer": "Co-Authored-By",
				"user-name":    "John Doe",
				"user-email":   "john.doe@example.com",
			},
			expectedOutput: []string{"Co-Authored-By: John Doe <john.doe@example.com>"},
		},
		{
			name: "Author trailer disabled",
			viperSettings: map[string]any{
				"user-trailer": "",
				"user-name":    "John Doe",
				"user-email":   "john.doe@example.com",
			},
			expectedOutput: []string{},
		},
		{
			name: "Multiple trailers",
			viperSettings: map[string]any{
				"user-trailer": "Co-Authored-By",
				"user-name":    "John Doe",
				"user-email":   "john.doe@example.com",
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
			viperSettings: map[string]any{
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
