package local

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestParseRemote(t *testing.T) {
	tests := []struct {
		name      string
		remote    string
		wantOwner string
		wantRepo  string
		wantOk    bool
	}{
		// SSH format with git@ prefix
		{
			name:      "SSH format with git@ prefix",
			remote:    "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantOk:    true,
		},
		{
			name:      "SSH format without .git suffix",
			remote:    "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantOk:    true,
		},

		// HTTPS format
		{
			name:      "HTTPS format with .git suffix",
			remote:    "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantOk:    true,
		},
		{
			name:      "HTTPS format without .git suffix",
			remote:    "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantOk:    true,
		},

		// SSH protocol format
		{
			name:      "SSH protocol format with .git suffix",
			remote:    "ssh://git@github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantOk:    true,
		},
		{
			name:      "SSH protocol format without .git suffix",
			remote:    "ssh://git@github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantOk:    true,
		},

		// Edge cases with special characters in names
		{
			name:      "Owner and repo with hyphens and dots",
			remote:    "git@github.com:some-owner.dev/my-repo.name.git",
			wantOwner: "some-owner.dev",
			wantRepo:  "my-repo.name",
			wantOk:    true,
		},
		{
			name:      "Owner and repo with underscores",
			remote:    "https://github.com/owner_name/repo_name.git",
			wantOwner: "owner_name",
			wantRepo:  "repo_name",
			wantOk:    true,
		},

		// Invalid cases - non-GitHub hosts
		{
			name:   "GitLab URL should fail",
			remote: "git@gitlab.com:owner/repo.git",
			wantOk: false,
		},
		{
			name:   "Bitbucket URL should fail",
			remote: "https://bitbucket.org/owner/repo.git",
			wantOk: false,
		},
		{
			name:   "Custom Git server should fail",
			remote: "git@git.example.com:owner/repo.git",
			wantOk: false,
		},

		// Invalid cases - wrong path structure
		{
			name:   "Too many path components",
			remote: "https://github.com/owner/repo/extra.git",
			wantOk: false,
		},
		{
			name:   "Too few path components",
			remote: "https://github.com/owner.git",
			wantOk: false,
		},
		{
			name:   "Empty path",
			remote: "https://github.com/",
			wantOk: false,
		},
		{
			name:   "Root path only",
			remote: "https://github.com",
			wantOk: false,
		},

		// Invalid cases - malformed URLs
		{
			name:   "Invalid URL format",
			remote: "not-a-url",
			wantOk: false,
		},
		{
			name:   "Empty string",
			remote: "",
			wantOk: false,
		},
		{
			name:   "Just protocol",
			remote: "https://",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo, gotOk := parseRemote(tt.remote)

			if gotOk != tt.wantOk {
				t.Errorf("parseRemote() gotOk = %v, want %v", gotOk, tt.wantOk)
				return
			}

			if !tt.wantOk {
				// For invalid cases, we don't care about owner/repo values
				return
			}

			if gotOwner != tt.wantOwner {
				t.Errorf("parseRemote() gotOwner = %v, want %v", gotOwner, tt.wantOwner)
			}

			if gotRepo != tt.wantRepo {
				t.Errorf("parseRemote() gotRepo = %v, want %v", gotRepo, tt.wantRepo)
			}
		})
	}
}

// TestParseRemote_Examples provides some real-world examples
func TestParseRemote_Examples(t *testing.T) {
	realWorldExamples := []struct {
		name      string
		remote    string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "go-git repository",
			remote:    "https://github.com/go-git/go-git.git",
			wantOwner: "go-git",
			wantRepo:  "go-git",
		},
		{
			name:      "Kubernetes repository",
			remote:    "git@github.com:kubernetes/kubernetes.git",
			wantOwner: "kubernetes",
			wantRepo:  "kubernetes",
		},
		{
			name:      "Docker repository",
			remote:    "ssh://git@github.com/docker/docker.git",
			wantOwner: "docker",
			wantRepo:  "docker",
		},
	}

	for _, tt := range realWorldExamples {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo, gotOk := parseRemote(tt.remote)

			if !gotOk {
				t.Errorf("parseRemote() should have succeeded for %s", tt.remote)
				return
			}

			if gotOwner != tt.wantOwner {
				t.Errorf("parseRemote() gotOwner = %v, want %v", gotOwner, tt.wantOwner)
			}

			if gotRepo != tt.wantRepo {
				t.Errorf("parseRemote() gotRepo = %v, want %v", gotRepo, tt.wantRepo)
			}
		})
	}
}

// TestRepository_LinkedWorktree verifies that ghup works from a linked git
// worktree (created with `git worktree add`), where the .git entry is a file
// pointing into the parent repository's .git/worktrees/<name> directory and
// most repository data (config, refs, objects) lives in the common directory.
func TestRepository_LinkedWorktree(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git binary not available")
	}

	tmp := t.TempDir()
	mainDir := filepath.Join(tmp, "main")
	worktreeDir := filepath.Join(tmp, "wt")

	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	if err := os.Mkdir(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(mainDir, "init", "--initial-branch=main")
	git(mainDir, "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
	for name, content := range map[string]string{
		"staged.txt":    "original\n",
		"untouched.txt": "untouched\n",
	} {
		if err := os.WriteFile(filepath.Join(mainDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(mainDir, "add", ".")
	git(mainDir, "commit", "--no-gpg-sign", "-m", "initial")
	git(mainDir, "worktree", "add", worktreeDir, "-b", "feature")

	// stage a change in the linked worktree
	if err := os.WriteFile(filepath.Join(worktreeDir, "staged.txt"), []byte("updated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(worktreeDir, "add", "staged.txt")

	repo := &Repository{Path: worktreeDir}
	repo.SetDefaults()

	if repo.Repository == nil {
		t.Fatal("SetDefaults() failed to open linked worktree")
	}
	if repo.Branch != "feature" {
		t.Errorf("Branch = %q, want %q", repo.Branch, "feature")
	}
	if repo.Owner != "test-owner" || repo.Name != "test-repo" {
		t.Errorf("Owner/Name = %q/%q, want test-owner/test-repo (remote lookup requires the common directory's config)", repo.Owner, repo.Name)
	}

	status, err := repo.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}

	pathContent, deletionSet, err := repo.Staged(status)
	if err != nil {
		t.Fatalf("Staged() error: %v", err)
	}
	if len(deletionSet) != 0 {
		t.Errorf("Staged() deletionSet = %v, want empty", deletionSet.Keys())
	}
	if want := []string{"staged.txt"}; !slices.Equal(pathContent.Keys(), want) {
		t.Errorf("Staged() paths = %v, want %v (unstaged tracked files must not be reported)", pathContent.Keys(), want)
	}
	if got := string(pathContent["staged.txt"]); got != "updated\n" {
		t.Errorf("Staged() content = %q, want %q", got, "updated\n")
	}
}
