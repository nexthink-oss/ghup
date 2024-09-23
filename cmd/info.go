package cmd

import (
	"fmt"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type info struct {
	HasToken      bool     `yaml:"hasToken"`
	Trailers      []string `yaml:"trailers,omitempty"`
	Owner         string
	Repository    string
	Branch        string
	Commit        string
	IsClean       bool                   `yaml:"isClean"`
	CommitMessage githubv4.CommitMessage `yaml:"commitMessage"`
}

var infoCmd = &cobra.Command{
	Use:     "info [flags]",
	Short:   "Dump info on the current context",
	Args:    cobra.NoArgs,
	PreRunE: validateFlags,
	RunE:    runInfoCmd,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfoCmd(cmd *cobra.Command, args []string) (err error) {
	i := info{
		HasToken:      len(viper.GetString("token")) > 0,
		Trailers:      util.BuildTrailers(),
		Owner:         owner,
		Repository:    repo,
		Branch:        branch,
		CommitMessage: remote.CommitMessage(util.BuildCommitMessage()),
	}

	if localRepo != nil {
		i.Commit = localRepo.HeadCommit()

		status, err := localRepo.Status()
		if err == nil {
			i.IsClean = status.IsClean()
		}
	}

	fmt.Print(util.EncodeYAML(&i))
	return
}
