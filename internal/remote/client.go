package remote

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v64/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type TokenClient struct {
	Context context.Context
	V3      *github.Client
	V4      *githubv4.Client
}

type BranchInfo struct {
	Name   string
	Commit githubv4.GitObjectID
}

type RepositoryInfo struct {
	NodeID        string
	IsEmpty       bool
	DefaultBranch BranchInfo
	TargetBranch  BranchInfo
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

func WithAccept(accept string) github.RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Accept", accept)
	}
}

// GetCommitSHA validates the existence of and retrieves the full SHA
// of a commit given a short SHA
func (c *TokenClient) GetCommitSHA(ctx context.Context, owner string, repo string, sha string) (*string, *github.Response, error) {
	u := fmt.Sprintf("repos/%v/%v/commits/%v", owner, repo, sha)
	req, err := c.V3.NewRequest("GET", u, nil, WithAccept("application/vnd.github.sha"))
	if err != nil {
		return nil, nil, err
	}

	var commit bytes.Buffer
	resp, err := c.V3.Do(ctx, req, &commit)
	if err != nil {
		return nil, resp, err
	}

	commitSHA := commit.String()
	return &commitSHA, resp, nil
}

func (c *TokenClient) GetRepositoryInfo(owner string, repo string, branch string) (repository RepositoryInfo, err error) {
	var query struct {
		Repository struct {
			Id               githubv4.String
			IsEmpty          githubv4.Boolean
			DefaultBranchRef struct {
				Name   githubv4.String
				Target struct {
					Oid githubv4.GitObjectID
				}
			}
			Ref *struct {
				Target struct {
					Oid githubv4.GitObjectID
				}
			} `graphql:"ref(qualifiedName: $branch)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}
	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"repo":   githubv4.String(repo),
		"branch": githubv4.String(branch),
	}
	err = c.V4.Query(c.Context, &query, variables)
	if err != nil {
		return
	}

	repository = RepositoryInfo{
		NodeID:  string(query.Repository.Id),
		IsEmpty: bool(query.Repository.IsEmpty),
		DefaultBranch: BranchInfo{
			Name:   string(query.Repository.DefaultBranchRef.Name),
			Commit: query.Repository.DefaultBranchRef.Target.Oid,
		},
	}

	if query.Repository.Ref != nil {
		repository.TargetBranch = BranchInfo{
			Name:   branch,
			Commit: query.Repository.Ref.Target.Oid,
		}
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

func (c *TokenClient) GetRefOidV4(owner string, repo string, refName string) (oid githubv4.GitObjectID, err error) {
	var query struct {
		Repository struct {
			Ref struct {
				Target struct {
					Oid githubv4.GitObjectID
				}
			} `graphql:"ref(qualifiedName: $refName)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]interface{}{
		"owner":   githubv4.String(owner),
		"repo":    githubv4.String(repo),
		"refName": githubv4.String(refName),
	}

	err = c.V4.Query(c.Context, &query, variables)
	if err != nil {
		return
	}

	oid = query.Repository.Ref.Target.Oid
	if oid == "" {
		err = fmt.Errorf("ref %q does not exist", refName)
	}
	return
}

func (c *TokenClient) CreateRefV4(createRefInput githubv4.CreateRefInput) (err error) {
	var mutation struct {
		CreateRef struct {
			Ref struct {
				Target struct {
					Oid githubv4.GitObjectID
				}
			}
		} `graphql:"createRef(input: $input)"`
	}

	err = c.V4.Mutate(c.Context, &mutation, createRefInput, nil)

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

func (c *TokenClient) CreatePullRequestV4(createPullRequestInput githubv4.CreatePullRequestInput) (url string, err error) {
	var mutation struct {
		CreatePullRequest struct {
			PullRequest struct {
				Permalink githubv4.URI
			}
		} `graphql:"createPullRequest(input: $input)"`
	}

	err = c.V4.Mutate(c.Context, &mutation, createPullRequestInput, nil)
	if err != nil {
		return
	}

	url = mutation.CreatePullRequest.PullRequest.Permalink.String()
	return
}
