[![CodeQL](https://github.com/nexthink-oss/ghup/actions/workflows/codeql.yml/badge.svg)](https://github.com/nexthink-oss/ghup/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nexthink-oss/ghup)](https://goreportcard.com/report/github.com/nexthink-oss/ghup)

# ghup

A GitHub API client for managing tags and repository content from third-party automation systems (e.g. Jenkins).

## Features

* Create and update tags, both lightweight and annotated.
* Add, update and delete content idempotently.
* Idempotent update of git refs (e.g. fast-forward merge and mutable tags).
* GitHub-verified commits (when using a GitHub App-derived token), facilitating the enforcement of commit signing.
* Configuration defaults inferred from local context (e.g. git clone and environment).
* Completely self-contained: no external dependencies.

## Requirements

* A GitHub Token, preferably derived from GitHub App credentials (for verified commits), with `contents=write` and `metadata=read` permissions on target repository. In addition, `workflows=write` is also needed if used to manage `.github/workflows` content.

Note: works well with [vault-plugin-secrets-github](https://github.com/martinbaillie/vault-plugin-secrets-github)!

## Configuration

If the current working directory is a git repository, its first GitHub remote (if there is one) is used to infer default repository owner (`--owner`) and name (`--repo`), the current branch is used to set the default branch (`--branch`), and resolved git config is used to set a default author for a generated `Co-Authored-By` commit message trailer to help distinguish between different systems sharing common GitHub App credentials (override components with `--author.trailer`, `--user.name` and `--user.email`, or disable with `--author.trailer=` or `export GHUP_AUTHOR_TRAILER=`). Additional commit trailers can be specified with `--trailer key=value` flags.

If run outside a GitHub repository, then the `--owner` and `--repo` flags are required, with `--branch` defaulting to `main`.

All configuration may be passed via environment variable rather than flag. The environment variable associated with each flag is `GHUP_[UPPERCASED_FLAG_NAME]`, e.g. `GHUP_TOKEN`, `GHUP_OWNER`, `GHUP_REPO`, `GHUP_BRANCH`, `GHUP_AUTHOR_TRAILER`, etc.

In addition, various fallback environment variables are supported for better integration with Jenkins and similar CI tools: `GITHUB_OWNER`, `GITHUB_TOKEN`, `CHANGE_BRANCH`, `BRANCH_NAME`, `GIT_BRANCH`, `GIT_COMMITTER_NAME`, `GIT_COMMITTER_EMAIL`, etc.

For security, it is strongly recommended that the GitHub Token by passed via environment (`GHUP_TOKEN` or `GITHUB_TOKEN`) or file path (`--token /path/to/token-file`, `--token <(gh auth token)` or `export GHUP_TOKEN=/path/to/token-file ghup …`)

## Usage

### Content

The `content` verb is used to generate arbitrary (verified) commits via the GitHub V4 API.
An arbitrary number of content adds, removes and deletes can be committed without the need for a local signing key or for checking out the target repository.

```console
$ ghup content --help

Manage content via the GitHub V4 API

Usage:
  ghup content [flags] [<file-spec> ...]

Flags:
      --base-branch string   base branch name (default: "[remote-default-branch]")
      --create-branch        create missing target branch (default true)
  -d, --delete strings       file-path to delete
  -h, --help                 help for content
      --pr-body string       pull request description body
      --pr-draft             create pull request in draft mode
      --pr-title string      create pull request iff target branch is created and title is specified
  -s, --separator string     file-spec separator (default ":")
  -u, --update strings       file-spec to update

Global Flags:
      --author.trailer string    key for commit author trailer (blank to disable) (default "Co-Authored-By")
  -b, --branch string            target branch name (default "feature/ref")
  -f, --force                    force action
  -m, --message string           message (default "Commit via API")
  -o, --owner string             repository owner (default "isometry")
  -r, --repo string              repository name (default "ghup")
      --token string             GitHub Token or path/to/token-file
      --trailer stringToString   additional commit trailer (key=value; JSON via environment) (default [])
      --user.email string        email for commit author trailer (default "robin@isometry.net")
      --user.name string         name for commit author trailer (default "Robin Breathe")
  -v, --verbosity count          verbosity
```

Each `file-spec` provided as a positional argument or explicitly via the `--update` flag takes the form `<local-file-path>[:<remote-target-path>]`. Content is read from the local file `<local-file-path>` and written to `<remote-target-path>` (defaulting to `<local-file-path>` if not specified).

Each `file-path` provided to the `--delete` flag is a `<remote-target-path>`: the path to a file on the target repository:branch that should be deleted.

Unless `--force` is used, content that already matches the remote repository state is ignored.

Note: Due to limitations in the GitHub V4 API, when the target branch does not exist, branch creation and content push will trigger two distinct "push" events.

#### Content Examples

##### Idempotent file add/update

Update `.zshrc` in my `dotfiles` repo, adding if's missing and updating if-and-only-if changed:

```console
$ ghup content --owner=isometry --repo=dotfiles ~/.zshrc:.zshrc -m "chore: update zshrc"
https://github.com/isometry/dotfiles/commit/15b8630c81a051c2b128c94e5796c5d9c2bc8846
$ ghup content --owner=isometry --repo=dotfiles ~/.zshrc:.zshrc -m "chore: update zshrc"
nothing to do
```

##### Idempotent file deletion

Delete `.tcshrc` from my `dotfiles` repo:

```console
$ ghup content --owner=isometry --repo=dotfiles --delete .tcshrc -m "chore: remove tcshrc"
https://github.com/isometry/dotfiles/commit/bf120a96c65cb482eacc3c9e27d2d0935d108eca
$ ghup content --owner=isometry --repo=dotfiles --delete .tcshrc -m "chore: remove tcshrc"
nothing to do
```

### Tagging

The `tag` verb is used to create lightweight or annotated tags without the need to checkout the target repository.

Note: annotated tags are the default, but only lightweight tags (i.e. `--lightweight`), which simply point at an existing commit, are "verified".

```console
$ ghup tag --help

Manage tags via the GitHub V3 API

Usage:
  ghup tag [flags] [<name>]

Flags:
  -h, --help          help for tag
  -l, --lightweight   force lightweight tag
      --tag string    tag name

Global Flags:
      --author.trailer string    key for commit author trailer (blank to disable) (default "Co-Authored-By")
  -b, --branch string            target branch name (default "feature/ref")
  -f, --force                    force action
  -m, --message string           message (default "Commit via API")
  -o, --owner string             repository owner (default "isometry")
  -r, --repo string              repository name (default "ghup")
      --token string             GitHub Token or path/to/token-file
      --trailer stringToString   additional commit trailer (key=value; JSON via environment) (default [])
      --user.email string        email for commit author trailer (default "robin@isometry.net")
      --user.name string         name for commit author trailer (default "Robin Breathe")
  -v, --verbosity count          verbosity
```

#### Tagging Examples

##### Create lightweight tag

Create lightweight tag `v1.0.0` pointing at the head of the local repository's checked out branch:

```console
$ ghup tag v1.0.0
https://github.com/nexthink-oss/ghup/releases/tag/v1.0.0
```

##### Create annotated tag

Create an annotated repo `v1.0` pointed at the head of the `main` branch of the `ghup` repo owned by `nexthink-oss`:

```console
$ ghup -o nexthink-oss -r ghup -b main tag v1.0 -m "Release v1.0!"
https://github.com/nexthink-oss/ghup/releases/tag/v1.0
```

### Update Refs

The `update-ref` verb is used to update an arbitrary number `head` or `tag` references to match a source reference.

The `source` may take the form of a partial commit hash, or of a fully- or partially-qualified reference, defaulting to a branch reference (`heads/…`; overrideable via `--source-type=tags`).
The `target`(s) must take the form of fully- or partially-qualified references, defaulting to tag references, defaulting to tag references (`tags/…`; overrideable via `--target-type=heads`).
The `--force` flag will override standard fast-forward-only protection on branch updates.

```console
$ ghup update-ref --help
Update target refs to match source

Usage:
  ghup update-ref [flags] -s <source> <target> ...

Flags:
  -s, --source ref-or-commit     source ref-or-commit
  -S, --source-type heads|tags   unqualified source ref type (default heads)
  -T, --target-type heads|tags   unqualified target ref type (default tags)
  -h, --help                     help for update-ref

Global Flags:
      --author.trailer key   key for commit author trailer (blank to disable) (default "Co-Authored-By")
  -b, --branch name          target branch name (default "feature/ref")
  -f, --force                force action
  -m, --message string       message (default "Commit via API")
  -o, --owner name           repository owner name (default "isometry")
  -r, --repo name            repository name (default "ghup")
      --token string         GitHub Token or path/to/token-file
      --trailer key=value    extra key=value commit trailers (default [])
      --user.email email     email for commit author trailer (default "robin.breathe@nexthink.com")
      --user.name name       name for commit author trailer (default "Robin Breathe")
  -v, --verbosity count      verbosity
```

Note: the `--branch`, `--message` and trailer-related flags are not used by the `ref` verb.

#### Updated Refs Examples

##### Fast-forward production branch to match staging

```console
$ ghup update-ref -s staging heads/production
source:
  ref: heads/staging
  sha: 206e1a484f03cd320a2125a50aa73bd8a2b045dc
target:
  - ref: heads/production
    updated: true
    old_sha: b7ccc4db9bc43551fd3571c260869f4c69aa2fd4
    sha: 206e1a484f03cd320a2125a50aa73bd8a2b045dc
```

##### Create a lightweight tag pointing at a specific commit

```console
$ ghup update-ref -s b7ccc4d example
source:
  ref: b7ccc4d
  sha: b7ccc4db9bc43551fd3571c260869f4c69aa2fd4
target:
  - ref: tags/example
    updated: true
    sha: b7ccc4db9bc43551fd3571c260869f4c69aa2fd4
```

##### Update GitHub Actions-style major and minor tags following patch release:

```console
$ ghup update-ref -s tags/v1.1.7 v1.1 v1
source:
  ref: tags/v1.1.7
  sha: b7ccc4db9bc43551fd3571c260869f4c69aa2fd4
target:
  - ref: tags/v1.1
    updated: true
    sha: b7ccc4db9bc43551fd3571c260869f4c69aa2fd4
  - ref: tags/v1
    updated: true
    sha: b7ccc4db9bc43551fd3571c260869f4c69aa2fd4
```

### Debug Info

In order to better validate the configuration derived from context (working directory, environment variables and global flags), the `info` verb is available:

```console
$ ghup info
hasToken: true
trailers:
  - 'Co-Authored-By: Example User <user@example.com>'
owner: nexthink-oss
repository: ghup
branch: feature/branch
commit: 5e1692253399bd9ea6077dba27e4cdc8a15b9720
isClean: false
commitMessage:
  headline: Commit via API
  body: |2-
    Co-Authored-By: Example User <user@example.com>
```
