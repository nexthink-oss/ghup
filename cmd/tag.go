package cmd

import (
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/google/go-github/v68/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
)

var tagCmd = &cobra.Command{
	Use:   "tag [flags] [<name>]",
	Short: "Manage tags via the GitHub V3 API",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTagCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := tagCmd.Flags()
	flags.String("tag", "", "tag `name`")
	flags.BoolP("lightweight", "l", false, "force lightweight tag")
	addBranchFlag(flags)
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
		Owner: viper.GetString("owner"),
		Name:  viper.GetString("repo"),
	}
	branch := viper.GetString("branch")
	force := viper.GetBool("force")

	client, err := remote.NewClient(ctx, repo, token)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	targetSHA, err := client.GetSHA(branch, "heads")
	if err != nil {
		return fmt.Errorf("GetSHA(%s, %s): %w", repo, branch, err)
	}

	tagRefName, err := util.NormalizeRefName(tagName, "tags")
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
		if targetSHA == *existingTagRef.Object.SHA {
			// matching tag already exists
			fmt.Printf("https://github.com/%s/releases/tag/%s\n", repo, tagName)
			return nil
		} else if !force {
			// tag exists but points to a different commit
			return fmt.Errorf("tag '%s' already exists: %s", tagName, *existingTagRef.Object.SHA)
		}
	}

	if !viper.GetBool("lightweight") {
		message := util.BuildCommitMessage()
		tag, err := client.CreateAnnotationTag(tagName, message, targetSHA)
		if err != nil {
			return fmt.Errorf("CreateAnnotationTag(%s, %s): %w", repo, tagName, err)
		}

		targetSHA = tag.GetSHA()
	}

	tagRef := &github.Reference{
		Ref:    &tagRefName,
		Object: &github.GitObject{SHA: github.Ptr(targetSHA)},
	}

	if err := client.CreateOrUpdateRef(existingTagRef, tagRef, true); err != nil {
		return fmt.Errorf("CreateOrUpdateRef(%s, %s): %w", repo, tagRefName, err)
	}

	fmt.Printf("https://github.com/%s/releases/tag/%s\n", repo, tagName)

	return nil
}
