package remote

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v48/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type TokenClient struct {
	Context context.Context
	V3      *github.Client
	V4      *githubv4.Client
}

func NewTokenClient(ctx context.Context, token string) (client *TokenClient, err error) {
	token, err = ResolveToken(token)
	if err != nil {
		return
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	httpClient := oauth2.NewClient(ctx, src)

	client = &TokenClient{
		Context: ctx,
		V3:      github.NewClient(httpClient),
		V4:      githubv4.NewClient(httpClient),
	}

	return client, nil
}

func ResolveToken(tokenVar string) (token string, err error) {
	token = tokenVar

	if _, err := os.Stat(token); err == nil {
		tokenBytes, err := os.ReadFile(token)
		if err != nil {
			return "", err
		}
		token = strings.TrimSpace(string(tokenBytes))
	}

	if token == "" {
		return "", fmt.Errorf("no GitHub Token found")
	}

	return
}

func (c *TokenClient) GetFileHashV4(owner string, repo string, branch string, path string) (hash string) {
	var query struct {
		Repository struct {
			Object struct {
				Commit struct {
					File struct {
						Oid githubv4.GitObjectID
					} `graphql:"file(path: $path)"`
				} `graphql:"... on Commit"`
			} `graphql:"object(expression: $branch)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}
	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"repo":   githubv4.String(repo),
		"branch": githubv4.String(branch),
		"path":   githubv4.String(path),
	}
	err := c.V4.Query(c.Context, &query, variables)
	if err == nil {
		hash = string(query.Repository.Object.Commit.File.Oid)
	}
	return
}

func (c *TokenClient) GetHeadOidV4(owner string, repo string, branch string) (oid githubv4.GitObjectID, err error) {
	var query struct {
		Repository struct {
			Ref struct {
				Target struct {
					Oid githubv4.String
				}
			} `graphql:"ref(qualifiedName: $branchName)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]interface{}{
		"owner":      githubv4.String(owner),
		"repo":       githubv4.String(repo),
		"branchName": githubv4.String(branch),
	}

	err = c.V4.Query(c.Context, &query, variables)
	if err != nil {
		return
	}

	oid = githubv4.GitObjectID(query.Repository.Ref.Target.Oid)
	return
}

func (c *TokenClient) CommitOnBranchV4(createCommitOnBranchInput githubv4.CreateCommitOnBranchInput) (oid githubv4.GitObjectID, url string, err error) {
	var mutation struct {
		CreateCommitOnBranch struct {
			Commit struct {
				Oid githubv4.GitObjectID
				Url githubv4.String
			}
		} `graphql:"createCommitOnBranch(input: $input)"`
	}

	err = c.V4.Mutate(c.Context, &mutation, createCommitOnBranchInput, nil)
	if err != nil {
		return
	}

	oid = mutation.CreateCommitOnBranch.Commit.Oid
	url = string(mutation.CreateCommitOnBranch.Commit.Url)
	return
}
