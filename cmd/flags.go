package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"strings"

	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type FlagConfig struct {
	Env []string
}

type FlagConfigMap map[string]FlagConfig

// flagConfigMap maps flag names to environment variable bindings.
// The first environment variable that is set will be used.
// Flags not in the map are still bound to `GHUP_<FLAG_NAME>`.
var flagConfigMap = FlagConfigMap{
	"token":          {Env: []string{"GHUP_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"}},
	"owner":          {Env: []string{"GHUP_OWNER", "GITHUB_OWNER", "GITHUB_REPOSITORY_OWNER"}},
	"repo":           {Env: []string{"GHUP_REPO", "GITHUB_REPO", "GITHUB_REPOSITORY_NAME"}},
	"branch":         {Env: []string{"GHUP_BRANCH", "CHANGE_BRANCH", "BRANCH_NAME", "GIT_BRANCH"}},
	"ref":            {Env: []string{"GHUP_REF", "GITHUB_REF"}},
	"author-trailer": {Env: []string{"GHUP_AUTHOR_TRAILER", "GHUP_TRAILER_KEY"}},
	"user-name":      {Env: []string{"GHUP_TRAILER_NAME", "GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"}},
	"user-email":     {Env: []string{"GHUP_TRAILER_EMAIL", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"}},
	"pr-title":       {Env: []string{"GHUP_PR_TITLE"}},
	"pr-body":        {Env: []string{"GHUP_PR_BODY"}},
	"pr-draft":       {Env: []string{"GHUP_PR_DRAFT"}},
}

func bindEnvFlag(flag *pflag.Flag) {
	name := flag.Name
	if flagConfig, ok := flagConfigMap[name]; ok && len(flagConfig.Env) > 0 {
		args := make([]string, 1+len(flagConfig.Env))
		args[0] = name
		copy(args[1:], flagConfig.Env)
		viper.BindEnv(args...)
	}
}

func normalizeFlags(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	// Normalize 'foo.bar' to 'foo-bar'
	name = strings.Replace(name, ".", "-", -1)

	// Support alternative flag names
	switch name {
	case "name":
		name = "repo"
		break
	case "author-trailer":
		name = "user-trailer"
		break
	case "author-name":
		name = "user-name"
		break
	case "author-email":
		name = "user-email"
		break
	}

	return pflag.NormalizedName(name)
}

// processFlags binds flags to viper and checks mandatory flags are set
func processFlags(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	viper.BindPFlags(flags)
	flags.VisitAll(bindEnvFlag)

	// bindEnvRootFlags()

	errs := make([]error, 0)

	token = cmp.Or[string](viper.GetString("token"), util.GetCliAuthToken())
	if token == "" {
		errs = append(errs, fmt.Errorf("token is required"))
	}

	if viper.GetString("owner") == "" {
		errs = append(errs, fmt.Errorf("owner is required"))
	}

	if viper.GetString("repo") == "" {
		errs = append(errs, fmt.Errorf("repo is required"))
	}

	if flags.Lookup("branch") != nil && viper.GetString("branch") == "" {
		errs = append(errs, fmt.Errorf("branch is required"))
	}

	return errors.Join(errs...)
}

func addDryRunFlag(flagSet *pflag.FlagSet) {
	flagSet.BoolP("dry-run", "n", false, "dry-run mode")
}

func addForceFlag(flagSet *pflag.FlagSet) {
	flagSet.BoolP("force", "f", false, "force operation")
}

func addBranchFlag(flagSet *pflag.FlagSet) {
	flagSet.StringP("branch", "b", localRepo.Branch, "target branch `name`")
}

func addCommitMessageFlags(flagSet *pflag.FlagSet) {
	flagSet.StringP("message", "m", "Commit via API", "commit message")
	flagSet.String("author-trailer", "Co-Authored-By", "`key` for commit author trailer (blank to disable)")
	flagSet.String("user-name", localRepo.User.Name, "`name` for commit author trailer")
	flagSet.String("user-email", localRepo.User.Email, "`email` for commit author trailer")
	flagSet.StringToString("trailer", nil, "extra `key=value` commit trailers")
}

func addPullRequestFlags(flagSet *pflag.FlagSet) {
	flagSet.String("pr-title", "", "pull request title")
	flagSet.String("pr-body", "", "pull request body")
	flagSet.Bool("pr-draft", false, "create pull request in draft mode")
}
