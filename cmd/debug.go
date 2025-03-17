package cmd

import (
	"fmt"

	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
)

type DebugOutput struct {
	Remote   remote.Repo            `json:"remote" yaml:"remote"`
	HasToken bool                   `json:"has_token" yaml:"has_token"`
	Branch   string                 `json:"branch" yaml:"branch"`
	Commit   string                 `json:"commit,omitempty" yaml:"commit,omitempty"`
	Clean    bool                   `json:"clean" yaml:"clean"`
	Message  githubv4.CommitMessage `json:"message" yaml:"message"`
	Trailers []string               `json:"trailers,omitempty" yaml:"trailers,omitempty"`
	Error    string                 `json:"error,omitempty" yaml:"error,omitempty"`
}

func (o *DebugOutput) GetError() error {
	return nil
}

func (o *DebugOutput) SetError(_ error) {}

func cmdDebug() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "debug [flags]",
		Aliases: []string{"info"},
		Short:   "Dump contextual information to aid debugging.",
		Args:    cobra.NoArgs,
		RunE:    runDebugCmd,
	}

	flags := cmd.Flags()

	addBranchFlag(flags)
	addCommitMessageFlags(flags)

	flags.SetNormalizeFunc(normalizeFlags)

	return cmd
}

func runDebugCmd(cmd *cobra.Command, args []string) error {
	output := &DebugOutput{
		HasToken: len(viper.GetString("token")) > 0,
		Trailers: util.BuildTrailers(),
		Remote: remote.Repo{
			Owner: viper.GetString("owner"),
			Name:  viper.GetString("repo"),
		},
		Branch:  viper.GetString("branch"),
		Commit:  localRepo.HeadCommit(),
		Message: remote.CommitMessage(util.BuildCommitMessage()),
	}

	status, err := localRepo.Status()
	if err != nil {
		output.SetError(fmt.Errorf("local repository status: %w", err))
	} else {
		output.Clean = status.IsClean()
	}

	return cmdOutput(cmd, output)
}
