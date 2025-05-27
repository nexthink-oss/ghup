package util

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/spf13/viper"
)

type RepositoryContext struct {
	Owner  string
	Name   string
	Branch string
}

// IsBinaryInPath checks if a given binary is available in the PATH
func IsBinaryInPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// GetCliAuthToken tries to get a GitHub token by execing `gh auth token`
func GetCliAuthToken() string {
	if !viper.GetBool("no-cli-token") && IsBinaryInPath("gh") {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		ghCmd := exec.CommandContext(ctx, "gh", "auth", "token")
		output, err := ghCmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output))
		}
	}

	return ""
}

// GithubActionsBranch returns the branch name if running in a GitHub Actions environment, or an empty string
func GithubActionsBranch() (branch string) {
	if os.Getenv("GITHUB_REF_TYPE") == "branch" {
		branch = cmp.Or(
			os.Getenv("GITHUB_HEAD_REF"), // PR context
			os.Getenv("GITHUB_REF_NAME"), // other contexts
		)
	}
	return
}

// GithubActionsContext returns repository context if running in a GitHub Actions environment
func GithubActionsContext() *RepositoryContext {
	if owner, name, found := strings.Cut(os.Getenv("GITHUB_REPOSITORY"), "/"); found {
		return &RepositoryContext{
			Owner:  owner,
			Name:   name,
			Branch: GithubActionsBranch(),
		}
	}
	return nil
}

// IsCommitHash returns true if the ref looks like a commit hash
func IsCommitHash(ref string) bool {
	commitHashPattern := `^[0-9a-f]{7,40}$`
	matched, _ := regexp.MatchString(commitHashPattern, ref)
	return matched
}

// IsValidRefName checks if the ref name matches git ref requirements
func IsValidRefName(refName string) error {
	if refName == "" {
		return errors.New("empty ref name")
	}

	if refName == "@" {
		return errors.New("ref name cannot be @")
	}

	// They can include slash / for hierarchical (directory) grouping, but no slash-separated component can begin with a dot `.` or end with the sequence `.lock`.
	for _, part := range strings.Split(refName, "/") {
		if strings.HasPrefix(part, ".") {
			return fmt.Errorf("no ref name component can begin with a dot: %s", refName)
		}
		if strings.HasSuffix(part, ".lock") {
			return fmt.Errorf("no ref name component can end with .lock: %s", refName)
		}
	}

	// They cannot have two consecutive dots `..` anywhere.
	if strings.Contains(refName, "..") {
		return fmt.Errorf("ref name cannot contain two consecutive dots: %s", refName)
	}

	// The cannot have ASCII control characters (i.e. bytes whose values are lower than \x20, or \x7F DEL), space, tilde `~`, caret `^`, colon `:`, question-mark `?`, asterisk `*`, open-bracket `[`, or backslash `\` anywhere.
	if strings.ContainsAny(refName, ` ~^:?*[\`) {
		return fmt.Errorf("ref contains invalid characters: %s", refName)
	}
	for _, c := range refName {
		if c < 0x20 || c == 0x7F {
			return fmt.Errorf("ref contains control characters: %s", refName)
		}
	}

	// They cannot begin or end with a slash / or contain multiple consecutive slashes.
	if strings.HasPrefix(refName, "/") || strings.HasSuffix(refName, "/") || strings.Contains(refName, "//") {
		return fmt.Errorf("ref name cannot begin or end with a slash: %s", refName)
	}

	// They cannot end with a dot `.`.
	if strings.HasSuffix(refName, ".") {
		return fmt.Errorf("ref name cannot end with a dot: %s", refName)
	}

	// They cannot contain a sequence @{.
	if strings.Contains(refName, "@{") {
		return fmt.Errorf("ref name cannot contain a sequence @{: %s", refName)
	}

	return nil
}

// QualifiedRefName normalizes the ref name to a fully qualified format
func QualifiedRefName(refName string, refType string) (string, error) {
	if err := IsValidRefName(refName); err != nil {
		return "", err
	}

	if strings.HasPrefix(refName, "refs/") {
		return refName, nil
	}

	if strings.HasPrefix(refName, "heads/") || strings.HasPrefix(refName, "tags/") {
		return strings.Join([]string{"refs", refName}, "/"), nil
	}

	refType = cmp.Or(refType, "heads")
	return strings.Join([]string{"refs", refType, refName}, "/"), nil
}

// BuildCommitMessage generates a commit message from the message and trailers configuration
func BuildCommitMessage() (message string) {
	messageParts := []string{}
	if message := viper.GetString("message"); message != "" {
		if strings.Index(message, "\n") > 72 {
			log.Warn("commit message title exceeds 72 characters and will be wrapped by GitHub")
		}
		messageParts = append(messageParts, message)
	}
	if trailers := BuildTrailers(); len(trailers) > 0 {
		messageParts = append(messageParts, "")
		messageParts = append(messageParts, trailers...)
	}
	message = strings.Join(messageParts, "\n")
	return
}

// BuildTrailers generates the complete list of trailers from the configuration
func BuildTrailers() (trailers []string) {
	if trailerKey := viper.GetString("user-trailer"); trailerKey != "" && trailerKey != "-" {
		var userParts []string
		if userName := viper.GetString("user-name"); userName != "" {
			userParts = append(userParts, userName)
		}
		if userEmail := viper.GetString("user-email"); userEmail != "" {
			userParts = append(userParts, fmt.Sprintf("<%s>", userEmail))
		}
		if len(userParts) > 0 {
			trailers = append(trailers, fmt.Sprintf("%s: %s", trailerKey, strings.Join(userParts, " ")))
		}
	}
	for key, value := range viper.GetStringMapString("trailer") {
		trailers = append(trailers, fmt.Sprintf("%s: %s", key, value))
	}
	return
}

func ShortJson(r any) string {
	bytes, err := json.Marshal(r)
	if err != nil {
		log.Fatalf("serialising object: %w", err)
	}

	return string(bytes)
}

func PrettyJson(r any) string {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatalf("serialising object: %w", err)
	}

	return string(bytes)
}

func SliceChain[T any](slices ...[]T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, slice := range slices {
			for _, e := range slice {
				if !yield(e) {
					return
				}
			}
		}
	}
}

func MapValues[T any](m map[string]T) []T {
	slice := make([]T, 0, len(m))
	for _, v := range m {
		slice = append(slice, v)
	}
	return slice
}
