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
	Commitish string `json:"commitish" yaml:"commitish"`
	SHA       string `json:"sha,omitempty" yaml:"sha,omitempty"`
	Error     string `json:"error,omitempty" yaml:"error,omitempty"`
}

type targetRef struct {
	Ref     string `json:"ref" yaml:"ref"`
	OldSHA  string `json:"old,omitempty" yaml:"old,omitempty"`
	SHA     string `json:"sha,omitempty" yaml:"sha,omitempty"`
	Updated bool   `json:"updated" yaml:"updated"`
	Error   string `json:"error,omitempty" yaml:"error,omitempty"`
}

type UpdateRefOutput struct {
	Repository string      `json:"repository,omitempty" yaml:"repository,omitempty"`
	Source     source      `json:"source" yaml:"source"`
	Target     []targetRef `json:"target,omitempty" yaml:"target,omitempty"`
	Error      error       `json:"-" yaml:"-"`
}

func (o *UpdateRefOutput) GetError() error {
	return o.Error
}

func (o *UpdateRefOutput) SetError(err error) {
	o.Error = err
}

func cmdUpdateRef() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-ref [flags] -s <source-commitish> <target-ref> ...",
		Short: "Update target refs to match source commitish.",
		Long: `Update target refs to match source commitish.
Source commitish may also be passed via the GHUP_SOURCE environment variable,
and target refs via GHUP_TARGETS (space-delimited).`,
		RunE: runUpdateRefCmd,
	}

	flags := cmd.Flags()

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

	return cmd
}

func runUpdateRefCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

	commitish := viper.GetString("source")
	if commitish == "" {
		return errors.New("no source ref specified")
	}

	repo := remote.Repo{
		Owner: viper.GetString("owner"),
		Name:  viper.GetString("repo"),
	}
	force := viper.GetBool("force")

	client, err := remote.NewClient(ctx, &repo)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	output := &UpdateRefOutput{
		Repository: repo.String(),
		Source: source{
			Commitish: commitish,
		},
	}

	commitSha, err := client.ResolveCommitish(commitish)
	if err != nil {
		err = fmt.Errorf("resolving commitish: %w", err)
		output.SetError(err)
		output.Source.Error = err.Error()
		return cmdOutput(cmd, output)
	}
	if commitSha == "" {
		err = fmt.Errorf("source commitish does not exist")
		output.SetError(err)
		output.Source.Error = err.Error()
		return cmdOutput(cmd, output)
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

	output.Source.SHA = commitSha
	output.Target = make([]targetRef, 0, len(targetRefNames))

	var updateRefErrors = make([]error, 0)

	for _, targetRefName := range targetRefNames {
		targetReport := targetRef{
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
			output.Target = append(output.Target, targetReport)
			continue
		}
		targetReport.SHA = newHash
		if oldHash != newHash {
			targetReport.OldSHA = oldHash
			targetReport.Updated = true
		}
		output.Target = append(output.Target, targetReport)
	}

	if len(updateRefErrors) > 0 {
		output.SetError(fmt.Errorf("updating refs: %w", errors.Join(updateRefErrors...)))
	}

	return cmdOutput(cmd, output)
}
