package cmd

import (
	"os"

	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
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
	Use:   "info [flags]",
	Short: "Dump info on the current context",
	Args:  cobra.NoArgs,
	RunE:  runInfoCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := infoCmd.Flags()

	addBranchFlag(flags)
	addCommitMessageFlags(flags)

	flags.SetNormalizeFunc(normalizeFlags)
	rootCmd.AddCommand(infoCmd)
}

func runInfoCmd(cmd *cobra.Command, args []string) (err error) {
	i := info{
		HasToken:   len(token) > 0,
		Trailers:   util.BuildTrailers(),
		Owner:      viper.GetString("owner"),
		Repository: viper.GetString("repo"),
		Branch:     viper.GetString("branch"),
		Commit:     localRepo.HeadCommit(),
		Message:    remote.CommitMessage(util.BuildCommitMessage()),
	}

	status, err := localRepo.Status()
	if err == nil {
		i.Clean = status.IsClean()
	}

	return util.FprintJSON(os.Stdout, i)
}
