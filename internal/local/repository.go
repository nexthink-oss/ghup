package local

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/apex/log"
	giturls "github.com/chainguard-dev/git-urls"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/format/index"
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

type PathContent map[string][]byte
type DeletionSet map[string]struct{}

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

	remoteName := "origin" // default remote name
	r.Branch = "main"      // default branch name

	// Try to parse the GIT_BRANCH environment variable set by CI/CD systems
	if parts := strings.SplitN(os.Getenv("GIT_BRANCH"), "/", 2); len(parts) == 2 {
		remoteName = parts[0]
		r.Branch = parts[1]
	} else if head.Name().IsBranch() {
		branchName := head.Name().Short()
		r.Branch = branchName

		if branch, err := repo.Branch(branchName); err == nil {
			remoteName = cmp.Or(branch.Remote, "origin")
		}
	}

	// Try to parse the GIT_URL environment variable set by CI/CD systems
	if owner, name, ok := parseRemote(os.Getenv("GIT_URL")); ok {
		r.Owner = owner
		r.Name = name
	} else if remote, err := repo.Remote(remoteName); err == nil {
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
	if r.Repository != nil {
		head, err := r.Repository.Head()
		if err == nil {
			hash = head.Hash().String()
		}
	}
	return
}

func (r *Repository) Status() (status git.Status, err error) {
	if r.Repository != nil {
		worktree, err := r.Repository.Worktree()
		if err != nil {
			return nil, err
		}
		return worktree.StatusWithOptions(git.StatusOptions{Strategy: git.Preload})
	}
	return nil, nil
}

// Tracked returns all changes to tracked files in the repository.
// If the file is added or modified, the contents are read from the filesystem.
func (r *Repository) Tracked(gitStatus git.Status) (
	pathContent PathContent,
	deletionSet DeletionSet,
	err error,
) {
	if r.Repository == nil {
		return nil, nil, fmt.Errorf("repository not initialized")
	}

	pathContent = make(PathContent)
	deletionSet = make(DeletionSet)
	errs := make([]error, 0)

	for path, status := range gitStatus {
		log.Debugf("%c%c %s\n", status.Staging, status.Worktree, path)
		switch {
		case status.Staging == git.Added,
			status.Staging == git.Modified,
			status.Worktree == git.Modified:
			contents, err := os.ReadFile(path)
			if err != nil {
				errs = append(errs, fmt.Errorf("ReadFile(%s): %w", path, err))
				continue
			}
			pathContent[path] = contents
		case status.Worktree == git.Deleted:
			// if updated in staging area, but deleted in worktree, it's a deletion
			delete(pathContent, path)
			fallthrough
		case status.Staging == git.Deleted:
			deletionSet[path] = struct{}{}
		}
	}

	return pathContent, deletionSet, errors.Join(errs...)
}

// Staged returns all changes to staged files in the repository.
// If the file is added or modified, the contents are read from the git index.
func (r *Repository) Staged(gitStatus git.Status) (
	pathContent PathContent,
	deletionSet DeletionSet,
	err error,
) {
	if r.Repository == nil {
		return nil, nil, fmt.Errorf("repository not initialized")
	}

	index, err := r.Repository.Storer.Index()
	if err != nil {
		return nil, nil, fmt.Errorf("repository storer index: %w", err)
	}

	pathContent = make(PathContent)
	deletionSet = make(DeletionSet)
	errs := make([]error, 0)

	for path, status := range gitStatus {
		log.Debugf("%c%c %s\n", status.Staging, status.Worktree, path)
		switch status.Staging {
		case git.Added, git.Modified:
			// get content of the file from the git index
			content, err := r.contentForIndexPath(index, path)
			if err != nil {
				errs = append(errs, fmt.Errorf("ContentForIndexPath(%s): %w", path, err))
				continue
			}
			pathContent[path] = content
		case git.Deleted:
			deletionSet[path] = struct{}{}
		}
	}

	return pathContent, deletionSet, errors.Join(errs...)
}

func (r *Repository) contentForIndexPath(idx *index.Index, path string) (content []byte, err error) {
	entry, err := idx.Entry(path)
	if err != nil {
		return nil, fmt.Errorf("finding index entry for %q: %w", path, err)
	}

	// Retrieve the blob object from the index entry’s Hash
	blob, err := r.Repository.BlobObject(entry.Hash)
	if err != nil {
		return nil, fmt.Errorf("finding blob for %q: %w", path, err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("reading blob for %q: %w", path, err)
	}
	defer func() { _ = reader.Close() }()

	return io.ReadAll(reader)
}

func (p PathContent) Keys() []string {
	return slices.Sorted(maps.Keys(p))
}

func (d DeletionSet) Keys() []string {
	return slices.Sorted(maps.Keys(d))
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
