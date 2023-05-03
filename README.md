[![CodeQL](https://github.com/nexthink-oss/ghup/actions/workflows/codeql.yml/badge.svg)](https://github.com/nexthink-oss/ghup/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nexthink-oss/ghup)](https://goreportcard.com/report/github.com/nexthink-oss/ghup)

# ghup

A GitHub API client for managing tags and repository content from third-party automation systems (e.g. Jenkins).

## Features

* Create and update tags, both lightweight and annotated.
* Add, update and delete content idempotently.
* GitHub-verified commits (when using a GitHub App-derived token), facilitating the enforcement of commit signing.
* Configuration defaults inferred from local context (e.g. git clone and environment).
* Completely self-contained: no external dependencies.

## Requirements

* A GitHub Token, preferably derived from GitHub App credentials, with `contents=write` and `metadata=read` permissions on target repository.

Note: works well with [vault-plugin-secrets-github](https://github.com/martinbaillie/vault-plugin-secrets-github)!

## Configuration

If the current working directory is a git repository, the first GitHub remote (if there is one) is used to infer default repository owner (`--owner`) and name (`--repo`), the current branch is used to set the default branch (`--branch`), and resolved git config is used to set a default author for generated `Signed-off-by` message suffix (`--trailer.name` and `--trailer.email`) to help distinguish between different systems sharing common GitHub App credentials.

If run outside a GitHub repository, then the `--owner` and `--repo` flags are required, with `--branch` defaulting to `main`.

All configuration may be passed via environment variable rather than flag. The environment variable associated with each flag is `GHUP_[UPPERCASED_FLAG_NAME]`, e.g. `GHUP_TOKEN`, `GHUP_OWNER`, `GHUP_REPO`, `GHUP_BRANCH`, `GHUP_TRAILER_KEY`, etc.

In addition, various fallback environment variables are supported for better integration with Jenkins and similar CI tools: `GITHUB_OWNER`, `GITHUB_TOKEN`, `CHANGE_BRANCH`, `BRANCH_NAME`, `GIT_BRANCH`, `GIT_COMMITTER_NAME`, `GIT_COMMITTER_EMAIL`, etc.

For security, it is strongly recommended that the GitHub Token by passed via environment (`GHUP_TOKEN` or `GITHUB_TOKEN`) or, better, file path (`--token /path/to/token-file`)

## Usage

### Tagging

```console
$ ghup tag --help

Manage tags via the GitHub V3 API

Usage:
  ghup tag [flags] [<name>]

Flags:
  -h, --help          help for tag
      --lightweight   force lightweight tag
      --tag string    tag name

Global Flags:
  -b, --branch string          branch name (default "[local-branch-or-main]")
  -f, --force                  force action
  -m, --message string         message (default "Commit via API")
  -o, --owner string           repository owner (default "[owner-of-first-github-remote-or-required]")
  -r, --repo string            repository name (default "[repo-of-first-github-remote-or-required]")
      --token string           GitHub Token or path/to/token-file
      --trailer.email string   email for commit trailer (default "[user.email]")
      --trailer.key string     key for commit trailer (blank to disable) (default "Co-Authored-By")
      --trailer.name string    name for commit trailer (default "[user.name]")
  -v, --verbosity count        verbosity
```

Note: annotated tags are the default, but only lightweight tags (i.e. `--lightweight`), which simply point at an existing commit, are "verified".

#### Tagging Example

Tag the current repo with `v1.0`:

```console
$ ghup tag v1.0 -m "Release v1.0!"
https://github.com/nexthink-oss/ghup/releases/tag/v1.0
```

### Content

```console
$ ghup content --help

Manage content via the GitHub V4 API

Usage:
  ghup content [flags] [<file-spec> ...]

Flags:
      --base-branch string   base branch name
      --create-branch        create missing target branch (default true)
  -d, --delete strings       file-path to delete
  -h, --help                 help for content
  -s, --separator string     file-spec separator (default ":")
  -u, --update strings       file-spec to update

Global Flags:
  -b, --branch string          branch name (default "[local-branch-or-main]")
  -f, --force                  force action
  -m, --message string         message (default "Commit via API")
  -o, --owner string           repository owner (default "[owner-of-first-github-remote-or-required]")
  -r, --repo string            repository name (default "[repo-of-first-github-remote-or-required]")
      --token string           GitHub Token or path/to/token-file
      --trailer.email string   email for commit trailer (default "[user.email]")
      --trailer.key string     key for commit trailer (blank to disable) (default "Co-Authored-By")
      --trailer.name string    name for commit trailer (default "[user.name]")
  -v, --verbosity count        verbosity
```

Each `file-spec` provided as a positional argument or explicitly via the `--update` flag takes the form `<local-file-path>[:<remote-target-path>]`. Content is read from the local file `<local-file-path>` and written to `<remote-target-path>` (defaulting to `<local-file-path>` if not specified).

Each `file-path` provided to the `--delete` flag is a `<remote-target-path>`: the path to a file on the target repository:branch that should be deleted.

Unless `--force` is used, content that already matches the remote repository state is ignored.

#### Content Example

Update `.zshrc` in my `dotfiles` repo, adding if 's missing and updating if-and-only-if changed:

```console
$ ghup content --owner=isometry --repo=dotfiles ~/.zshrc:.zshrc -m "Update zshrc"
https://github.com/isometry/dotfiles/commit/15b8630c81a051c2b128c94e5796c5d9c2bc8846
$ ghup content --owner=isometry --repo=dotfiles ~/.zshrc:.zshrc -m "Update zshrc"
nothing to do
```

Delete `.tcshrc` from my `dotfiles` repo:

```console
$ ghup content --owner=isometry --repo=dotfiles --delete .tcshrc -m "Remove tcshrc"
https://github.com/isometry/dotfiles/commit/bf120a96c65cb482eacc3c9e27d2d0935d108eca
$ ghup content --owner=isometry --repo=dotfiles --delete .tcshrc -m "Remove tcshrc"
nothing to do
```
