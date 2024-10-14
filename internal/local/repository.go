package local

import (
	"strings"

	giturls "github.com/chainguard-dev/git-urls"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

type Repository struct {
	Repository *git.Repository
	Owner      string
	Name       string
	Branch     string
	UserName   string
	UserEmail  string
}

func GetRepository(path string) (ghr *Repository) {
	options := &git.PlainOpenOptions{
		DetectDotGit: true,
	}
	repo, err := git.PlainOpenWithOptions(path, options)
	if err != nil {
		return
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return
	}

	for _, remote := range remotes {
		remoteConfig := *remote.Config()
		if o, r, ok := parseRemote(remoteConfig.URLs[0]); ok {
			ghr = &Repository{
				Repository: repo,
				Owner:      o,
				Name:       r,
				Branch:     "main",
			}
			head, err := repo.Head()
			if err == nil {
				ghr.Branch = head.Name().Short()
			}
			config, err := repo.ConfigScoped(config.GlobalScope)
			if err == nil {
				ghr.UserName = config.User.Name
				ghr.UserEmail = config.User.Email
			}
			return
		}
	}
	return
}

func (r *Repository) HeadCommit() (hash string) {
	head, err := r.Repository.Head()
	if err == nil {
		hash = head.Hash().String()
	}
	return
}

func (r *Repository) Status() (status git.Status, err error) {
	worktree, err := r.Repository.Worktree()
	if err != nil {
		return nil, err
	}
	return worktree.Status()
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
