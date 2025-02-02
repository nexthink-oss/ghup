package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
)

type resolveReport struct {
	Repository string   `json:"repository" yaml:"repository"`
	Commitish  string   `json:"commitish" yaml:"commitish"`
	SHA        string   `json:"sha" yaml:"sha"`
	Branches   []string `json:"branches,omitempty" yaml:"branches,omitempty"`
	Tags       []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

var resolveCmd = &cobra.Command{
	Use:   "resolve [<commit-ish>]",
	Short: "Resolve commit-ish to SHA",
	Long:  `Resolve a commit-ish to a SHA, optionally finding matching branches and/or tags.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runResolveCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := resolveCmd.Flags()

	flags.String("commitish", "HEAD", "commitish to match")
	flags.MarkHidden("commitish")
	flags.BoolP("branches", "b", false, "list matching branches/heads")
	flags.BoolP("tags", "t", false, "list matching tags")

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	rootCmd.AddCommand(resolveCmd)
}

func runResolveCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

	repo := remote.Repo{
		Owner: repoOwner,
		Name:  repoName,
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

	client, err := remote.NewClient(ctx, repo, githubToken)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	sha, err := client.ResolveCommitish(commitish)
	if err != nil {
		return fmt.Errorf("ResolveCommitish(%q): %w", commitish, err)
	}

	if sha == "" {
		return fmt.Errorf("failed to resolve %q", commitish)
	}

	report := resolveReport{
		Repository: repo.String(),
		Commitish:  commitish,
		SHA:        sha,
	}

	if viper.GetBool("branches") {
		heads, err := client.GetMatchingHeads(sha)
		if err != nil {
			return fmt.Errorf("GetMatchingHeads: %w", err)
		}
		report.Branches = heads
	}

	if viper.GetBool("tags") {
		tags, err := client.GetMatchingTags(sha)
		if err != nil {
			return fmt.Errorf("GetMatchingTags: %w", err)
		}
		report.Tags = tags
	}

	commandOutput = report

	return nil
}
