package remote

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v72/github"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/nexthink-oss/ghup/internal/util"
)

var (
	ErrNoMatchingObject = errors.New("no matching object found")
)

// Auto-merge method constants
const (
	AutoMergeOff    = "off"
	AutoMergeMerge  = "merge"
	AutoMergeSquash = "squash"
	AutoMergeRebase = "rebase"
)

// GetAutoMergeChoices returns the available auto-merge choices
func GetAutoMergeChoices() []string {
	return []string{AutoMergeOff, AutoMergeMerge, AutoMergeSquash, AutoMergeRebase}
}

// ErrImmutableRef is returned when an update is skipped because the ref is immutable and has diverged
type ErrImmutableRef struct {
	RefName      string
	ExistingHash string
	ProposedHash string
}

func (e *ErrImmutableRef) Error() string {
	return fmt.Sprintf("ref %s is immutable and has diverged (existing: %s, proposed: %s)", e.RefName, e.ExistingHash, e.ProposedHash)
}

type Client struct {
	context context.Context
	repo    *Repo
	V3      *github.Client
	V4      *githubv4.Client
}

type Repo struct {
	Owner string `json:"owner"`
	Name  string `json:"repo"`
}

func (r *Repo) String() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

type PullRequest struct {
	RepoId        string `json:"-" yaml:"-"`
	Number        int    `json:"number,omitzero" yaml:"number,omitempty"`
	Url           string `json:"url" yaml:"url"`
	Head          string `json:"head" yaml:"head"`
	Base          string `json:"base" yaml:"base"`
	Draft         bool   `json:"draft" yaml:"draft"`
	Title         string `json:"title" yaml:"title"`
	Body          string `json:"-" yaml:"-"`
	AutoMergeMode string `json:"auto_merge_mode,omitempty" yaml:"auto_merge_mode,omitempty"`
}

func NewClient(ctx context.Context, repo *Repo) (*Client, error) {
	token, err := ResolveToken(viper.GetString("token"))
	if err != nil {
		return nil, err
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	httpClient := oauth2.NewClient(ctx, src)
	rateLimiter := github_ratelimit.NewClient(httpClient.Transport)

	client := &Client{
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

func (c *Client) GetCommitURL(sha string) string {
	return fmt.Sprintf("https://github.com/%s/%s/commit/%s", c.repo.Owner, c.repo.Name, sha)
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
		err = ErrNoMatchingObject
	}

	return sha, err
}

type branchInfo struct {
	Name      string
	Commit    githubv4.GitObjectID
	CommitUrl githubv4.URI
}

type repositoryInfo struct {
	NodeID             string
	IsEmpty            bool
	AutoMergeAllowed   bool
	MergeCommitAllowed bool
	SquashMergeAllowed bool
	RebaseMergeAllowed bool
	DefaultBranch      branchInfo
	TargetBranch       branchInfo
}

// GetRepositoryInfo returns information about a repository
func (c *Client) GetRepositoryInfo(branch string) (repository repositoryInfo, err error) {
	var query struct {
		Repository struct {
			Id                 githubv4.String
			IsEmpty            githubv4.Boolean
			AutoMergeAllowed   githubv4.Boolean
			MergeCommitAllowed githubv4.Boolean
			SquashMergeAllowed githubv4.Boolean
			RebaseMergeAllowed githubv4.Boolean
			DefaultBranchRef   struct {
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
		NodeID:             string(query.Repository.Id),
		IsEmpty:            bool(query.Repository.IsEmpty),
		AutoMergeAllowed:   bool(query.Repository.AutoMergeAllowed),
		MergeCommitAllowed: bool(query.Repository.MergeCommitAllowed),
		SquashMergeAllowed: bool(query.Repository.SquashMergeAllowed),
		RebaseMergeAllowed: bool(query.Repository.RebaseMergeAllowed),
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

func (c *Client) GetRef(refName string) (*github.Reference, error) {
	ref, resp, err := c.V3.Git.GetRef(c.context, c.repo.Owner, c.repo.Name, refName)
	if err != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNoMatchingObject
	}
	return ref, err
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

type TagObj struct {
	Name   string
	Commit struct {
		SHA string
		URL string
	}
	Lightweight bool
	Object      struct {
		SHA     string
		Message string
	}
}

// GetTag resolves a tag reference, returning a pseudo-reference
func (c *Client) GetTagObj(name string) (tagObj *TagObj, err error) {
	refName, err := util.QualifiedRefName(name, "tags")
	if err != nil {
		return nil, fmt.Errorf("QualifiedRefName(%s, tags): %w", name, err)
	}

	var query struct {
		Repository struct {
			Ref *struct {
				Target struct {
					Typename     githubv4.String `graphql:"__typename"`
					CommitFields struct {
						Oid githubv4.GitObjectID
						Url githubv4.URI
					} `graphql:"... on Commit"`
					TagFields struct {
						Oid     githubv4.GitObjectID
						Message githubv4.String
						Target  struct {
							Typename githubv4.String `graphql:"__typename"`
							Commit   struct {
								Oid githubv4.GitObjectID
								Url githubv4.URI
							} `graphql:"... on Commit"`
						}
					} `graphql:"... on Tag"`
				}
			} `graphql:"ref(qualifiedName: $ref)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]any{
		"owner": githubv4.String(c.repo.Owner),
		"repo":  githubv4.String(c.repo.Name),
		"ref":   githubv4.String(refName),
	}

	tagObj = &TagObj{
		Name: refName,
	}

	err = c.V4.Query(c.context, &query, variables)
	if err != nil {
		return nil, err
	}

	if query.Repository.Ref == nil {
		return nil, ErrNoMatchingObject
	}

	switch tag := query.Repository.Ref.Target; tag.Typename {
	case "Commit":
		tagObj.Lightweight = true
		tagObj.Commit.SHA = string(tag.CommitFields.Oid)
		tagObj.Commit.URL = tag.CommitFields.Url.String()

	case "Tag":
		tagObj.Lightweight = false
		tagObj.Object.SHA = string(tag.TagFields.Oid)
		tagObj.Object.Message = string(tag.TagFields.Message)
		if tag.TagFields.Target.Typename == "Commit" {
			tagObj.Commit.SHA = string(tag.TagFields.Target.Commit.Oid)
			tagObj.Commit.URL = tag.TagFields.Target.Commit.Url.String()
		} else {
			return nil, fmt.Errorf("unsupported annotated tag type: %s", tag.TagFields.Target.Typename)
		}

	default:
		return nil, fmt.Errorf("unsupported tag type: %s", tag.Typename)
	}

	return tagObj, nil
}

func (c *Client) CreateTag(name, message, sha string) (*github.Tag, error) {
	tag := &github.Tag{
		Tag:     &name,
		Message: &message,
		Object: &github.GitObject{
			Type: github.Ptr("commit"),
			SHA:  github.Ptr(sha),
		},
	}
	log.Debugf("Tag: %+v", tag)
	tag, _, err := c.V3.Git.CreateTag(c.context, c.repo.Owner, c.repo.Name, tag)
	if err != nil {
		return nil, fmt.Errorf("CreateTag(%s, %s): %w", c.repo, name, err)
	}

	return tag, nil
}

func (c *Client) CreateRef(ref *github.Reference) (*github.Reference, error) {
	log.Infof("CreateRef(%s, %s)", c.repo, ref.String())
	ref, _, err := c.V3.Git.CreateRef(c.context, c.repo.Owner, c.repo.Name, ref)
	if err != nil {
		return nil, fmt.Errorf("CreateRef(%s, %s): %w", c.repo, ref.String(), err)
	}

	return ref, nil
}

func (c *Client) UpdateRef(ref *github.Reference, force bool) (*github.Reference, error) {
	log.Infof("UpdateRef(%s, %s, %v)", c.repo, ref.String(), force)
	ref, _, err := c.V3.Git.UpdateRef(c.context, c.repo.Owner, c.repo.Name, ref, force)
	if err != nil {
		return nil, fmt.Errorf("UpdateRef(%s, %s, %v): %w", c.repo, ref.String(), force, err)
	}

	return ref, nil
}

func (c *Client) DeleteRef(ref string) error {
	log.Infof("DeleteRef(%s, %s)", c.repo, ref)
	if _, err := c.V3.Git.DeleteRef(c.context, c.repo.Owner, c.repo.Name, ref); err != nil {
		return fmt.Errorf("DeleteRef(%s, %s): %w", c.repo, ref, err)
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
				Id        githubv4.ID
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

	// Enable auto-merge if requested
	if pullRequest.AutoMergeMode != AutoMergeOff {
		err = c.enableAutoMerge(mutation.CreatePullRequest.PullRequest.Id, pullRequest.AutoMergeMode)
		if err != nil {
			log.Warnf("failed to enable auto-merge for pull request #%d: %v", pullRequest.Number, err)
			// Don't fail the entire operation if auto-merge fails
			err = nil
		}
	}

	return
}

func (c *Client) enableAutoMerge(pullRequestId githubv4.ID, mergeMethod string) error {
	var mutation struct {
		EnablePullRequestAutoMerge struct {
			PullRequest struct {
				Id githubv4.ID
			}
		} `graphql:"enablePullRequestAutoMerge(input: $input)"`
	}

	var apiMergeMethod githubv4.PullRequestMergeMethod
	switch mergeMethod {
	case AutoMergeMerge:
		apiMergeMethod = githubv4.PullRequestMergeMethodMerge
	case AutoMergeSquash:
		apiMergeMethod = githubv4.PullRequestMergeMethodSquash
	case AutoMergeRebase:
		apiMergeMethod = githubv4.PullRequestMergeMethodRebase
	default:
		return fmt.Errorf("unsupported merge method: %s", mergeMethod)
	}

	input := githubv4.EnablePullRequestAutoMergeInput{
		PullRequestID: pullRequestId,
		MergeMethod:   &apiMergeMethod,
	}

	return c.V4.Mutate(c.context, &mutation, input, nil)
}

func (c *Client) UpdateRefName(refName string, targetRef *github.Reference, force bool, immutable bool) (oldHash string, newHash string, err error) {
	legacyRef, _, err := c.V3.Git.GetRef(c.context, c.repo.Owner, c.repo.Name, refName)
	if err != nil {
		// Ref doesn't exist yet - create it (immutable flag has no effect on creation)
		log.Infof("creating ref %q", refName)
		updatedRef, _, err := c.V3.Git.CreateRef(c.context, c.repo.Owner, c.repo.Name, targetRef)
		if err != nil {
			return "", "", err
		}
		return "", updatedRef.Object.GetSHA(), nil
	}

	// Check if immutable and ref has diverged
	// Note: immutable only prevents updates to existing refs that point to different commits
	existingHash := legacyRef.Object.GetSHA()
	proposedHash := targetRef.Object.GetSHA()

	if immutable && existingHash != proposedHash {
		log.Infof("skipping update of ref %q (immutable and diverged: %s -> %s)", refName, existingHash, proposedHash)
		return existingHash, proposedHash, &ErrImmutableRef{
			RefName:      refName,
			ExistingHash: existingHash,
			ProposedHash: proposedHash,
		}
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

func (r *repositoryInfo) IsAutoMergeMethodSupported(method string) bool {
	switch method {
	case AutoMergeOff:
		return true
	case AutoMergeMerge:
		return r.MergeCommitAllowed
	case AutoMergeSquash:
		return r.SquashMergeAllowed
	case AutoMergeRebase:
		return r.RebaseMergeAllowed
	default:
		return false
	}
}

func (r *repositoryInfo) GetSupportedAutoMergeMethods() []string {
	var methods []string
	methods = append(methods, AutoMergeOff) // Always supported

	if r.MergeCommitAllowed {
		methods = append(methods, AutoMergeMerge)
	}
	if r.SquashMergeAllowed {
		methods = append(methods, AutoMergeSquash)
	}
	if r.RebaseMergeAllowed {
		methods = append(methods, AutoMergeRebase)
	}

	return methods
}
