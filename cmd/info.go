package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type info struct {
	HasToken   bool                   `json:"has_token"`
	Owner      string                 `json:"owner"`
	Repository string                 `json:"repository"`
	Branch     string                 `json:"branch"`
	Commit     string                 `json:"commit,omitempty"`
	Clean      bool                   `json:"clean"`
	Message    githubv4.CommitMessage `json:"message"`
	Trailers   []string               `json:"trailers,omitempty"`
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
		HasToken:   len(viper.GetString("token")) > 0,
		Trailers:   util.BuildTrailers(),
		Owner:      owner,
		Repository: repo,
		Branch:     branch,
		Message:    remote.CommitMessage(util.BuildCommitMessage()),
	}

	if localRepo != nil {
		i.Commit = localRepo.HeadCommit()

		status, err := localRepo.Status()
		if err == nil {
			i.Clean = status.IsClean()
		}
	}

	m, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return err
	}

	fmt.Print(string(m))
	return
}
