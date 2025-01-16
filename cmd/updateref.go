package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/apex/log"
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
	OldSHA  string `json:"old_sha,omitempty"`
	SHA     string `json:"sha,omitempty"`
	Error   error  `json:"error,omitempty"`
}

type report struct {
	Source sRef   `json:"source"`
	Target []tRef `json:"target"`
}

func (r report) String() string {
	m, err := json.Marshal(r)
	if err != nil {
		log.Error(fmt.Sprintf("json.Marshal: %s", err))
		return ""
	}
	return string(m)
}

var updateRefCmd = &cobra.Command{
	Use:     "update-ref [flags] -s <source> <target> ...",
	Short:   "Update target refs to match source",
	PreRunE: validateFlags,
	RunE:    runUpdateRefCmd,
}

func init() {
	updateRefCmd.Flags().StringP("source", "s", "", "source `ref-or-commit`")
	viper.BindPFlag("source", updateRefCmd.Flags().Lookup("source"))

	viper.BindEnv("targets", "GHUP_TARGETS")

	refTypes := []string{"heads", "tags"}

	defaultSourceType := choiceflag.NewChoiceFlag(refTypes)
	_ = defaultSourceType.Set("heads")
	updateRefCmd.Flags().VarP(defaultSourceType, "source-type", "S", "unqualified source ref type")
	viper.BindPFlag("source-type", updateRefCmd.Flags().Lookup("source-type"))

	defaultTargetType := choiceflag.NewChoiceFlag(refTypes)
	_ = defaultTargetType.Set("tags")
	updateRefCmd.Flags().VarP(defaultTargetType, "target-type", "T", "unqualified target ref type")
	viper.BindPFlag("target-type", updateRefCmd.Flags().Lookup("target-type"))

	updateRefCmd.Flags().SortFlags = false

	rootCmd.AddCommand(updateRefCmd)
}

func runUpdateRefCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()

	client, err := remote.NewTokenClient(ctx, viper.GetString("token"))
	if err != nil {
		return fmt.Errorf("NewTokenClient: %w", err)
	}

	sourceRefName := viper.GetString("source")
	if sourceRefName == "" {
		return errors.New("no source ref specified")
	}

	var sourceObject string

	if util.IsCommitHash(sourceRefName) {
		sourceCommit, _, err := client.GetCommitSHA(ctx, owner, repo, sourceRefName)
		if err != nil {
			return fmt.Errorf("GetCommitSHA(%s, %s, %s): %w", owner, repo, sourceRefName, err)
		}
		sourceObject = *sourceCommit
	} else {
		sourceRefName, err = util.NormalizeRefName(sourceRefName, viper.GetString("source-type"))
		if err != nil {
			return fmt.Errorf("NormalizeRefName(%s, %s): %w", sourceRefName, viper.GetString("source-type"), err)
		}

		log.Infof("resolving source ref: %s", sourceRefName)
		sourceRef, _, err := client.V3.Git.GetRef(ctx, owner, repo, sourceRefName)
		if err != nil {
			return fmt.Errorf("GetSourceRef(%s, %s, %s): %w", owner, repo, sourceRefName, err)
		}

		sourceObject = sourceRef.Object.GetSHA()
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
	for i, targetRefName := range targetRefNames {
		targetRefName, err = util.NormalizeRefName(targetRefName, viper.GetString("target-type"))
		if err != nil {
			return fmt.Errorf("NormalizeRefName(%s, %s): %w", targetRefName, viper.GetString("target-type"), err)
		}

		targetRefNames[i] = targetRefName
	}

	report := report{
		Source: sRef{
			Ref: sourceRefName,
			SHA: sourceObject,
		},
		Target: make([]tRef, 0, len(targetRefNames)),
	}

	var returnError error = nil

	for _, targetRefName := range targetRefNames {
		targetReport := tRef{
			Ref: targetRefName,
		}

		targetRef := &github.Reference{
			Ref: &targetRefName,
			Object: &github.GitObject{
				SHA: github.Ptr(sourceObject),
			},
		}

		oldHash, newHash, err := client.UpdateRefName(ctx, owner, repo, targetRefName, targetRef, viper.GetBool("force"))
		if err != nil {
			returnError = fmt.Errorf("Error(s) Detected")
			targetReport.Error = fmt.Errorf("UpdateRefName: %w", err)
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

	fmt.Print(report)

	return returnError
}
