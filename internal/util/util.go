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
	if author := viper.GetString("author"); author != "" && withSignoff {
		messageParts = append(messageParts, fmt.Sprintf("Signed-off-by: %s", author))
	}
	message = strings.Join(messageParts, "\n")
	return
}
