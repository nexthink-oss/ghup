package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
)

type ResolveOutput struct {
	Repository   string   `json:"repository" yaml:"repository"`
	Commitish    string   `json:"commitish" yaml:"commitish"`
	SHA          string   `json:"sha,omitempty" yaml:"sha,omitempty"`
	Branches     []string `json:"branches,omitempty" yaml:"branches,omitempty"`
	Tags         []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Error        error    `json:"-" yaml:"-"`
	ErrorMessage string   `json:"error,omitempty" yaml:"error,omitempty"`
}

func (o *ResolveOutput) GetError() error {
	return o.Error
}

func (o *ResolveOutput) SetError(err error) {
	o.Error = err
	if err != nil {
		o.ErrorMessage = err.Error()
	}
}

func cmdResolve() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "resolve [<commit-ish>]",
		Short: "Resolve commit-ish to SHA",
		Long:  `Resolve a commit-ish to a SHA, optionally finding matching branches and/or tags.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runResolveCmd,
	}

	flags := cmd.Flags()

	flags.String("commitish", "HEAD", "commitish to match")
	flags.MarkHidden("commitish")
	flags.BoolP("branches", "b", false, "list matching branches/heads")
	flags.BoolP("tags", "t", false, "list matching tags")

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	return cmd
}

func runResolveCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

	repo := remote.Repo{
		Owner: viper.GetString("owner"),
		Name:  viper.GetString("repo"),
	}

	var commitish string
	if len(args) == 1 {
		commitish = args[0]
	} else {
		commitish = viper.GetString("commitish")
	}

	if commitish == "" {
		return fmt.Errorf("commitish is required")
	}

	client, err := remote.NewClient(ctx, &repo)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	sha, err := client.ResolveCommitish(commitish)
	if err != nil {
		return fmt.Errorf("ResolveCommitish(%q): %w", commitish, err)
	}

	output := &ResolveOutput{
		Repository: repo.String(),
		Commitish:  commitish,
		SHA:        sha,
	}

	if sha == "" {
		output.SetError(errors.New("commitish does not exist"))
		return cmdOutput(cmd, output)
	}

	errs := make([]error, 0)

	if viper.GetBool("branches") {
		heads, err := client.GetMatchingHeads(sha)
		if err != nil {
			errs = append(errs, fmt.Errorf("finding matching branches: %w", err))
		} else {
			output.Branches = heads
		}
	}

	if viper.GetBool("tags") {
		tags, err := client.GetMatchingTags(sha)
		if err != nil {
			errs = append(errs, fmt.Errorf("finding matching tags: %w", err))
		} else {
			output.Tags = tags
		}
	}

	if len(errs) > 0 {
		output.SetError(errors.Join(errs...))
	}

	return cmdOutput(cmd, output)
}
