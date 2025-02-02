package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
)

type matchingReport struct {
	SHA   string   `json:"sha" yaml:"sha"`
	Heads []string `json:"heads,omitempty" yaml:"heads,omitempty"`
	Tags  []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

var matchingCmd = &cobra.Command{
	Use:   "matching [<name>]",
	Short: "List refs matching name",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runMatchingCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := matchingCmd.Flags()

	flags.String("ref", "", "ref `name` or commit to match")
	flags.Bool("heads", true, "list matching heads/branches")
	flags.Bool("tags", true, "list matching tags")

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	rootCmd.AddCommand(matchingCmd)
}

func runMatchingCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

	repo := remote.Repo{
		Owner: viper.GetString("owner"),
		Name:  viper.GetString("repo"),
	}

	refName := viper.GetString("ref")

	if len(args) == 1 {
		refName = args[0]
	}

	if refName == "" {
		return fmt.Errorf("ref name is required")
	}

	client, err := remote.NewClient(ctx, repo, token)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	sha, err := client.GetSHA(refName, "heads")
	if err != nil {
		return fmt.Errorf("GetSHA: %w", err)
	}

	report := matchingReport{
		SHA: sha,
	}

	if viper.GetBool("heads") {
		heads, err := client.GetMatchingHeads(sha)
		if err != nil {
			return fmt.Errorf("GetMatchingHeads: %w", err)
		}
		report.Heads = heads
	}

	if viper.GetBool("tags") {
		tags, err := client.GetMatchingTags(sha)
		if err != nil {
			return fmt.Errorf("GetMatchingTags: %w", err)
		}
		report.Tags = tags
	}

	return util.FprintJSON(os.Stdout, report)
}
