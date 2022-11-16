package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/apex/log"
	"github.com/google/go-github/v48/github"
	"github.com/isometry/ghup/internal/remote"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var fileCmd = &cobra.Command{
	Use:        "file [flags] <path>",
	Short:      "Manage a single file's contents via the GitHub V3 API (deprecated, use update instead)",
	Args:       cobra.ExactArgs(1),
	ArgAliases: []string{"path"},
	RunE:       runFileCmd,
}

func init() {
	fileCmd.Flags().StringP("content", "c", "", "override content for path (default = content of file at path; '-' for stdin)")
	viper.BindPFlag("content", fileCmd.Flags().Lookup("content"))

	rootCmd.AddCommand(fileCmd)
}

func runFileCmd(cmd *cobra.Command, args []string) (err error) {
	defer log.WithField("path", args[0]).Trace("file").Stop(&err)

	ctx := context.Background()

	client, err := remote.NewTokenClient(ctx)
	if err != nil {
		return err
	}

	filePath := args[0]
	contentArg := viper.GetString("content")
	var fileReader io.Reader

	switch contentArg {
	case "-":
		fileReader = os.Stdin
	case "":
		fileReader, err = os.Open(filePath)
		if err != nil {
			return err
		}
	default:
		fileReader, err = os.Open(contentArg)
		if err != nil {
			return err
		}
	}

	branchRef, _, err := client.V3.Git.GetRef(ctx, owner, repo, fmt.Sprintf("heads/%s", branch))
	if err != nil {
		return err
	}

	var content []byte
	content, err = io.ReadAll(fileReader)
	if err != nil {
		return err
	}

	action := "update"
	existingContent, _, resp, err := client.V3.Repositories.GetContents(ctx, owner, repo, filePath, &github.RepositoryContentGetOptions{Ref: branchRef.GetRef()})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			action = "create"
		} else {
			return err
		}
	}

	switch action {
	case "create":
		log.Infof("creating github.com/%s/%s/%s", owner, repo, filePath)
		_, _, err := client.V3.Repositories.CreateFile(ctx, owner, repo, filePath, &github.RepositoryContentFileOptions{
			Message: github.String(message),
			Content: content,
			Branch:  github.String(branch),
		})
		if err != nil {
			return err
		}
	case "update":
		log.Infof("updating github.com/%s/%s/%s", owner, repo, filePath)
		_, _, err := client.V3.Repositories.UpdateFile(ctx, owner, repo, filePath, &github.RepositoryContentFileOptions{
			Message: github.String(message),
			Content: content,
			SHA:     github.String(existingContent.GetSHA()),
			Branch:  github.String(branch),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
