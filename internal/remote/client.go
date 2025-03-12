package remote

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v69/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/nexthink-oss/ghup/internal/util"
)

type Client struct {
	context context.Context
	repo    Repo
	V3      *github.Client
	V4      *githubv4.Client
}

type Repo struct {
	Owner string `json:"owner"`
	Name  string `json:"repo"`
}

func (r Repo) String() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

type PullRequest struct {
	RepoId string `json:"-" yaml:"-"`
	Number int    `json:"number,omitzero" yaml:"number,omitempty"`
	Url    string `json:"url"`
	Head   string `json:"head"`
	Base   string `json:"base"`
	Draft  bool   `json:"draft"`
	Title  string `json:"title"`
	Body   string `json:"-" yaml:"-"`
}

func NewClient(ctx context.Context, repo Repo, token string) (client *Client, err error) {
	token, err = ResolveToken(token)
	if err != nil {
		return
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	httpClient := oauth2.NewClient(ctx, src)
	rateLimiter := github_ratelimit.NewClient(httpClient.Transport)
	if err != nil {
		return nil, err
	}

	client = &Client{
		// disable go-github's built-in rate limiting
		context: context.WithValue(ctx, github.BypassRateLimitCheck, true),
		repo:    repo,
		V3:      github.NewClient(rateLimiter),
		V4:      githubv4.NewClient(rateLimiter),
	}

	return client, nil
}

// ResolveToken tries to find a GitHub token in the following order:
// 1. If the token is a file path, read the file and return the contents
// 2. If the token is non-empty, return the token as is
// 3. If the token is empty, return an error
func ResolveToken(token string) (string, error) {
	if _, err := os.Stat(token); err == nil {
		tokenBytes, err := os.ReadFile(token)
		if err != nil {
			return "", fmt.Errorf("read token file: %w", err)
		}
		token = strings.TrimSpace(string(tokenBytes))
	}

	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("unable to resolve token")
}

// GetCommitSHA validates the existence of and retrieves the full SHA
// of a commit given a short SHA
func (c *Client) GetCommitSHA(short string) (sha string, err error) {
	sha, _, err = c.V3.Repositories.GetCommitSHA1(c.context, c.repo.Owner, c.repo.Name, short, "")
	if err != nil {
		return "", err
	}

	return sha, nil
}

// GetRefSHA validates the existing of and returns the HEAD SHA of a ref
func (c *Client) GetRefSHA(refName, refType string) (sha string, err error) {
	refNorm, err := util.QualifiedRefName(refName, refType)
	if err != nil {
		return "", fmt.Errorf("QualifiedRefName(%s, %s): %w", refName, refType, err)
	}

	log.Infof("resolving ref: %s", refNorm)
	ref, _, err := c.V3.Git.GetRef(c.context, c.repo.Owner, c.repo.Name, refNorm)
	if err != nil {
		return "", fmt.Errorf("GetRef(%s, %s, %s): %w", c.repo.Owner, c.repo.Name, refNorm, err)
	}

	return ref.Object.GetSHA(), nil
}

// GetSHA returns the full SHA of a commitish
func (c *Client) GetSHA(commitish, defaultRefType string) (sha string, err error) {
	if util.IsCommitHash(commitish) {
		return c.GetCommitSHA(commitish)
	}

	return c.GetRefSHA(commitish, defaultRefType)
}

// ResolveCommitish resolves a commitish to a full SHA using the GitHub GraphQL API
func (c *Client) ResolveCommitish(commitish string) (sha string, err error) {
	var query struct {
		Repository struct {
			Object struct {
				Commit struct {
					Oid githubv4.GitObjectID
				} `graphql:"... on Commit"`
			} `graphql:"object(expression: $commitish)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]any{
		"owner":     githubv4.String(c.repo.Owner),
		"repo":      githubv4.String(c.repo.Name),
		"commitish": githubv4.String(commitish),
	}

	err = c.V4.Query(c.context, &query, variables)
	if err != nil {
		return "", err
	}

	sha = string(query.Repository.Object.Commit.Oid)
	if sha == "" {
		err = fmt.Errorf("commitish %q does not exist", commitish)
	}
	return sha, nil
}

type branchInfo struct {
	Name      string
	Commit    githubv4.GitObjectID
	CommitUrl githubv4.URI
}

type repositoryInfo struct {
	NodeID        string
	IsEmpty       bool
	DefaultBranch branchInfo
	TargetBranch  branchInfo
}

// GetRepositoryInfo returns information about a repository
func (c *Client) GetRepositoryInfo(branch string) (repository repositoryInfo, err error) {
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
					Oid       githubv4.GitObjectID
					CommitUrl githubv4.URI
				}
			} `graphql:"ref(qualifiedName: $branch)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]any{
		"owner":  githubv4.String(c.repo.Owner),
		"repo":   githubv4.String(c.repo.Name),
		"branch": githubv4.String(branch),
	}
	err = c.V4.Query(c.context, &query, variables)
	if err != nil {
		return
	}

	repository = repositoryInfo{
		NodeID:  string(query.Repository.Id),
		IsEmpty: bool(query.Repository.IsEmpty),
		DefaultBranch: branchInfo{
			Name:   string(query.Repository.DefaultBranchRef.Name),
			Commit: query.Repository.DefaultBranchRef.Target.Oid,
		},
	}

	if query.Repository.Ref != nil {
		repository.TargetBranch = branchInfo{
			Name:      branch,
			Commit:    query.Repository.Ref.Target.Oid,
			CommitUrl: query.Repository.Ref.Target.CommitUrl,
		}
	}

	return
}

// GetFileContentV4 returns the content and hash of a file on the given branch.
// It is limited to non-binary files, as the content is returned as a string.
// If there is any error (including binary file)
func (c *Client) GetFileContentV4(branch string, path string) (content string, ok bool) {
	branchPath := fmt.Sprintf("%s:%s", branch, path)
	var query struct {
		Repository struct {
			Object struct {
				Blob struct {
					Text     githubv4.String
					IsBinary githubv4.Boolean
				} `graphql:"... on Blob"`
			} `graphql:"object(expression: $branchPath)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]any{
		"owner":      githubv4.String(c.repo.Owner),
		"repo":       githubv4.String(c.repo.Name),
		"branchPath": githubv4.String(branchPath),
	}
	err := c.V4.Query(c.context, &query, variables)
	if err == nil {
		log.Warnf("Got file content for %q; binary=%v; len=%d", branchPath, bool(query.Repository.Object.Blob.IsBinary), len(query.Repository.Object.Blob.Text))
		content = string(query.Repository.Object.Blob.Text)
		ok = !bool(query.Repository.Object.Blob.IsBinary)
	}
	return
}

// GetFileHashV4 returns the hash of a file on the given branch
func (c *Client) GetFileHashV4(branch string, path string) (hash string) {
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

	variables := map[string]any{
		"owner":  githubv4.String(c.repo.Owner),
		"repo":   githubv4.String(c.repo.Name),
		"branch": githubv4.String(branch),
		"path":   githubv4.String(path),
	}
	err := c.V4.Query(c.context, &query, variables)
	if err == nil {
		hash = string(query.Repository.Object.Commit.File.Oid)
	}
	return
}

func (c *Client) GetRefOidV4(refName string) (oid githubv4.GitObjectID, err error) {
	var query struct {
		Repository struct {
			Ref struct {
				Target struct {
					Oid githubv4.GitObjectID
				}
			} `graphql:"ref(qualifiedName: $refName)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]any{
		"owner":   githubv4.String(c.repo.Owner),
		"repo":    githubv4.String(c.repo.Name),
		"refName": githubv4.String(refName),
	}

	err = c.V4.Query(c.context, &query, variables)
	if err != nil {
		return
	}

	oid = query.Repository.Ref.Target.Oid
	if oid == "" {
		err = fmt.Errorf("ref %q does not exist", refName)
	}
	return
}

func (c *Client) CreateAnnotationTag(name, message, sha string) (*github.Tag, error) {
	annotatedTag := &github.Tag{
		Tag:     &name,
		Message: &message,
		Object: &github.GitObject{
			Type: github.Ptr("commit"),
			SHA:  github.Ptr(sha),
		},
	}
	log.Debugf("Tag: %+v", annotatedTag)
	annotatedTag, _, err := c.V3.Git.CreateTag(c.context, c.repo.Owner, c.repo.Name, annotatedTag)
	if err != nil {
		return nil, fmt.Errorf("CreateTag(%s, %s): %w", c.repo, name, err)
	}

	return annotatedTag, nil
}

func (c *Client) CreateOrUpdateRef(old, new *github.Reference, force bool) error {
	if old == nil {
		log.Infof("CreateRef(%s, %s)", c.repo, new.String())
		if _, _, err := c.V3.Git.CreateRef(c.context, c.repo.Owner, c.repo.Name, new); err != nil {
			return fmt.Errorf("CreateRef(%s, %s): %w", c.repo, new.String(), err)
		}
	} else {
		log.Infof("UpdateRef(%s, %s)", c.repo, new.String())
		if _, _, err := c.V3.Git.UpdateRef(c.context, c.repo.Owner, c.repo.Name, new, force); err != nil {
			return fmt.Errorf("UpdateRef(%s, %s): %w", c.repo, new.String(), err)
		}
	}

	return nil
}

func (c *Client) CreateRefV4(input githubv4.CreateRefInput) (err error) {
	var mutation struct {
		CreateRef struct {
			Ref struct {
				Target struct {
					Oid githubv4.GitObjectID
				}
			}
		} `graphql:"createRef(input: $input)"`
	}

	err = c.V4.Mutate(c.context, &mutation, input, nil)

	return
}

func (c *Client) CreateCommitOnBranchV4(input githubv4.CreateCommitOnBranchInput) (oid githubv4.GitObjectID, url string, err error) {
	var mutation struct {
		CreateCommitOnBranch struct {
			Commit struct {
				Oid githubv4.GitObjectID
				Url githubv4.String
			}
		} `graphql:"createCommitOnBranch(input: $input)"`
	}

	err = c.V4.Mutate(c.context, &mutation, input, nil)
	if err != nil {
		return
	}

	oid = mutation.CreateCommitOnBranch.Commit.Oid
	url = string(mutation.CreateCommitOnBranch.Commit.Url)

	return
}

func (c *Client) FindPullRequestUrl(pullRequest *PullRequest) (found bool, err error) {
	if pullRequest == nil {
		return false, fmt.Errorf("pull request is nil")
	}

	var query struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					Number            githubv4.Int
					Url               githubv4.String
					Title             githubv4.String
					IsCrossRepository githubv4.Boolean
				}
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage githubv4.Boolean
				}
			} `graphql:"pullRequests(states: OPEN, baseRefName: $baseBranch, headRefName: $headBranch, first: 100, after: $cursor)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]any{
		"owner":      githubv4.String(c.repo.Owner),
		"repo":       githubv4.String(c.repo.Name),
		"baseBranch": githubv4.String(pullRequest.Base),
		"headBranch": githubv4.String(pullRequest.Head),
		"cursor":     (*githubv4.String)(nil),
	}

	for {
		if c.V4.Query(c.context, &query, variables) != nil {
			return false, err
		}

		for _, pr := range query.Repository.PullRequests.Nodes {
			if pr.IsCrossRepository {
				continue // ignore cross-repository PRs
			}

			pullRequest.Number = int(pr.Number)
			pullRequest.Url = string(pr.Url)
			pullRequest.Title = string(pr.Title)
			return true, nil
		}

		if !query.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Repository.PullRequests.PageInfo.EndCursor)
	}

	return false, nil
}

func (c *Client) CreatePullRequestV4(pullRequest *PullRequest) (err error) {
	var mutation struct {
		CreatePullRequest struct {
			PullRequest struct {
				Permalink githubv4.URI
				Number    githubv4.Int
			}
		} `graphql:"createPullRequest(input: $input)"`
	}

	body := githubv4.String(pullRequest.Body)
	input := githubv4.CreatePullRequestInput{
		RepositoryID: pullRequest.RepoId,
		BaseRefName:  githubv4.String(pullRequest.Base),
		Draft:        githubv4.NewBoolean(githubv4.Boolean(pullRequest.Draft)),
		HeadRefName:  githubv4.String(pullRequest.Head),
		Title:        githubv4.String(pullRequest.Title),
		Body:         &body,
	}

	err = c.V4.Mutate(c.context, &mutation, input, nil)
	if err != nil {
		return
	}

	pullRequest.Url = mutation.CreatePullRequest.PullRequest.Permalink.String()
	pullRequest.Number = int(mutation.CreatePullRequest.PullRequest.Number)

	return
}

func (c *Client) UpdateRefName(refName string, targetRef *github.Reference, force bool) (oldHash string, newHash string, err error) {
	legacyRef, _, err := c.V3.Git.GetRef(c.context, c.repo.Owner, c.repo.Name, refName)
	if err != nil {
		log.Infof("creating ref %q", refName)
		updatedRef, _, err := c.V3.Git.CreateRef(c.context, c.repo.Owner, c.repo.Name, targetRef)
		if err != nil {
			return "", "", err
		}
		return "", updatedRef.Object.GetSHA(), nil
	}

	log.Infof("updating ref %q", refName)
	updatedRef, _, err := c.V3.Git.UpdateRef(c.context, c.repo.Owner, c.repo.Name, targetRef, force)
	if err != nil {
		return "", "", err
	}

	return legacyRef.Object.GetSHA(), updatedRef.Object.GetSHA(), nil
}

func (c *Client) GetMatchingHeads(commitish string) (headNames []string, err error) {
	branches, _, err := c.V3.Repositories.ListBranchesHeadCommit(c.context, c.repo.Owner, c.repo.Name, commitish)
	if err != nil {
		return nil, err
	}

	for _, branch := range branches {
		headNames = append(headNames, *branch.Name)
	}

	return headNames, nil
}

func (c *Client) GetMatchingTags(sha string) (tagNames []string, err error) {
	// get all tags, iterating over all pages
	opts := &github.ListOptions{PerPage: 100}
	for {
		tags, resp, err := c.V3.Repositories.ListTags(c.context, c.repo.Owner, c.repo.Name, opts)
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			if *tag.Commit.SHA == sha {
				tagNames = append(tagNames, *tag.Name)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return tagNames, nil
}
