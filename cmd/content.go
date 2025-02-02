package cmd

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/local"
	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
)

type contentReport struct {
	Repository  string              `json:"repository,omitempty"`
	SHA         string              `json:"sha"`
	Updated     bool                `json:"updated"`
	PullRequest *remote.PullRequest `json:"pullrequest,omitempty" yaml:"pullrequest,omitempty"`
}

var contentCmd = &cobra.Command{
	Use:     "content [flags] [<file-spec> ...]",
	Aliases: []string{"commit"},
	Short:   "Manage repository content.",
	Long:    `Directly manage repository content via the GitHub API, ensuring verified commits from CI systems.`,
	Args:    cobra.ArbitraryArgs,
	RunE:    runContentCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := contentCmd.Flags()

	flags.Bool("tracked", false, "commit changes to tracked files")
	flags.Bool("staged", false, "commit staged changes")
	flags.StringSliceP("copy", "c", []string{}, "remote file-spec to copy (`[src-branch<separator>]src-path[<separator>dst-path]`); non-binary files only!")
	flags.StringSliceP("update", "u", []string{}, "file-spec to update (`local-path[<separator>remote-path]`)")
	flags.StringSliceP("delete", "d", []string{}, "`remote-path` to delete")
	flags.StringP("separator", "s", ":", "file-spec `separator`")
	addCommitMessageFlags(flags)
	addBranchFlag(flags)
	flags.Bool("create-branch", true, "create missing target branch")
	flags.StringP("base-branch", "B", "", `base branch `+"`name`"+` (default: "[remote-default-branch])"`)
	addPullRequestFlags(flags)
	addDryRunFlag(flags)
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
		Owner: repoOwner,
		Name:  repoName,
	}
	targetBranch := branchName
	dryRun := viper.GetBool("dry-run")
	force := viper.GetBool("force")

	report := contentReport{
		Repository: repo.String(),
	}

	client, err := remote.NewClient(ctx, repo, githubToken)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	repoInfo, err := client.GetRepositoryInfo(targetBranch)
	if err != nil {
		return fmt.Errorf("GetRepositoryInfo(%s, %s): %w", repo, targetBranch, err)
	}

	if repoInfo.IsEmpty {
		return fmt.Errorf("cannot push to empty repository")
	}

	targetOid := repoInfo.TargetBranch.Commit
	targetBranchIsNew := targetOid == ""

	baseBranch := cmp.Or(viper.GetString("base-branch"), repoInfo.DefaultBranch.Name)

	if targetBranchIsNew {
		log.Debug("target branch is new")

		if !viper.GetBool("create-branch") {
			return fmt.Errorf("branch %q does not exist", targetBranch)
		}

		// determine the oid for the target branch
		if baseBranch == repoInfo.DefaultBranch.Name {
			targetOid = repoInfo.DefaultBranch.Commit
		} else {
			targetOid, err = client.GetRefOidV4(baseBranch)
			if err != nil {
				return fmt.Errorf("GetRefOidV4(%s, %s): %w", repo, baseBranch, err)
			}
		}

		createRefInput := githubv4.CreateRefInput{
			RepositoryID: repoInfo.NodeID,
			Name:         githubv4.String(fmt.Sprintf("refs/heads/%s", targetBranch)),
			Oid:          targetOid,
		}

		if !dryRun {
			log.Infof("creating target branch %q", targetBranch)
			log.Debugf("CreateRefInput: %+v", createRefInput)

			if err := client.CreateRefV4(createRefInput); err != nil {
				return fmt.Errorf("CreateRefV4: %w", err)
			}
		} else {
			log.Infof("dry-run: skipping creation of branch: %q from %s", targetBranch, targetOid)
		}
	}

	pathContent := make(local.PathContent)
	deletionSet := make(local.DeletionSet)

	commitStaged := viper.GetBool("staged")
	commitTracked := viper.GetBool("tracked")

	if commitStaged || commitTracked {
		gitStatus, err := localRepo.Status()
		if err != nil {
			return fmt.Errorf("Local repository status: %w", err)
		}

		if commitStaged {
			pathContent, deletionSet, err = localRepo.Staged(gitStatus)
			if err != nil {
				return fmt.Errorf("calculating staged changes: %w", err)
			}
		} else { // commitTracked
			pathContent, deletionSet, err = localRepo.Tracked(gitStatus)
			if err != nil {
				return fmt.Errorf("calculating all changes: %w", err)
			}
		}
	}

	for _, spec := range viper.GetStringSlice("copy") {
		branch, source, target, err := local.ParseCopySpec(spec, separator)
		if err != nil {
			return fmt.Errorf("ParseCopySpec(%q, %q): %w", spec, separator, err)
		}

		branch = cmp.Or(branch, baseBranch)
		if content, ok := client.GetFileContentV4(branch, source); ok {
			pathContent[target] = []byte(content)
		}
	}

	for spec := range util.SliceChain(viper.GetStringSlice("update"), args) {
		source, target, err := local.ParseUpdateSpec(spec, separator)
		if err != nil {
			return fmt.Errorf("ParseUpdateSpec(%q, %q): %w", spec, separator, err)
		}

		content, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("ReadFile(%s): %w", source, err)
		}

		pathContent[target] = content
		// an explicit update overrides previous deletions
		delete(deletionSet, target)
	}

	for _, target := range viper.GetStringSlice("delete") {
		target = filepath.Clean(target)
		deletionSet[target] = struct{}{}
		// an explicit deletion overrides previous updates
		delete(pathContent, target)
	}

	// we now have the full set of changes, so can proceed to calculate idempotent operations

	additionMap := make(map[string]githubv4.FileAddition, 0)
	deletionMap := make(map[string]githubv4.FileDeletion, 0)

	for path, content := range pathContent {
		localHash := plumbing.ComputeHash(plumbing.BlobObject, content).String()
		remoteHash := client.GetFileHashV4(targetBranch, path)
		if localHash != remoteHash || force {
			additionMap[path] = githubv4.FileAddition{
				Path:     githubv4.String(path),
				Contents: githubv4.Base64String(base64.StdEncoding.EncodeToString(content)),
			}
			log.Debugf("%q queued for addition", path)
		} else {
			log.Debugf("%q (%s) on target branch: skipping addition", path, remoteHash)
		}
	}

	for path := range deletionSet {
		remoteHash := client.GetFileHashV4(targetBranch, path)
		if remoteHash != "" || force {
			deletionMap[path] = githubv4.FileDeletion{
				Path: githubv4.String(path),
			}
			log.Debugf("%q queued for deletion", path)
		} else {
			log.Debugf("%q absent on target branch: skipping deletion", path)
		}
	}

	additions := util.MapValues(additionMap)
	deletions := util.MapValues(deletionMap)

	if len(additions) == 0 && len(deletions) == 0 {
		report.SHA = string(repoInfo.TargetBranch.Commit)
	} else {

		changes := githubv4.FileChanges{
			Additions: &additions,
			Deletions: &deletions,
		}

		message := util.BuildCommitMessage()

		input := githubv4.CreateCommitOnBranchInput{
			Branch:          remote.CommittableBranch(repo, targetBranch),
			Message:         remote.CommitMessage(message),
			ExpectedHeadOid: targetOid,
			FileChanges:     &changes,
		}

		log.Debugf("CreateCommitOnBranchInput: %+v", input)

		if !dryRun {
			sha, _, err := client.CreateCommitOnBranchV4(input)
			if err != nil {
				return fmt.Errorf("CommitOnBranchV4: %w", err)
			}

			report.SHA = string(sha)
		}

		report.Updated = true
	}

	if prTitle := viper.GetString("pr-title"); prTitle != "" {
		pullRequest := remote.PullRequest{
			RepoId: repoInfo.NodeID,
			Head:   targetBranch,
			Base:   baseBranch,
			Title:  prTitle,
			Body:   viper.GetString("pr-body"),
			Draft:  viper.GetBool("pr-draft"),
		}

		var prExists bool
		// check for existing pull request if target branch was pre-existing
		if !targetBranchIsNew {
			prExists, err = client.FindPullRequestUrl(&pullRequest)
			if err != nil {
				return fmt.Errorf("finding open pull requests: %w", err)
			}
		}

		if prExists {
			log.Debugf("found open pull request: %s", pullRequest.Url)
			report.PullRequest = &pullRequest
		} else {
			if !dryRun {
				log.Debugf("opening pull request from %q to %q", pullRequest.Head, pullRequest.Base)
				err = client.CreatePullRequestV4(&pullRequest)
				if err != nil {
					return fmt.Errorf("CreatePullRequestV4: %w", err)
				}
			}
			report.PullRequest = &pullRequest
		}
	}

	commandOutput = report

	return nil
}
