package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/google/go-github/v48/github"
	"github.com/isometry/ghup/internal/auth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tagCmd = &cobra.Command{
	Use:        "tag [flags] <tagname>",
	Short:      "create tag",
	Args:       cobra.ExactArgs(1),
	ArgAliases: []string{"tagname"},
	RunE:       runTagCmd,
}

func init() {
	rootCmd.AddCommand(tagCmd)
}

func runTagCmd(cmd *cobra.Command, args []string) (err error) {
	defer log.Trace("tag").Stop(&err)

	ctx := context.Background()

	client, err := auth.NewTokenClient(ctx)
	if err != nil {
		return err
	}

	owner := viper.GetString("owner")
	repo := viper.GetString("repo")
	branch := viper.GetString("branch")
	branchRefName := fmt.Sprintf("heads/%s", branch)
	message := viper.GetString("message")

	tagName := args[0]
	tagRefName := fmt.Sprintf("tags/%s", tagName)
	var tagRefObject string

	log.Debugf("getting tag reference: %s", tagRefName)
	existingTagRef, resp, err := client.Git.GetRef(ctx, owner, repo, tagRefName)
	if err != nil && (resp == nil || (resp != nil && resp.StatusCode != http.StatusNotFound)) {
		return err
	} else if err == nil && !viper.GetBool("force") {
		return fmt.Errorf("tag '%s' already exists: %s", tagName, *existingTagRef.Object.SHA)
	}

	log.Debugf("getting branch reference: %s", branchRefName)
	branchRef, _, err := client.Git.GetRef(ctx, owner, repo, branchRefName)
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
		annotatedTag, _, err = client.Git.CreateTag(ctx, owner, repo, annotatedTag)
		if err != nil {
			return err
		}
		tagRefObject = annotatedTag.GetSHA()
	} else {
		tagRefObject = branchRef.Object.GetSHA()
	}

	if existingTagRef != nil {
		log.Debugf("deleting existing tag reference")
		if _, err := client.Git.DeleteRef(ctx, owner, repo, existingTagRef.GetRef()); err != nil {
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
	_, _, err = client.Git.CreateRef(ctx, owner, repo, tagRef)
	if err != nil {
		return err
	}

	return nil
}
