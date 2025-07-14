package remote

import (
	"reflect"
	"testing"

	"github.com/shurcooL/githubv4"
)

func TestCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected githubv4.CommitMessage
	}{
		{
			name:    "Empty message",
			message: "",
			expected: githubv4.CommitMessage{
				Headline: githubv4.String(""),
			},
		},
		{
			name:    "Single line message",
			message: "This is a headline",
			expected: githubv4.CommitMessage{
				Headline: githubv4.String("This is a headline"),
			},
		},
		{
			name:    "Multi-line message",
			message: "This is a headline\nThis is the body",
			expected: githubv4.CommitMessage{
				Headline: githubv4.String("This is a headline"),
				Body:     githubv4.NewString(githubv4.String("This is the body")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommitMessage(tt.message)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("CommitMessage(%v) = %v; expected %v", tt.message, result, tt.expected)
			}
		})
	}
}

func TestPullRequestAutoMerge(t *testing.T) {
	tests := []struct {
		name         string
		pr           PullRequest
		expectedMode string
	}{
		{
			name: "AutoMerge disabled",
			pr: PullRequest{
				RepoId:        "repo123",
				Head:          "feature",
				Base:          "main",
				Title:         "Test PR",
				AutoMergeMode: AutoMergeOff,
			},
			expectedMode: AutoMergeOff,
		},
		{
			name: "AutoMerge merge method",
			pr: PullRequest{
				RepoId:        "repo123",
				Head:          "feature",
				Base:          "main",
				Title:         "Test PR",
				AutoMergeMode: AutoMergeMerge,
			},
			expectedMode: AutoMergeMerge,
		},
		{
			name: "AutoMerge squash method",
			pr: PullRequest{
				RepoId:        "repo123",
				Head:          "feature",
				Base:          "main",
				Title:         "Test PR",
				AutoMergeMode: AutoMergeSquash,
			},
			expectedMode: AutoMergeSquash,
		},
		{
			name: "AutoMerge rebase method",
			pr: PullRequest{
				RepoId:        "repo123",
				Head:          "feature",
				Base:          "main",
				Title:         "Test PR",
				AutoMergeMode: AutoMergeRebase,
			},
			expectedMode: AutoMergeRebase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pr.AutoMergeMode != tt.expectedMode {
				t.Errorf("PullRequest.AutoMergeMode = %v; expected %v", tt.pr.AutoMergeMode, tt.expectedMode)
			}
		})
	}
}

func TestRepositoryInfoAutoMergeAllowed(t *testing.T) {
	tests := []struct {
		name     string
		repoInfo repositoryInfo
		expected bool
	}{
		{
			name: "AutoMerge not allowed",
			repoInfo: repositoryInfo{
				NodeID:             "repo123",
				IsEmpty:            false,
				AutoMergeAllowed:   false,
				MergeCommitAllowed: true,
				SquashMergeAllowed: true,
				RebaseMergeAllowed: true,
			},
			expected: false,
		},
		{
			name: "AutoMerge allowed",
			repoInfo: repositoryInfo{
				NodeID:             "repo123",
				IsEmpty:            false,
				AutoMergeAllowed:   true,
				MergeCommitAllowed: true,
				SquashMergeAllowed: true,
				RebaseMergeAllowed: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.repoInfo.AutoMergeAllowed != tt.expected {
				t.Errorf("repositoryInfo.AutoMergeAllowed = %v; expected %v", tt.repoInfo.AutoMergeAllowed, tt.expected)
			}
		})
	}
}

func TestRepositoryInfoIsAutoMergeMethodSupported(t *testing.T) {
	repoInfo := repositoryInfo{
		NodeID:             "repo123",
		IsEmpty:            false,
		AutoMergeAllowed:   true,
		MergeCommitAllowed: true,
		SquashMergeAllowed: false,
		RebaseMergeAllowed: true,
	}

	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		{
			name:     "Off method always supported",
			method:   AutoMergeOff,
			expected: true,
		},
		{
			name:     "Merge method supported",
			method:   AutoMergeMerge,
			expected: true,
		},
		{
			name:     "Squash method not supported",
			method:   AutoMergeSquash,
			expected: false,
		},
		{
			name:     "Rebase method supported",
			method:   AutoMergeRebase,
			expected: true,
		},
		{
			name:     "Invalid method not supported",
			method:   "invalid",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repoInfo.IsAutoMergeMethodSupported(tt.method)
			if result != tt.expected {
				t.Errorf("IsAutoMergeMethodSupported(%s) = %v; expected %v", tt.method, result, tt.expected)
			}
		})
	}
}

func TestRepositoryInfoGetSupportedAutoMergeMethods(t *testing.T) {
	tests := []struct {
		name     string
		repoInfo repositoryInfo
		expected []string
	}{
		{
			name: "All methods supported",
			repoInfo: repositoryInfo{
				MergeCommitAllowed: true,
				SquashMergeAllowed: true,
				RebaseMergeAllowed: true,
			},
			expected: []string{AutoMergeOff, AutoMergeMerge, AutoMergeSquash, AutoMergeRebase},
		},
		{
			name: "Only merge supported",
			repoInfo: repositoryInfo{
				MergeCommitAllowed: true,
				SquashMergeAllowed: false,
				RebaseMergeAllowed: false,
			},
			expected: []string{AutoMergeOff, AutoMergeMerge},
		},
		{
			name: "Only off supported",
			repoInfo: repositoryInfo{
				MergeCommitAllowed: false,
				SquashMergeAllowed: false,
				RebaseMergeAllowed: false,
			},
			expected: []string{AutoMergeOff},
		},
		{
			name: "Squash and rebase supported",
			repoInfo: repositoryInfo{
				MergeCommitAllowed: false,
				SquashMergeAllowed: true,
				RebaseMergeAllowed: true,
			},
			expected: []string{AutoMergeOff, AutoMergeSquash, AutoMergeRebase},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.repoInfo.GetSupportedAutoMergeMethods()
			if len(result) != len(tt.expected) {
				t.Errorf("GetSupportedAutoMergeMethods() length = %d; expected %d", len(result), len(tt.expected))
				return
			}
			for i, method := range result {
				if method != tt.expected[i] {
					t.Errorf("GetSupportedAutoMergeMethods()[%d] = %s; expected %s", i, method, tt.expected[i])
				}
			}
		})
	}
}

func TestGetAutoMergeChoices(t *testing.T) {
	choices := GetAutoMergeChoices()
	expected := []string{AutoMergeOff, AutoMergeMerge, AutoMergeSquash, AutoMergeRebase}

	if len(choices) != len(expected) {
		t.Errorf("GetAutoMergeChoices() length = %d; expected %d", len(choices), len(expected))
		return
	}

	for i, choice := range choices {
		if choice != expected[i] {
			t.Errorf("GetAutoMergeChoices()[%d] = %s; expected %s", i, choice, expected[i])
		}
	}
}
