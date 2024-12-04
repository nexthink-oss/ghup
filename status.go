package cmd

import (
	"fmt"

	"github.com/apex/log"
	"github.com/go-git/go-git/v5"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [flags]",
	Short: "status",
	Args:  cobra.NoArgs,
	RunE:  runStatusCmd,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatusCmd(cmd *cobra.Command, args []string) (err error) {
	defer log.Trace("commit").Stop(&err)

	if localRepo == nil {
		return fmt.Errorf("the commit action is only valid within a repo with GitHub remote")
	}

	worktree, err := localRepo.Repository.Worktree()
	if err != nil {
		return err
	}

	worktreeStatus, err := worktree.Status()
	if err != nil {
		return err
	}

	idx, err := localRepo.Repository.Storer.Index()
	if err != nil {
		return err
	}
	for _, e := range idx.Entries {
		log.Debugf("index entry: %+v (stage: %+v)\n", e, e.Stage)
	}

	additions := []githubv4.FileAddition{}
	deletions := []githubv4.FileDeletion{}

	for path, status := range worktreeStatus {
		log.Debugf("%c%c %s\n", status.Staging, status.Worktree, path)
		switch status.Staging {
		case git.Added, git.Modified:
			additions = append(additions, githubv4.FileAddition{
				Path: githubv4.String(path),
			})
		}
	}

	fmt.Printf("additions: %+v\n", additions)
	fmt.Printf("deletions: %+v\n", deletions)

	// changes := githubv4.FileChanges{
	// 	Additions: &additions
	// 	Deletions: &deletions
	// }

	// ctx := context.Background()

	// client, err := auth.NewTokenClient(ctx)
	// if err != nil {
	// 	return err
	// }

	// owner := viper.GetString("owner")
	// repo := viper.GetString("repo")
	// branch := viper.GetString("branch")
	// message := viper.GetString("message")

	// branchRef, _, err := client.Git.GetRef(ctx, owner, repo, fmt.Sprintf("heads/%s", branch))
	// if err != nil {
	// 	return err
	// }

	return nil
}
