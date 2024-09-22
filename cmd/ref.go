package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/google/go-github/v64/github"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/nexthink-oss/ghup/pkg/choiceflag"
)

type sRef struct {
	Ref string `yaml:"ref"`
	SHA string `yaml:"sha"`
}

type tRef struct {
	Ref     string `yaml:"ref"`
	Updated bool   `yaml:"updated"`
	OldSHA  string `yaml:"old_sha,omitempty"`
	SHA     string `yaml:"sha,omitempty"`
	Error   string `yaml:"error,omitempty"`
}

type report struct {
	Source sRef   `yaml:"source"`
	Target []tRef `yaml:"target"`
}

func (r report) String() string {
	return util.EncodeYAML(&r)
}

var refCmd = &cobra.Command{
	Use:     "ref [flags] <source_ref> <target_ref>...",
	Short:   "Update target_refs to match source_ref",
	Args:    cobra.MinimumNArgs(2),
	PreRunE: validateFlags,
	RunE:    runRefCmd,
}

func init() {
	defaultSourceTypes := []string{"heads", "tags"}
	defaultSourceTypeFlag := choiceflag.NewChoiceFlag(defaultSourceTypes)
	refCmd.Flags().VarP(defaultSourceTypeFlag, "source-type", "S", fmt.Sprintf("default source ref type (choices: [%s])", strings.Join(defaultSourceTypes, ", ")))
	refCmd.Flags().Lookup("source-type").DefValue = defaultSourceTypes[0]
	viper.BindPFlag("source-type", refCmd.Flags().Lookup("source-type"))

	defaultTargetTypes := []string{"tags", "heads"}
	defaultTargetTypeFlag := choiceflag.NewChoiceFlag(defaultTargetTypes)
	refCmd.Flags().VarP(defaultTargetTypeFlag, "target-type", "T", fmt.Sprintf("default target ref type (choices: [%s])", strings.Join(defaultTargetTypes, ", ")))
	refCmd.Flags().Lookup("target-type").DefValue = defaultTargetTypes[0]
	viper.BindPFlag("target-type", refCmd.Flags().Lookup("target-type"))

	// TODO: disable branch flag
	// TODO: disable commit-message related flags

	refCmd.Flags().SortFlags = false

	rootCmd.AddCommand(refCmd)
}

func runRefCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()

	client, err := remote.NewTokenClient(ctx, viper.GetString("token"))
	if err != nil {
		return errors.Wrap(err, "NewTokenClient")
	}

	sourceRefName := strings.TrimPrefix(args[0], "refs/")
	var sourceRefObject string

	if sourceRefName == "" {
		return fmt.Errorf("no source ref specified")
	}

	if util.IsCommitHash(sourceRefName) {
		sourceCommit, _, err := client.GetCommitSHA(ctx, owner, repo, sourceRefName)
		if err != nil {
			return errors.Wrapf(err, "GetCommitSHA(%s, %s, %s)", owner, repo, sourceRefName)
		}
		sourceRefObject = *sourceCommit
	} else {
		if !(strings.HasPrefix(sourceRefName, "heads/") || strings.HasPrefix(sourceRefName, "tags/")) {
			sourceRefName = strings.Join([]string{viper.GetString("source-type"), sourceRefName}, "/")
		}

		log.Infof("resolving source ref: %s", sourceRefName)
		sourceRef, _, err := client.V3.Git.GetRef(ctx, owner, repo, sourceRefName)
		if err != nil {
			return errors.Wrapf(err, "GetSourceRef(%s, %s, %s)", owner, repo, sourceRefName)
		}

		sourceRefObject = sourceRef.Object.GetSHA()
	}

	targetRefNames := args[1:]

	// ensure all target refs are properly qualified
	for i, targetRefName := range targetRefNames {
		targetRefName = strings.TrimPrefix(targetRefName, "refs/")
		if !(strings.HasPrefix(targetRefName, "heads/") || strings.HasPrefix(targetRefName, "tags/")) {
			targetRefName = strings.Join([]string{viper.GetString("target-type"), targetRefName}, "/")
		}
		targetRefNames[i] = targetRefName
	}

	report := report{
		Source: sRef{
			Ref: sourceRefName,
			SHA: sourceRefObject,
		},
		Target: make([]tRef, 0, len(targetRefNames)),
	}

	for _, targetRefName := range targetRefNames {
		targetRef := &github.Reference{
			Ref: &targetRefName,
			Object: &github.GitObject{
				SHA: github.String(sourceRefObject),
			},
		}

		log.Infof("resolving target ref: %s", targetRefName)
		legacyRef, _, err := client.V3.Git.GetRef(ctx, owner, repo, targetRefName)
		if err == nil {
			log.Infof("updating target ref")
			updatedRef, _, err := client.V3.Git.UpdateRef(ctx, owner, repo, targetRef, viper.GetBool("force"))
			if err != nil {
				report.Target = append(report.Target, tRef{
					Ref:   targetRefName,
					Error: errors.Wrapf(err, "UpdateRef").Error(),
				})
				continue
			}
			if updatedRef.Object.GetSHA() == legacyRef.Object.GetSHA() {
				report.Target = append(report.Target, tRef{
					Ref: targetRefName,
					SHA: legacyRef.Object.GetSHA(),
				})
				continue
			}
			report.Target = append(report.Target, tRef{
				Ref:     targetRefName,
				OldSHA:  legacyRef.Object.GetSHA(),
				SHA:     updatedRef.Object.GetSHA(),
				Updated: true,
			})
		} else {
			log.Infof("creating target ref")
			createdRef, _, err := client.V3.Git.CreateRef(ctx, owner, repo, targetRef)
			if err != nil {
				report.Target = append(report.Target, tRef{
					Ref:   targetRefName,
					Error: errors.Wrapf(err, "CreateRef").Error(),
				})
				continue
			}
			report.Target = append(report.Target, tRef{
				Ref:     targetRefName,
				SHA:     createdRef.Object.GetSHA(),
				Updated: true,
			})
		}
	}

	fmt.Print(report)
	return
}
