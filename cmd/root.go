package cmd

import (
	"fmt"
	"os"

	"github.com/isometry/ghup/internal/gitutil"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var token string
var remote *gitutil.Remote
var owner string
var repo string
var branch string
var message string

var rootCmd = &cobra.Command{
	Use:               "ghup",
	Short:             "Update GitHub content and tags via API",
	SilenceUsage:      true,
	PersistentPreRunE: validateFlags,
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
	var defaultOwner string
	var defaultRepo string
	defaultBranch := "main"

	remote = gitutil.NewRemote(cwd)
	if remote != nil {
		defaultOwner = remote.Owner
		defaultRepo = remote.Repository
		defaultBranch = remote.Branch
	}

	rootCmd.PersistentFlags().CountP("verbosity", "v", "verbosity")
	viper.BindPFlag("verbosity", rootCmd.PersistentFlags().Lookup("verbosity"))

	rootCmd.PersistentFlags().String("token", "", "GitHub Personal Access Token")
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	rootCmd.MarkFlagRequired("token")

	rootCmd.PersistentFlags().StringP("owner", "o", defaultOwner, "repository owner")
	viper.BindPFlag("owner", rootCmd.PersistentFlags().Lookup("owner"))

	rootCmd.PersistentFlags().StringP("repo", "r", defaultRepo, "repository name")
	viper.BindPFlag("repo", rootCmd.PersistentFlags().Lookup("repo"))

	rootCmd.PersistentFlags().StringP("branch", "b", defaultBranch, "branch name")
	viper.BindPFlag("branch", rootCmd.PersistentFlags().Lookup("branch"))

	rootCmd.PersistentFlags().StringP("message", "m", "", "message")
	viper.BindPFlag("message", rootCmd.PersistentFlags().Lookup("message"))

	rootCmd.PersistentFlags().BoolP("force", "f", false, "force override")
	viper.BindPFlag("force", rootCmd.PersistentFlags().Lookup("force"))
}

// initViper initializes Viper to load config from the environment
func initViper() {
	viper.SetEnvPrefix("GITHUB")
	viper.AutomaticEnv() // read in environment variables that match bound variables
}

func initLogger() {
	log.SetHandler(cli.New(os.Stderr))

	verbosity := viper.GetInt("verbosity")
	log.SetLevel(log.Level(int(log.ErrorLevel) - verbosity))
}

func validateFlags(cmd *cobra.Command, args []string) error {
	token = viper.GetString("token")
	if token == "" {
		return fmt.Errorf("invalid token: '%+v'", token)
	}

	owner = viper.GetString("owner")
	if owner == "" {
		return fmt.Errorf("invalid owner: '%+v'", owner)
	}

	repo = viper.GetString("repo")
	if repo == "" {
		return fmt.Errorf("invalid repo: '%+v'", repo)
	}

	branch = viper.GetString("branch")
	if branch == "" {
		return fmt.Errorf("invalid branch: '%+v'", branch)
	}

	message = viper.GetString("message")

	return nil
}