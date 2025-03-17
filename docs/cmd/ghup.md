# ghup

Update GitHub content and tags via API

## Synopsis

The `ghup` tool enables managing GitHub repository content and tags directly via the GitHub API,
ensuring verified commits from CI systems even without git repo access.

```
ghup [flags]
```

## Description

`ghup` provides commands to manage repository content and tags via the GitHub API. This is particularly useful in CI/CD environments where you need to make commits with proper verification status, even without direct Git repository checkout access.

Key features:
- Create or update content (files) in a repository
- Create or update lightweight and annotated tags
- Resolve commit references to SHAs
- Open pull requests for changes
- Support for JSON or YAML output formats

## Options

```
      --config-path strings   configuration paths (default [.])
  -C, --config-name string    configuration name
      --token string          GitHub Token or path/to/token-file
  -o, --owner string          repository owner name
  -r, --repo string           repository name
      --no-cli-token          disable fallback to GitHub CLI Token
  -v, --verbose               increase verbosity
  -O, --output-format string  output format (json|j, yaml|y) (default "json")
      --compact               compact output
  -h, --help                  help for ghup
      --version               version for ghup
```

## Environment Variables

The following environment variables can be used instead of command-line flags:

- `GHUP_TOKEN`, `GH_TOKEN`, `GITHUB_TOKEN` - GitHub token for API authentication
- `GHUP_OWNER`, `GITHUB_OWNER`, `GITHUB_REPOSITORY_OWNER` - Repository owner
- `GHUP_REPO`, `GITHUB_REPO`, `GITHUB_REPOSITORY_NAME` - Repository name
- `GHUP_BRANCH`, `CHANGE_BRANCH`, `BRANCH_NAME`, `GIT_BRANCH` - Default branch name

## Commands

- [content](ghup_content.md) - Manage repository content
- [tag](ghup_tag.md) - Create or update lightweight or annotated tags
- [resolve](ghup_resolve.md) - Resolve a commit-ish to a SHA
- [update-ref](ghup_update-ref.md) - Update target refs to match source commitish
- [debug](ghup_debug.md) - Dump contextual information to aid debugging

## Examples

```bash
# Create a file in a repository
ghup content --branch my-feature --update local-file.txt:remote-path.txt

# Create or update a tag pointing to a commit or branch
ghup tag v1.0.0 --commitish main

# Resolve a branch name to a commit SHA
ghup resolve main

# Create a file and open a pull request
ghup content --branch feature-branch --update file.txt --pr-title "Add new file"
```
