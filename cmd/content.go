package cmd

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/local"
	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
)

var contentCmd = &cobra.Command{
	Use:   "content [flags] [<file-spec> ...]",
	Short: "Manage content via the GitHub V4 API",
	Args:  cobra.ArbitraryArgs,
	RunE:  runContentCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := contentCmd.Flags()

	flags.StringSliceP("update", "u", []string{}, "`file-spec` to update")
	flags.StringSliceP("delete", "d", []string{}, "`file-path` to delete")
	flags.StringP("separator", "s", ":", "file-spec separator")
	addCommitMessageFlags(flags)
	addBranchFlag(flags)
	flags.Bool("create-branch", true, "create missing target branch")
	flags.StringP("base-branch", "B", "", `base branch `+"`name`"+` (default: "[remote-default-branch])"`)
	addPullRequestFlags(flags)
	// addDryRunFlag(flags)
	addForceFlag(flags)

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	rootCmd.AddCommand(contentCmd)
}

func runContentCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

	separator := viper.GetString("separator")
	if len(separator) < 1 {
		return fmt.Errorf("invalid separator")
	}

	repo := remote.Repo{
		Owner: viper.GetString("owner"),
		Name:  viper.GetString("repo"),
	}
	branch := viper.GetString("branch")
	force := viper.GetBool("force")

	client, err := remote.NewClient(ctx, repo, token)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	repoInfo, err := client.GetRepositoryInfo(branch)
	if err != nil {
		return fmt.Errorf("GetRepositoryInfo(%s, %s): %w", repo, branch, err)
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
			targetOid, err = client.GetRefOidV4(baseBranch)
			if err != nil {
				return fmt.Errorf("GetRefOidV4(%s, %s): %w", repo, baseBranch, err)
			}
		}

		createRefInput := githubv4.CreateRefInput{
			RepositoryID: repoInfo.NodeID,
			Name:         githubv4.String(fmt.Sprintf("refs/heads/%s", branch)),
			Oid:          targetOid,
		}
		log.Debugf("CreateRefInput: %+v", createRefInput)
		if err := client.CreateRefV4(createRefInput); err != nil {
			return fmt.Errorf("CreateRefV4: %w", err)
		}
		newBranch = true
	}

	updateFiles := make(map[string]githubv4.FileAddition, 0)
	deleteFiles := make(map[string]githubv4.FileDeletion, 0)

	for spec := range util.SliceChain(viper.GetStringSlice("update"), args) {
		source, target, err := local.SplitUpdateSpec(spec, separator)
		if err != nil {
			return fmt.Errorf("GetLocalFileContent(%s, %s): %w", spec, separator, err)
		}
		content, err := os.ReadFile(source)
		local_hash := plumbing.ComputeHash(plumbing.BlobObject, content).String()
		remote_hash := client.GetFileHashV4(branch, target)
		log.Infof("local: %s, remote: %s", local_hash, remote_hash)
		if local_hash != remote_hash || force {
			log.Infof("%q queued for addition", target)
			updateFiles[target] = githubv4.FileAddition{
				Path:     githubv4.String(target),
				Contents: githubv4.Base64String(base64.StdEncoding.EncodeToString(content)),
			}
		} else {
			log.Infof("%q (%s) on target branch: skipping addition", target, remote_hash)
		}
	}

	for _, path := range viper.GetStringSlice("delete") {
		deleteFiles[path] = githubv4.FileDeletion{}
		remote_hash := client.GetFileHashV4(branch, path)
		if remote_hash != "" || force {
			log.Infof("%q queued for deletion", path)
			deleteFiles[path] = githubv4.FileDeletion{
				Path: githubv4.String(path),
			}
		} else {
			log.Infof("%q absent on target branch: skipping deletion", path)
		}

	}

	additions := util.MapValues(updateFiles)
	deletions := util.MapValues(deleteFiles)

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

	message := util.BuildCommitMessage()

	input := githubv4.CreateCommitOnBranchInput{
		Branch:          remote.CommittableBranch(repo, branch),
		Message:         remote.CommitMessage(message),
		ExpectedHeadOid: targetOid,
		FileChanges:     &changes,
	}
	log.Debugf("CreateCommitOnBranchInput: %+v", input)

	_, commitUrl, err := client.CreateCommitOnBranchV4(input)
	if err != nil {
		return fmt.Errorf("CommitOnBranchV4: %w", err)
	}

	if title := viper.GetString("pr-title"); newBranch && title != "" {
		body := githubv4.String(viper.GetString("pr-body"))
		log.Infof("opening pull request from %q to %q", branch, baseBranch)
		input := githubv4.CreatePullRequestInput{
			RepositoryID: repoInfo.NodeID,
			BaseRefName:  githubv4.String(baseBranch),
			Draft:        githubv4.NewBoolean(githubv4.Boolean(viper.GetBool("pr-draft"))),
			HeadRefName:  githubv4.String(branch),
			Title:        githubv4.String(title),
			Body:         &body,
		}
		log.Debugf("CreatePullRequestInput: %+v", input)
		pullRequestUrl, err := client.CreatePullRequestV4(input)
		if err != nil {
			return fmt.Errorf("CreatePullRequestV4: %w", err)
		}
		fmt.Println(pullRequestUrl)
	} else {
		fmt.Println(commitUrl)
	}
	return
}
