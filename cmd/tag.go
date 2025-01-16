package cmd

import (
	"context"
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
	Use:     "tag [flags] [<name>]",
	Short:   "Manage tags via the GitHub V3 API",
	Args:    cobra.MaximumNArgs(1),
	PreRunE: validateFlags,
	RunE:    runTagCmd,
}

func init() {
	tagCmd.Flags().String("tag", "", "tag name")
	viper.BindPFlag("tag", tagCmd.Flags().Lookup("tag"))

	tagCmd.Flags().BoolP("lightweight", "l", false, "force lightweight tag")
	viper.BindPFlag("lightweight", tagCmd.Flags().Lookup("lightweight"))

	tagCmd.Flags().SortFlags = false

	rootCmd.AddCommand(tagCmd)
}

func runTagCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()

	client, err := remote.NewTokenClient(ctx, viper.GetString("token"))
	if err != nil {
		return fmt.Errorf("NewTokenClient: %w", err)
	}

	tagName := viper.GetString("tag")

	if len(args) == 1 {
		tagName = args[0]
	}

	if tagName == "" {
		return fmt.Errorf("no tag specified")
	}

	branchRefName := fmt.Sprintf("heads/%s", branch)

	tagRefName := fmt.Sprintf("tags/%s", tagName)
	if err := util.IsValidRefName(tagRefName); err != nil {
		return fmt.Errorf("Invalid tag reference: %s: %w", tagRefName, err)
	}

	var tagRefObject string

	log.Infof("getting tag reference: %s", tagRefName)
	existingTagRef, resp, err := client.V3.Git.GetRef(ctx, owner, repo, tagRefName)
	if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
		return fmt.Errorf("GetRef: %w", err)
	} else if err == nil && !viper.GetBool("force") {
		return fmt.Errorf("tag '%s' already exists: %s", tagName, *existingTagRef.Object.SHA)
	}

	log.Infof("getting branch reference: %s", branchRefName)
	branchRef, _, err := client.V3.Git.GetRef(ctx, owner, repo, branchRefName)
	if err != nil {
		return fmt.Errorf("GetRef(%s, %s, %s): %w", owner, repo, branchRefName, err)
	}

	if message = util.BuildCommitMessage(); message != "" && !viper.GetBool("lightweight") {
		annotatedTag := &github.Tag{
			Tag:     &tagName,
			Message: &message,
			Object: &github.GitObject{
				Type: github.Ptr("commit"),
				SHA:  github.Ptr(branchRef.Object.GetSHA()),
			},
		}
		log.Infof("creating annotated tag")
		log.Debugf("Tag: %+v", annotatedTag)
		annotatedTag, _, err = client.V3.Git.CreateTag(ctx, owner, repo, annotatedTag)
		if err != nil {
			return fmt.Errorf("CreateTag: %w", err)
		}
		tagRefObject = annotatedTag.GetSHA()
	} else {
		tagRefObject = branchRef.Object.GetSHA()
	}

	tagRef := &github.Reference{
		Ref: &tagRefName,
		Object: &github.GitObject{
			SHA: github.Ptr(tagRefObject),
		},
	}

	if existingTagRef == nil {
		log.Infof("creating tag reference")
		_, _, err = client.V3.Git.CreateRef(ctx, owner, repo, tagRef)
		if err != nil {
			return fmt.Errorf("CreateRef: %w", err)
		}
	} else {
		log.Infof("updating tag reference")
		_, _, err = client.V3.Git.UpdateRef(ctx, owner, repo, tagRef, true)
		if err != nil {
			return fmt.Errorf("UpdateRef: %w", err)
		}
	}

	fmt.Printf("https://github.com/%s/%s/releases/tag/%s\n", owner, repo, tagName)

	return
}
