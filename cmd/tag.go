package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/google/go-github/v69/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
)

type tagReport struct {
	Name        string `json:"name"`
	SHA         string `json:"sha"`
	ObjectSHA   string `json:"target,omitempty"`
	Lightweight bool   `json:"lightweight"`
	URL         string `json:"url"`
}

var tagCmd = &cobra.Command{
	Use:   "tag [flags] [<name>]",
	Short: "Create or update lightweight or annotated tags.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTagCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := tagCmd.Flags()
	flags.String("tag", "", "tag `name`")
	flags.BoolP("lightweight", "l", false, "force lightweight tag")
	flags.StringP("commitish", "c", localRepo.Branch, "target `commitish`")
	addBranchFlag(flags)
	flags.MarkDeprecated("branch", "pass commitish via -c/--commitish instead")
	flags.MarkHidden("branch")
	addCommitMessageFlags(flags)
	addForceFlag(flags)

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	rootCmd.AddCommand(tagCmd)
}

func runTagCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

	tagName := viper.GetString("tag")

	if len(args) == 1 {
		tagName = args[0]
	}

	if tagName == "" {
		return fmt.Errorf("tag is required")
	}

	repo := remote.Repo{
		Owner: repoOwner,
		Name:  repoName,
	}
	force := viper.GetBool("force")

	report := tagReport{
		Name: tagName,
	}

	client, err := remote.NewClient(ctx, repo, githubToken)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	repoInfo, err := client.GetRepositoryInfo("")
	if err != nil {
		return fmt.Errorf("GetRepositoryInfo(%s): %w", repo, err)
	}

	if repoInfo.IsEmpty {
		return errors.New("cannot tag empty repository")
	}

	commitish := viper.GetString("commitish")
	var commitSha string
	if commitish != "" {
		commitSha, err = client.ResolveCommitish(commitish)
		if err != nil {
			return fmt.Errorf("ResolveCommitish(%s, %s): %w", repo, commitish, err)
		}
	} else {
		commitish = repoInfo.DefaultBranch.Name
		commitSha = string(repoInfo.DefaultBranch.Commit)
	}

	report.SHA = commitSha

	tagRefName, err := util.QualifiedRefName(tagName, "tags")
	if err != nil {
		return fmt.Errorf("Invalid tag reference: %s: %w", tagRefName, err)
	}

	log.Infof("checking tag reference: %s", tagRefName)
	existingTagRef, resp, err := client.V3.Git.GetRef(ctx, repo.Owner, repo.Name, tagRefName)
	if err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("GetRef(%s, %s): %w", repo, tagRefName, err)
		}
	} else {
		report.URL = existingTagRef.GetURL()
		existingSha := existingTagRef.GetObject().GetSHA()
		if commitSha == existingSha {
			// matching tag already exists
			commandOutput = report
			return nil
		} else if !force {
			// tag exists but points to a different commit
			return fmt.Errorf("tag '%s' already exists: %s", tagName, existingSha)
		}
	}

	if !viper.GetBool("lightweight") {
		message := util.BuildCommitMessage()
		tag, err := client.CreateAnnotationTag(tagName, message, commitSha)
		if err != nil {
			return fmt.Errorf("CreateAnnotationTag(%s, %s): %w", repo, tagName, err)
		}

		commitSha = tag.GetSHA()
		report.ObjectSHA = commitSha
	}

	tagRef := &github.Reference{
		Ref:    &tagRefName,
		Object: &github.GitObject{SHA: github.Ptr(commitSha)},
	}

	if err := client.CreateOrUpdateRef(existingTagRef, tagRef, true); err != nil {
		return fmt.Errorf("CreateOrUpdateRef(%s, %s): %w", repo, tagRefName, err)
	}

	if existingTagRef != nil {
		report.URL = existingTagRef.GetURL()
	} else {
		report.URL = tagRef.GetURL()
	}

	commandOutput = report

	return nil
}
