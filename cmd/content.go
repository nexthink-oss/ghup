package cmd

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/apex/log"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/nexthink-oss/ghup/internal/local"
	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var contentCmd = &cobra.Command{
	Use:     "content [flags] [<file-spec> ...]",
	Short:   "Manage content via the GitHub V4 API",
	Args:    cobra.ArbitraryArgs,
	PreRunE: validateFlags,
	RunE:    runContentCmd,
}

func init() {
	contentCmd.PersistentFlags().Bool("create-branch", true, "create missing target branch")
	viper.BindPFlag("create-branch", contentCmd.PersistentFlags().Lookup("create-branch"))
	viper.BindEnv("create-branch", "GHUP_CREATE_BRANCH")

	contentCmd.PersistentFlags().String("pr-title", "", "create pull request iff target branch is created and title is specified")
	viper.BindPFlag("pr-title", contentCmd.PersistentFlags().Lookup("pr-title"))
	viper.BindEnv("pr-title", "GHUP_PR_TITLE")

	contentCmd.PersistentFlags().Bool("pr-draft", false, "create pull request in draft mode")
	viper.BindPFlag("pr-draft", contentCmd.PersistentFlags().Lookup("pr-draft"))
	viper.BindEnv("pr-draft", "GHUP_PR_DRAFT")

	contentCmd.PersistentFlags().String("base-branch", "", `base branch name (default: "[remote-default-branch])"`)
	viper.BindPFlag("base-branch", contentCmd.PersistentFlags().Lookup("base-branch"))
	viper.BindEnv("base-branch", "GHUP_BASE_BRANCH")

	contentCmd.Flags().StringP("separator", "s", ":", "file-spec separator")
	viper.BindPFlag("separator", contentCmd.Flags().Lookup("separator"))

	contentCmd.Flags().StringSliceP("update", "u", []string{}, "file-spec to update")
	viper.BindPFlag("update", contentCmd.Flags().Lookup("update"))

	contentCmd.Flags().StringSliceP("delete", "d", []string{}, "file-path to delete")
	viper.BindPFlag("delete", contentCmd.Flags().Lookup("delete"))

	contentCmd.Flags().SortFlags = false

	rootCmd.AddCommand(contentCmd)
}

func runContentCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()

	client, err := remote.NewTokenClient(ctx, viper.GetString("token"))
	if err != nil {
		return errors.Wrap(err, "NewTokenClient")
	}

	separator := viper.GetString("separator")
	if len(separator) < 1 {
		return fmt.Errorf("invalid separator")
	}

	repoInfo, err := client.GetRepositoryInfo(owner, repo, branch)
	if err != nil {
		return errors.Wrapf(err, "GetRepositoryInfo(%s, %s, %s)", owner, repo, branch)
	}

	if repoInfo.IsEmpty {
		return fmt.Errorf("cannot push to empty repository")
	}

	targetOid := repoInfo.TargetBranch.Commit
	baseBranch := viper.GetString("base-branch")
	newBranch := false

	if targetOid == "" {
		if !viper.GetBool("create-branch") {
			return fmt.Errorf("target branch %q does not exist", branch)
		}
		log.Infof("creating target branch %q", branch)
		if baseBranch == "" {
			baseBranch = repoInfo.DefaultBranch.Name
			targetOid = repoInfo.DefaultBranch.Commit
			log.Infof("defaulting base branch to %q", baseBranch)
		} else {
			targetOid, err = client.GetRefOidV4(owner, repo, baseBranch)
			if err != nil {
				return errors.Wrapf(err, "GetRefOidV4(%s, %s, %s)", owner, repo, baseBranch)
			}
		}

		createRefInput := githubv4.CreateRefInput{
			RepositoryID: repoInfo.NodeID,
			Name:         githubv4.String(fmt.Sprintf("refs/heads/%s", branch)),
			Oid:          targetOid,
		}
		log.Debugf("CreateRefInput: %+v", createRefInput)
		if err := client.CreateRefV4(createRefInput); err != nil {
			return errors.Wrap(err, "CreateRefV4")
		}
		newBranch = true
	}

	updateFiles := append(args, viper.GetStringSlice("update")...)
	deleteFiles := viper.GetStringSlice("delete")

	additions := []githubv4.FileAddition{}
	deletions := []githubv4.FileDeletion{}

	for _, arg := range updateFiles {
		target, content, err := local.GetLocalFileContent(arg, separator)
		if err != nil {
			return errors.Wrapf(err, "GetLocalFileContent(%s, %s)", arg, separator)
		}
		local_hash := plumbing.ComputeHash(plumbing.BlobObject, content).String()
		remote_hash := client.GetFileHashV4(owner, repo, branch, target)
		log.Infof("local: %s, remote: %s", local_hash, remote_hash)
		if local_hash != remote_hash || force {
			log.Infof("%q queued for addition", target)
			additions = append(additions, githubv4.FileAddition{
				Path:     githubv4.String(target),
				Contents: githubv4.Base64String(base64.StdEncoding.EncodeToString(content)),
			})
		} else {
			log.Infof("%q (%s) on target branch: skipping addition", target, remote_hash)
		}
	}

	for _, target := range deleteFiles {
		remote_hash := client.GetFileHashV4(owner, repo, branch, target)
		if remote_hash != "" || force {
			log.Infof("%q queued for deletion", target)
			deletions = append(deletions, githubv4.FileDeletion{
				Path: githubv4.String(target),
			})
		} else {
			log.Infof("%q absent on target branch: skipping deletion", target)
		}
	}

	if len(additions) == 0 && len(deletions) == 0 {
		log.Warn("nothing to do")
		return nil
	}

	changes := githubv4.FileChanges{
		Additions: &additions,
		Deletions: &deletions,
	}
	log.Debugf("Additions: %+v", additions)
	log.Debugf("Deletions: %+v", deletions)

	message = util.BuildCommitMessage()

	input := githubv4.CreateCommitOnBranchInput{
		Branch:          remote.CommittableBranch(owner, repo, branch),
		Message:         remote.CommitMessage(message),
		ExpectedHeadOid: targetOid,
		FileChanges:     &changes,
	}
	log.Debugf("CreateCommitOnBranchInput: %+v", input)

	_, commitUrl, err := client.CommitOnBranchV4(input)
	if err != nil {
		return errors.Wrap(err, "CommitOnBranchV4")
	}

	if title := viper.GetString("pr-title"); newBranch && title != "" {
		log.Infof("opening pull request from %q to %q", branch, baseBranch)
		input := githubv4.CreatePullRequestInput{
			RepositoryID: repoInfo.NodeID,
			BaseRefName:  githubv4.String(baseBranch),
			Draft:        githubv4.NewBoolean(githubv4.Boolean(viper.GetBool("pr-draft"))),
			HeadRefName:  githubv4.String(branch),
			Title:        githubv4.String(title),
		}
		log.Debugf("CreatePullRequestInput: %+v", input)
		pullRequestUrl, err := client.CreatePullRequestV4(input)
		if err != nil {
			return errors.Wrap(err, "CreatePullRequestV4")
		}
		fmt.Println(pullRequestUrl)
	} else {
		fmt.Println(commitUrl)
	}
	return
}
