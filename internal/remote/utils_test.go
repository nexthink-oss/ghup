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
		name     string
		pr       PullRequest
		expected bool
	}{
		{
			name: "AutoMerge disabled",
			pr: PullRequest{
				RepoId:    "repo123",
				Head:      "feature",
				Base:      "main",
				Title:     "Test PR",
				AutoMerge: false,
			},
			expected: false,
		},
		{
			name: "AutoMerge enabled",
			pr: PullRequest{
				RepoId:    "repo123",
				Head:      "feature",
				Base:      "main",
				Title:     "Test PR",
				AutoMerge: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pr.AutoMerge != tt.expected {
				t.Errorf("PullRequest.AutoMerge = %v; expected %v", tt.pr.AutoMerge, tt.expected)
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
				NodeID:           "repo123",
				IsEmpty:          false,
				AutoMergeAllowed: false,
			},
			expected: false,
		},
		{
			name: "AutoMerge allowed",
			repoInfo: repositoryInfo{
				NodeID:           "repo123",
				IsEmpty:          false,
				AutoMergeAllowed: true,
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
