package cmd

import (
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/nexthink-oss/ghup/internal/local"
	"github.com/nexthink-oss/ghup/internal/util"
)

type Encoder interface {
	Encode(any) error
}

var (
	version string = "snapshot"
	commit  string = "unknown"
	date    string = "unknown"

	localRepo local.Repository

	githubToken string
	repoOwner   string
	repoName    string
	branchName  string

	outputEncoder Encoder
	commandOutput any
)

var rootCmd = &cobra.Command{
	Use:                "ghup",
	Short:              "Update GitHub content and tags via API",
	SilenceUsage:       true,
	PersistentPreRunE:  commonSetup,
	PersistentPostRunE: encodeOutput,
	Version:            fmt.Sprintf("%s-%s (built %s)", version, commit, date),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// load defaults from local repository context, if available
	defaultsOnce.Do(loadDefaults)

	// override defaults from GitHub Actions environment variables
	// in case we're running without a checkout
	if actionsCtx := util.GithubActionsContext(); actionsCtx != nil {
		localRepo.Owner = actionsCtx.Owner
		localRepo.Name = actionsCtx.Name
		if actionsCtx.Branch != "" {
			localRepo.Branch = actionsCtx.Branch
		}
	}

	log.Debugf("local repository: %+v", localRepo)

	flags := rootCmd.Flags()
	persistentFlags := rootCmd.PersistentFlags()

	persistentFlags.StringSlice("config-path", []string{"."}, "configuration `name`")
	persistentFlags.StringP("config-name", "C", "", "configuration `name`")
	persistentFlags.String("token", "", "GitHub Token or path/to/token-file")
	persistentFlags.StringP("owner", "o", localRepo.Owner, "repository owner `name`")
	persistentFlags.StringP("repo", "r", localRepo.Name, "repository `name`")
	persistentFlags.Bool("no-cli-token", false, "disable fallback to GitHub CLI Token")
	persistentFlags.CountP("verbose", "v", "increase verbosity``")
	persistentFlags.StringP("output-format", "O", "json", "output `format` (json|j, yaml|y)")
	persistentFlags.Bool("compact", false, "compact output")

	flags.SortFlags = false
	persistentFlags.SortFlags = false
}

func encodeOutput(_ *cobra.Command, _ []string) error {
	if commandOutput != nil {
		return outputEncoder.Encode(commandOutput)
	}
	return nil
}
