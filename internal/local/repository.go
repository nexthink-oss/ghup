package local

import (
	"cmp"
	"strings"

	giturls "github.com/chainguard-dev/git-urls"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

type Repository struct {
	Repository *git.Repository `json:"-"`
	Owner      string          `json:"owner"`
	Name       string          `json:"name"`
	Path       string          `json:"path" default:"."`
	Branch     string          `json:"branch" default:"main"`
	User       struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
}

// SetDefaults implements defaults.Setter interface
func (r *Repository) SetDefaults() {
	options := &git.PlainOpenOptions{DetectDotGit: true}
	repo, err := git.PlainOpenWithOptions(r.Path, options)
	if err != nil {
		return
	}

	r.Repository = repo

	head, err := repo.Head()
	if err != nil {
		return
	}

	branchName := cmp.Or[string](head.Name().Short(), "main")
	branch, err := repo.Branch(branchName)
	if err != nil {
		return
	}

	r.Branch = branchName

	remoteName := cmp.Or[string](branch.Remote, "origin")

	remote, err := repo.Remote(remoteName)
	if err == nil {
		remoteConfig := *remote.Config()
		if owner, name, ok := parseRemote(remoteConfig.URLs[0]); ok {
			r.Owner = owner
			r.Name = name
		}
	}

	config, err := repo.ConfigScoped(config.GlobalScope)
	if err == nil {
		r.User.Name = config.User.Name
		r.User.Email = config.User.Email
	}
}

func (r *Repository) HeadCommit() (hash string) {
	if r.Repository == nil {
		head, err := r.Repository.Head()
		if err == nil {
			hash = head.Hash().String()
		}
	}
	return
}

func (r *Repository) Status() (status git.Status, err error) {
	if r.Repository == nil {
		worktree, err := r.Repository.Worktree()
		if err != nil {
			return nil, err
		}
		return worktree.Status()
	}
	return nil, nil
}

func parseRemote(remote string) (owner string, repo string, ok bool) {
	url, err := giturls.Parse(remote)
	if err != nil {
		return
	}

	pathComponents := strings.Split(strings.TrimPrefix(url.Path, "/"), "/")

	switch {
	case url.Host != "github.com":
		return
	case len(pathComponents) != 2:
		return
	default:
		owner = pathComponents[0]
		repo = strings.TrimSuffix(pathComponents[1], ".git")
		ok = true
		return
	}
}
