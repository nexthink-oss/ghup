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
	contentCmd.Flags().StringP("separator", "s", ":", "file-spec separator")
	viper.BindPFlag("separator", contentCmd.Flags().Lookup("separator"))

	contentCmd.Flags().StringSliceP("update", "u", []string{}, "file-spec to update")
	viper.BindPFlag("update", contentCmd.Flags().Lookup("update"))

	contentCmd.Flags().StringSliceP("delete", "d", []string{}, "file-path to delete")
	viper.BindPFlag("delete", contentCmd.Flags().Lookup("delete"))

	rootCmd.AddCommand(contentCmd)
}

func runContentCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()

	client, err := remote.NewTokenClient(ctx, viper.GetString("token"))
	if err != nil {
		return err
	}

	separator := viper.GetString("separator")
	if len(separator) < 1 {
		return fmt.Errorf("invalid separator")
	}

	updateFiles := append(args, viper.GetStringSlice("update")...)
	deleteFiles := viper.GetStringSlice("delete")

	additions := []githubv4.FileAddition{}
	deletions := []githubv4.FileDeletion{}

	for _, arg := range updateFiles {
		target, content, err := local.GetLocalFileContent(arg, separator)
		if err != nil {
			return err
		}
		local_hash := plumbing.ComputeHash(plumbing.BlobObject, content).String()
		remote_hash := client.GetFileHashV4(owner, repo, branch, target)
		log.Debugf("local: %s, remote: %s", local_hash, remote_hash)
		if local_hash != remote_hash || force {
			log.Debugf("%q queued for addition", target)
			additions = append(additions, githubv4.FileAddition{
				Path:     githubv4.String(target),
				Contents: githubv4.Base64String(base64.StdEncoding.EncodeToString(content)),
			})
		} else {
			log.Debugf("%q (%s) on target branch: skipping addition", target, remote_hash)
		}
	}

	for _, target := range deleteFiles {
		remote_hash := client.GetFileHashV4(owner, repo, branch, target)
		if remote_hash != "" || force {
			log.Debugf("%q queued for deletion", target)
			deletions = append(deletions, githubv4.FileDeletion{
				Path: githubv4.String(target),
			})
		} else {
			log.Debugf("%q absent on target branch: skipping deletion", target)
		}
	}

	if len(additions) == 0 && len(deletions) == 0 {
		log.Info("nothing to do")
		return nil
	}

	changes := githubv4.FileChanges{
		Additions: &additions,
		Deletions: &deletions,
	}

	headOid, err := client.GetHeadOidV4(owner, repo, branch)
	if err != nil {
		return err
	}

	message = util.BuildCommitMessage()

	input := githubv4.CreateCommitOnBranchInput{
		Branch:          remote.CommittableBranch(owner, repo, branch),
		Message:         remote.CommitMessage(message),
		ExpectedHeadOid: headOid,
		FileChanges:     &changes,
	}

	_, commitUrl, err := client.CommitOnBranchV4(input)
	if err != nil {
		return err
	}

	fmt.Println(commitUrl)
	return
}
