package cmd

import (
	"errors"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/nexthink-oss/ghup/pkg/choiceflag"
)

type source struct {
	Commitish string `json:"commitish"`
	SHA       string `json:"sha"`
}

type tRef struct {
	Ref     string `json:"ref"`
	Error   string `json:"error,omitempty"`
	OldSHA  string `json:"old,omitempty"`
	SHA     string `json:"sha,omitempty"`
	Updated bool   `json:"updated"`
}

type UpdateRefReport struct {
	Repository string `json:"repository,omitempty"`
	Source     source `json:"source"`
	Target     []tRef `json:"target"`
}

var updateRefCmd = &cobra.Command{
	Use:   "update-ref [flags] -s <source-commitish> <target-ref> ...",
	Short: "Update target refs to match source commitish.",
	Long: `Update target refs to match source commitish.
Source commitish may also be passed via the GHUP_SOURCE environment variable,
and target refs via GHUP_TARGETS (space-delimited).`,
	RunE: runUpdateRefCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := updateRefCmd.Flags()

	flags.StringP("source", "s", "", "source `commitish`")

	refTypes := []string{"heads", "tags"}

	defaultSourceType := choiceflag.NewChoiceFlag(refTypes)
	_ = defaultSourceType.Set("heads")
	flags.VarP(defaultSourceType, "source-type", "S", "source ref type")
	flags.MarkDeprecated("source-type", "prefer use of qualified commitish source")
	flags.MarkHidden("source-type")

	defaultTargetType := choiceflag.NewChoiceFlag(refTypes)
	_ = defaultTargetType.Set("tags")
	flags.VarP(defaultTargetType, "target-type", "T", "type for unqualified target ref")
	flags.MarkDeprecated("target-type", "prefer fully-qualified target refs")
	flags.MarkHidden("target-type")

	addForceFlag(flags)

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	rootCmd.AddCommand(updateRefCmd)
}

func runUpdateRefCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

	commitish := viper.GetString("source")
	if commitish == "" {
		return errors.New("no source ref specified")
	}

	repo := remote.Repo{
		Owner: repoOwner,
		Name:  repoName,
	}
	force := viper.GetBool("force")

	client, err := remote.NewClient(ctx, repo, githubToken)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	commitSha, err := client.ResolveCommitish(commitish)
	if err != nil {
		return fmt.Errorf("resolving commitish %q: %w", commitish, err)
	}
	if commitSha == "" {
		return fmt.Errorf("failed to resolve %q", commitish)
	}

	var targetRefNames []string
	if len(args) > 0 {
		targetRefNames = args
	} else {
		targetRefNames = viper.GetStringSlice("targets")
	}

	if len(targetRefNames) == 0 {
		return errors.New("no target refs specified")
	}

	// ensure all target refs are properly qualified
	defaultTargetType := viper.GetString("target-type")
	for i, targetRefName := range targetRefNames {
		targetRefName, err = util.QualifiedRefName(targetRefName, defaultTargetType)
		if err != nil {
			return fmt.Errorf("QualifiedRefName(%s, %s): %w", targetRefName, defaultTargetType, err)
		}

		targetRefNames[i] = targetRefName
	}

	report := UpdateRefReport{
		Repository: repo.String(),
		Source: source{
			Commitish: commitish,
			SHA:       commitSha,
		},
		Target: make([]tRef, 0, len(targetRefNames)),
	}

	var updateRefErrors = make([]error, 0)

	for _, targetRefName := range targetRefNames {
		targetReport := tRef{
			Ref: targetRefName,
		}

		targetRef := &github.Reference{
			Ref: &targetRefName,
			Object: &github.GitObject{
				SHA: github.Ptr(commitSha),
			},
		}

		oldHash, newHash, err := client.UpdateRefName(targetRefName, targetRef, force)
		if err != nil {
			updateRefErrors = append(updateRefErrors, fmt.Errorf("%s: %w", targetRefName, err))
			targetReport.Error = err.Error()
			report.Target = append(report.Target, targetReport)
			continue
		}
		targetReport.SHA = newHash
		if oldHash != newHash {
			targetReport.OldSHA = oldHash
			targetReport.Updated = true
		}
		report.Target = append(report.Target, targetReport)
	}

	commandOutput = report

	return errors.Join(updateRefErrors...)
}
