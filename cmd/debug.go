package cmd

import (
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
)

type debugInfo struct {
	HasToken bool                   `json:"has_token"`
	Remote   remote.Repo            `json:"remote"`
	Branch   string                 `json:"branch"`
	Commit   string                 `json:"commit,omitempty"`
	Clean    bool                   `json:"clean"`
	Message  githubv4.CommitMessage `json:"message"`
	Trailers []string               `json:"trailers,omitempty"`
}

var debugCmd = &cobra.Command{
	Use:     "debug [flags]",
	Aliases: []string{"info"},
	Short:   "Dump contextual information to aid debugging.",
	Args:    cobra.NoArgs,
	RunE:    runDebugCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := debugCmd.Flags()

	addBranchFlag(flags)
	addCommitMessageFlags(flags)

	flags.SetNormalizeFunc(normalizeFlags)
	rootCmd.AddCommand(debugCmd)
}

func runDebugCmd(cmd *cobra.Command, args []string) (err error) {
	report := debugInfo{
		HasToken: len(githubToken) > 0,
		Trailers: util.BuildTrailers(),
		Remote: remote.Repo{
			Owner: repoOwner,
			Name:  repoName,
		},
		Branch:  branchName,
		Commit:  localRepo.HeadCommit(),
		Message: remote.CommitMessage(util.BuildCommitMessage()),
	}

	status, err := localRepo.Status()
	if err == nil {
		report.Clean = status.IsClean()
	}

	commandOutput = report

	return nil
}
