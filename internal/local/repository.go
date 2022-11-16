package local

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	giturls "github.com/whilp/git-urls"
)

type Repository struct {
	Repository *git.Repository
	Owner      string
	Name       string
	Branch     string
	User       string
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
				ghr.User = fmt.Sprintf("%s <%s>", config.User.Name, config.User.Email)
			}
			return
		}
	}
	return
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
