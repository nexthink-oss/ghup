package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/google/go-github/v68/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/nexthink-oss/ghup/pkg/choiceflag"
)

type sRef struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

type tRef struct {
	Ref     string `json:"ref"`
	Updated bool   `json:"updated"`
	OldSHA  string `json:"old,omitempty"`
	SHA     string `json:"sha,omitempty"`
	Error   string `json:"error,omitempty"`
}

type updateRefReport struct {
	Source sRef   `json:"source"`
	Target []tRef `json:"target"`
}

var updateRefCmd = &cobra.Command{
	Use:   "update-ref [flags] -s <source> <target> ...",
	Short: "Update target refs to match source",
	RunE:  runUpdateRefCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := updateRefCmd.Flags()

	flags.StringP("source", "s", "", "source `ref-or-commit`")

	refTypes := []string{"heads", "tags"}

	defaultSourceType := choiceflag.NewChoiceFlag(refTypes)
	_ = defaultSourceType.Set("heads")
	flags.VarP(defaultSourceType, "source-type", "S", "source ref type")

	defaultTargetType := choiceflag.NewChoiceFlag(refTypes)
	_ = defaultTargetType.Set("tags")
	flags.VarP(defaultTargetType, "target-type", "T", "target ref type")

	addForceFlag(flags)

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	rootCmd.AddCommand(updateRefCmd)
}

func runUpdateRefCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

	sourceRef := viper.GetString("source")
	if sourceRef == "" {
		return errors.New("no source ref specified")
	}

	repo := remote.Repo{
		Owner: viper.GetString("owner"),
		Name:  viper.GetString("repo"),
	}
	force := viper.GetBool("force")

	client, err := remote.NewClient(ctx, repo, token)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	var sourceSHA string

	if util.IsCommitHash(sourceRef) {
		sourceSHA, err = client.GetCommitSHA(sourceRef)
		if err != nil {
			return fmt.Errorf("GetCommitSHA(%s, %s): %w", repo, sourceRef, err)
		}
	} else {
		sourceSHA, err = client.GetRefSHA(sourceRef, viper.GetString("source-type"))
		if err != nil {
			return fmt.Errorf("GetSourceRef(%s, %s): %w", repo, sourceRef, err)
		}
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
		targetRefName, err = util.NormalizeRefName(targetRefName, defaultTargetType)
		if err != nil {
			return fmt.Errorf("NormalizeRefName(%s, %s): %w", targetRefName, defaultTargetType, err)
		}

		targetRefNames[i] = targetRefName
	}

	report := updateRefReport{
		Source: sRef{
			Ref: sourceRef,
			SHA: sourceSHA,
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
				SHA: github.Ptr(sourceSHA),
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

	err = util.FprintJSON(os.Stdout, report)
	if err != nil {
		updateRefErrors = append(updateRefErrors, fmt.Errorf("PrintJson: %w", err))
	}

	return errors.Join(updateRefErrors...)
}
