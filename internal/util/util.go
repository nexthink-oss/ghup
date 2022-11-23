package util

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Coalesce returns the first not empty string
func Coalesce(values ...string) (value string) {
	for _, value = range values {
		if value != "" {
			return
		}
	}
	return
}

func BuildCommitMessage(withSignoff bool) (message string) {
	messageParts := []string{}
	if message := viper.GetString("message"); message != "" {
		messageParts = append(messageParts, message)
	}
	if withSignoff {
		if committer := BuildCommitter(); committer != "" {
			messageParts = append(messageParts, fmt.Sprintf("Signed-off-by: %s", committer))
		}
	}
	message = strings.Join(messageParts, "\n")
	return
}

func BuildCommitter() (committer string) {
	var userParts []string
	if userName := viper.GetString("user.name"); userName != "" {
		userParts = append(userParts, userName)
	}
	if userEmail := viper.GetString("user.email"); userEmail != "" {
		userParts = append(userParts, fmt.Sprintf("<%s>", userEmail))
	}
	if len(userParts) > 0 {
		committer = strings.Join(userParts, " ")
	}
	return
}
