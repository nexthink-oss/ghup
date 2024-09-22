package util

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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

func EncodeYAML(obj any) string {
	var b bytes.Buffer

	e := yaml.NewEncoder(&b)
	e.SetIndent(2)

	_ = e.Encode(obj)

	return b.String()
}

func IsCommitHash(ref string) bool {
	commitHashPattern := `^[0-9a-f]{7,40}$`
	matched, _ := regexp.MatchString(commitHashPattern, ref)
	return matched
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
