package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/local"
	"github.com/nexthink-oss/ghup/internal/util"
)

var (
	version string = "snapshot"
	commit  string = "unknown"
	date    string = "unknown"

	localRepo local.Repository

	owner string
	repo  string
	token string
)

var rootCmd = &cobra.Command{
	Use:               "ghup",
	Short:             "Update GitHub content and tags via API",
	SilenceUsage:      true,
	PersistentPreRunE: processFlags,
	Version:           fmt.Sprintf("%s-%s (built %s)", version, commit, date),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initViper, initLogger)

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

	persistentFlags.String("token", "", "GitHub Token or path/to/token-file")
	persistentFlags.StringP("owner", "o", localRepo.Owner, "repository owner `name`")
	persistentFlags.StringP("repo", "r", localRepo.Name, "repository `name`")
	persistentFlags.Bool("no-cli-token", false, "disable fallback to GitHub CLI Token")
	persistentFlags.CountP("verbose", "v", "increase verbosity``")

	flags.SortFlags = false
	persistentFlags.SortFlags = false
}

// initViper initializes Viper to load config from the environment
func initViper() {
	viper.SetEnvPrefix("GHUP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()      // read in environment variables that match bound variables
	viper.AllowEmptyEnv(true) // respect empty environment variables
}

// initLogger initializes the logger subsystem
func initLogger() {
	log.SetHandler(cli.New(os.Stderr))

	verbosity := viper.GetInt("verbose")
	log.SetLevel(log.Level(int(log.WarnLevel) - verbosity))
}
