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
	version string = "snapshot"
	commit  string = "unknown"
	date    string = "unknown"

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
	Version:      fmt.Sprintf("%s-%s (built %s)", version, commit, date),
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

	rootCmd.PersistentFlags().StringP("owner", "o", defaultOwner, "repository owner `name`")
	viper.BindPFlag("owner", rootCmd.PersistentFlags().Lookup("owner"))
	viper.BindEnv("owner", "GHUP_OWNER", "GITHUB_OWNER")

	rootCmd.PersistentFlags().StringP("repo", "r", defaultRepo, "repository `name`")
	viper.BindPFlag("repo", rootCmd.PersistentFlags().Lookup("repo"))

	rootCmd.PersistentFlags().StringP("branch", "b", defaultBranch, "target branch `name`")
	viper.BindPFlag("branch", rootCmd.PersistentFlags().Lookup("branch"))
	viper.BindEnv("branch", "GHUP_BRANCH", "CHANGE_BRANCH", "BRANCH_NAME", "GIT_BRANCH")

	rootCmd.PersistentFlags().StringP("message", "m", "Commit via API", "message")
	viper.BindPFlag("message", rootCmd.PersistentFlags().Lookup("message"))

	rootCmd.PersistentFlags().String("author.trailer", "Co-Authored-By", "`key` for commit author trailer (blank to disable)")
	viper.BindPFlag("author.trailer", rootCmd.PersistentFlags().Lookup("author.trailer"))
	viper.BindEnv("author.trailer", "GHUP_TRAILER_KEY")

	rootCmd.PersistentFlags().String("user.name", defaultUserName, "`name` for commit author trailer")
	viper.BindPFlag("user.name", rootCmd.PersistentFlags().Lookup("user.name"))
	viper.BindEnv("user.name", "GHUP_TRAILER_NAME", "GIT_COMMITTER_NAME", "GIT_AUTHOR_NAME")

	rootCmd.PersistentFlags().String("user.email", defaultUserEmail, "`email` for commit author trailer")
	viper.BindPFlag("user.email", rootCmd.PersistentFlags().Lookup("user.email"))
	viper.BindEnv("user.email", "GHUP_TRAILER_EMAIL", "GIT_COMMITTER_EMAIL", "GIT_AUTHOR_EMAIL")

	rootCmd.PersistentFlags().StringToString("trailer", nil, "extra `key=value` commit trailers")
	viper.BindPFlag("trailer", rootCmd.PersistentFlags().Lookup("trailer"))

	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "force action")
	viper.BindPFlag("force", rootCmd.PersistentFlags().Lookup("force"))

	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false
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
