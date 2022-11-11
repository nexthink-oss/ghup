package gitutil

import (
	"strings"

	"github.com/go-git/go-git/v5"
	giturls "github.com/whilp/git-urls"
)

type GithubRepo struct {
	Owner      string
	Repository string
	Branch     string
}

func GithubRepoFromPath(path string) (*GithubRepo, bool) {
	options := &git.PlainOpenOptions{
		DetectDotGit: true,
	}
	repo, err := git.PlainOpenWithOptions(path, options)
	if err != nil {
		return nil, false
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, false
	}

	for _, remote := range remotes {
		remoteConfig := *remote.Config()
		if o, r, ok := parseRemote(remoteConfig.URLs[0]); ok {
			gh := &GithubRepo{
				Owner:      o,
				Repository: r,
				Branch:     "main",
			}
			head, err := repo.Head()
			if err == nil {
				gh.Branch = head.Name().Short()
			}
			return gh, true
		}
	}
	return nil, false
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
