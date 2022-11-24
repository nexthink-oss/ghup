package util

import (
	"fmt"
	"strings"

	"github.com/apex/log"
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

func BuildCommitMessage() (message string) {
	messageParts := []string{}
	if message := viper.GetString("message"); message != "" {
		if strings.Index(message, "\n") > 72 {
			log.Warn("commit message title exceeds 72 characters and will be wrapped by GitHub")
		}
		messageParts = append(messageParts, message)
	}
	if trailer := BuildTrailer(); trailer != "" {
		messageParts = append(messageParts, "", trailer)
	}
	message = strings.Join(messageParts, "\n")
	return
}

func BuildTrailer() (trailer string) {
	if trailerKey := viper.GetString("trailer.key"); trailerKey != "" {
		var userParts []string
		if userName := viper.GetString("trailer.name"); userName != "" {
			userParts = append(userParts, userName)
		}
		if userEmail := viper.GetString("trailer.email"); userEmail != "" {
			userParts = append(userParts, fmt.Sprintf("<%s>", userEmail))
		}
		if len(userParts) > 0 {
			trailer = fmt.Sprintf("%s: %s", trailerKey, strings.Join(userParts, " "))
		}
	}
	return
}
