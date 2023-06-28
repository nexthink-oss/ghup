package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/nexthink-oss/ghup/internal/local"
	"github.com/nexthink-oss/ghup/internal/util"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	buildVersion string = "snapshot"
	buildCommit  string = "unknown"
	buildDate    string = "unknown"

	localRepo        *local.Repository
	defaultUserName  string
	defaultUserEmail string
	defaultOwner     string
	defaultRepo      string
	defaultBranch    string = "main"

	owner   string
	repo    string
	branch  string
	message string
	force   bool
)

var rootCmd = &cobra.Command{
	Use:          "ghup",
	Short:        "Update GitHub content and tags via API",
	SilenceUsage: true,
	Version:      fmt.Sprintf("%s-%s (built %s)", buildVersion, buildCommit, buildDate),
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

	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(99)
	}

	localRepo = local.GetRepository(cwd)
	if localRepo != nil {
		defaultUserName = localRepo.UserName
		defaultUserEmail = localRepo.UserEmail
		defaultOwner = localRepo.Owner
		defaultRepo = localRepo.Name
		defaultBranch = localRepo.Branch
	}

	rootCmd.PersistentFlags().CountP("verbosity", "v", "verbosity")
	viper.BindPFlag("verbosity", rootCmd.PersistentFlags().Lookup("verbosity"))

	rootCmd.PersistentFlags().String("token", "", "GitHub Token or path/to/token-file")
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindEnv("token", "GHUP_TOKEN", "GITHUB_TOKEN")

	rootCmd.PersistentFlags().StringP("owner", "o", defaultOwner, "repository owner")
	viper.BindPFlag("owner", rootCmd.PersistentFlags().Lookup("owner"))
	viper.BindEnv("owner", "GHUP_OWNER", "GITHUB_OWNER")

	rootCmd.PersistentFlags().StringP("repo", "r", defaultRepo, "repository name")
	viper.BindPFlag("repo", rootCmd.PersistentFlags().Lookup("repo"))

	rootCmd.PersistentFlags().StringP("branch", "b", defaultBranch, "target branch name")
	viper.BindPFlag("branch", rootCmd.PersistentFlags().Lookup("branch"))
	viper.BindEnv("branch", "GHUP_BRANCH", "CHANGE_BRANCH", "BRANCH_NAME", "GIT_BRANCH")

	rootCmd.PersistentFlags().StringP("message", "m", "Commit via API", "message")
	viper.BindPFlag("message", rootCmd.PersistentFlags().Lookup("message"))

	rootCmd.PersistentFlags().String("trailer.key", "Co-Authored-By", "key for commit trailer (blank to disable)")
	viper.BindPFlag("trailer.key", rootCmd.PersistentFlags().Lookup("trailer.key"))

	rootCmd.PersistentFlags().String("trailer.name", defaultUserName, "name for commit trailer")
	viper.BindPFlag("trailer.name", rootCmd.PersistentFlags().Lookup("trailer.name"))
	viper.BindEnv("trailer.name", "GHUP_USER_NAME", "GIT_COMMITTER_NAME", "GIT_AUTHOR_NAME")

	rootCmd.PersistentFlags().String("trailer.email", defaultUserEmail, "email for commit trailer")
	viper.BindPFlag("trailer.email", rootCmd.PersistentFlags().Lookup("trailer.email"))
	viper.BindEnv("trailer.email", "GHUP_USER_EMAIL", "GIT_COMMITTER_EMAIL", "GIT_AUTHOR_EMAIL")

	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "force action")
	viper.BindPFlag("force", rootCmd.PersistentFlags().Lookup("force"))
}

// initViper initializes Viper to load config from the environment
func initViper() {
	viper.SetEnvPrefix("GHUP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv() // read in environment variables that match bound variables
}

// initLogger initializes the logger subsystem
func initLogger() {
	log.SetHandler(cli.New(os.Stderr))

	verbosity := viper.GetInt("verbosity")
	log.SetLevel(log.Level(int(log.WarnLevel) - verbosity))
}

// validateFlags checks mandatory flags are valid and stores results in shared variables
func validateFlags(cmd *cobra.Command, args []string) error {
	owner = util.Coalesce(viper.GetString("owner"), defaultOwner)
	if owner == "" {
		return fmt.Errorf("no owner specified")
	}

	repo = util.Coalesce(viper.GetString("repo"), defaultRepo)
	if repo == "" {
		return fmt.Errorf("no repo specified")
	}

	branch = util.Coalesce(viper.GetString("branch"), defaultBranch)
	if branch == "" {
		return fmt.Errorf("no branch specified")
	}

	return nil
}
