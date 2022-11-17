package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/google/go-github/v48/github"
	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tagName string

var tagCmd = &cobra.Command{
	Use:     "tag [flags] [<name>]",
	Short:   "Manage tags via the GitHub V3 API",
	Args:    cobra.MaximumNArgs(1),
	PreRunE: validateTagName,
	RunE:    runTagCmd,
}

func init() {
	tagCmd.Flags().String("tag", "", "tag name")
	viper.BindPFlag("tag", tagCmd.Flags().Lookup("tag"))

	rootCmd.AddCommand(tagCmd)
}

func runTagCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()

	client, err := remote.NewTokenClient(ctx, token)
	if err != nil {
		return err
	}

	branchRefName := fmt.Sprintf("heads/%s", branch)

	tagRefName := fmt.Sprintf("tags/%s", tagName)
	var tagRefObject string

	log.Debugf("getting tag reference: %s", tagRefName)
	existingTagRef, resp, err := client.V3.Git.GetRef(ctx, owner, repo, tagRefName)
	if err != nil && (resp == nil || (resp != nil && resp.StatusCode != http.StatusNotFound)) {
		return err
	} else if err == nil && !viper.GetBool("force") {
		return fmt.Errorf("tag '%s' already exists: %s", tagName, *existingTagRef.Object.SHA)
	}

	log.Debugf("getting branch reference: %s", branchRefName)
	branchRef, _, err := client.V3.Git.GetRef(ctx, owner, repo, branchRefName)
	if err != nil {
		return err
	}

	if message != "" {
		annotatedTag := &github.Tag{
			Tag:     &tagName,
			Message: &message,
			// Message: github.String("hard-coded message"),
			Object: &github.GitObject{
				Type: github.String("commit"),
				SHA:  github.String(branchRef.Object.GetSHA()),
			},
		}
		log.Debugf("creating annotated tag")
		annotatedTag, _, err = client.V3.Git.CreateTag(ctx, owner, repo, annotatedTag)
		if err != nil {
			return err
		}
		tagRefObject = annotatedTag.GetSHA()
	} else {
		tagRefObject = branchRef.Object.GetSHA()
	}

	if existingTagRef != nil {
		log.Debugf("deleting existing tag reference")
		if _, err := client.V3.Git.DeleteRef(ctx, owner, repo, existingTagRef.GetRef()); err != nil {
			return err
		}
	}

	tagRef := &github.Reference{
		Ref: &tagRefName,
		Object: &github.GitObject{
			SHA: github.String(tagRefObject),
		},
	}

	log.Debugf("creating tag reference")
	_, _, err = client.V3.Git.CreateRef(ctx, owner, repo, tagRef)
	if err != nil {
		return err
	}

	fmt.Printf("https://github.com/%s/%s/releases/tag/%s\n", owner, repo, tagName)
	return
}

func validateTagName(cmd *cobra.Command, args []string) (err error) {
	tagName = viper.GetString("tag")

	if len(args) == 1 {
		tagName = args[0]
	}

	if tagName == "" {
		return fmt.Errorf("no tag specified")
	}

	return
}
