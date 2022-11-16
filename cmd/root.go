package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/isometry/ghup/internal/local"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	localRepo *local.Repository
	token     string
	owner     string
	repo      string
	branch    string
	message   string
	author    string
	noSignOff bool
	force     bool
)

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
	var localOwner string
	var localName string
	localBranch := "main"

	localRepo = local.GetRepository(cwd)
	if localRepo != nil {
		localOwner = localRepo.Owner
		localName = localRepo.Name
		localBranch = localRepo.Branch
		author = localRepo.User
	}

	rootCmd.PersistentFlags().CountP("verbosity", "v", "verbosity")
	viper.BindPFlag("verbosity", rootCmd.PersistentFlags().Lookup("verbosity"))

	rootCmd.PersistentFlags().String("token", "", "GitHub Token or path/to/token-file")
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	rootCmd.MarkFlagRequired("token")

	rootCmd.PersistentFlags().StringP("owner", "o", localOwner, "repository owner")
	viper.BindPFlag("owner", rootCmd.PersistentFlags().Lookup("owner"))

	rootCmd.PersistentFlags().StringP("repo", "r", localName, "repository name")
	viper.BindPFlag("repo", rootCmd.PersistentFlags().Lookup("repo"))

	rootCmd.PersistentFlags().StringP("branch", "b", localBranch, "branch name")
	viper.BindPFlag("branch", rootCmd.PersistentFlags().Lookup("branch"))

	rootCmd.PersistentFlags().StringP("message", "m", "", "message")
	viper.BindPFlag("message", rootCmd.PersistentFlags().Lookup("message"))

	rootCmd.PersistentFlags().BoolVar(&noSignOff, "no-signoff", false, "don't add Signed-off-by to message")
	viper.BindPFlag("no_signoff", rootCmd.PersistentFlags().Lookup("no-signoff"))

	rootCmd.PersistentFlags().StringVar(&author, "author", author, "user details for sign-off")
	viper.BindPFlag("author", rootCmd.PersistentFlags().Lookup("author"))

	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "force action")
	viper.BindPFlag("force", rootCmd.PersistentFlags().Lookup("force"))
}

// initViper initializes Viper to load config from the environment
func initViper() {
	viper.SetEnvPrefix("GITHUB")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv() // read in environment variables that match bound variables
}

// initLogger initializes the logger subsystem
func initLogger() {
	log.SetHandler(cli.New(os.Stderr))

	verbosity := viper.GetInt("verbosity")
	log.SetLevel(log.Level(int(log.InfoLevel) - verbosity))
}

// validateFlags checks mandatory flags are valid and stores results in shared variables
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

	messageParts := []string{}
	if message := viper.GetString("message"); message != "" {
		messageParts = append(messageParts, message)
	}
	if author := viper.GetString("author"); author != "" && !noSignOff {
		messageParts = append(messageParts, fmt.Sprintf("Signed-off-by: %s", author))
	}
	message = strings.Join(messageParts, "\n")

	return nil
}
