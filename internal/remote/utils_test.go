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
