package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const envPrefix = "GHUP"

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
		_ = viper.BindEnv(args...)
	}
}

func normalizeFlags(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	// Normalize 'foo.bar' to 'foo-bar'
	name = strings.ReplaceAll(name, ".", "-")

	// Support alternative flag names
	switch name {
	case "name":
		name = "repo"
	case "author-trailer":
		name = "user-trailer"
	case "author-name":
		name = "user-name"
	case "author-email":
		name = "user-email"
	}

	return pflag.NormalizedName(name)
}

// commonSetup binds flags to viper and checks mandatory flags are set
func commonSetup(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	errs := make([]error, 0)

	configPaths, err := cmd.Flags().GetStringSlice("config-path")
	if err == nil {
		for _, configPath := range configPaths {
			viper.AddConfigPath(configPath)
		}
		configName, err := cmd.Flags().GetString("config-name")
		if err != nil {
			errs = append(errs, fmt.Errorf("loading config-name flag: %w", err))
		} else if configName != "" {
			viper.SetConfigName(configName)
			errs = append(errs, viper.ReadInConfig())
		}
	}

	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()      // read in environment variables that match bound variables
	viper.AllowEmptyEnv(true) // respect empty environment variables

	// Bind flags to viper variables
	if err := viper.BindPFlags(flags); err != nil {
		errs = append(errs, fmt.Errorf("binding flags: %w", err))
	}

	// Bind flags non-default environment variables
	flags.VisitAll(bindEnvFlag)

	// Initialize logging
	log.SetHandler(cli.New(cmd.ErrOrStderr()))
	log.SetLevel(log.Level(int(log.WarnLevel) - viper.GetInt("verbose")))

	if viper.GetString("token") == "" {
		token := util.GetCliAuthToken()
		if token != "" {
			viper.Set("token", token)
		} else {
			errs = append(errs, fmt.Errorf("token is required"))
		}
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

	switch viper.GetString("output-format") {
	case "json", "j":
		jsonEncoder := json.NewEncoder(cmd.OutOrStdout())
		if !viper.GetBool("compact") {
			jsonEncoder.SetIndent("", "  ")
		}
		outputEncoder = jsonEncoder
	case "yaml", "y":
		yamlEncoder := yaml.NewEncoder(cmd.OutOrStdout())
		yamlEncoder.SetIndent(2)
		outputEncoder = yamlEncoder
	default:
		errs = append(errs, errors.New("invalid output format"))
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
	flagSet.String("user-trailer", "Co-Authored-By", "`key` for commit author trailer (blank to disable)")
	flagSet.String("user-name", localRepo.User.Name, "`name` for commit author trailer")
	flagSet.String("user-email", localRepo.User.Email, "`email` for commit author trailer")
	flagSet.StringToString("trailer", nil, "extra `key=value` commit trailers")
}

func addPullRequestFlags(flagSet *pflag.FlagSet) {
	flagSet.String("pr-title", "", "pull request title")
	flagSet.String("pr-body", "", "pull request body")
	flagSet.Bool("pr-draft", false, "create pull request in draft mode")
}
