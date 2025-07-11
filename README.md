[![CodeQL](https://github.com/nexthink-oss/ghup/actions/workflows/codeql.yml/badge.svg)](https://github.com/nexthink-oss/ghup/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nexthink-oss/ghup)](https://goreportcard.com/report/github.com/nexthink-oss/ghup)

# ghup

`ghup` is a command-line tool for managing GitHub repository content and refs (branches and tags) directly via the GitHub API, with a focus on enabling verified commits from build systems such as GitHub Actions or Jenkins.

## Key Features

- Create, update, and delete repository content with verified commits via GitHub API
- Create and update both lightweight and annotated tags
- Synchronise arbitrary git refs, including fast-forward merges
- Resolve commit references to full SHAs
- Open pull requests for changes
- Smart context detection for repository and branch information
- 12-Factor app style configuration via flags, environment variables, or files
- No external dependencies required

## Requirements

- A GitHub token with `contents=write` and `metadata=read` permissions (plus `workflows=write` if managing GitHub workflows)
- For verified commits, use a token derived from GitHub App credentials

## Installation

### Pre-built Binaries

Download binaries for all platforms from [GitHub Releases](https://github.com/nexthink-oss/ghup/releases/latest).

### Using Go

```sh
go install github.com/nexthink-oss/ghup@latest
```

### Homebrew

```sh
brew install isometry/tap/ghup
```

### GitHub Actions

```yaml
- uses: nexthink-oss/ghup/actions/setup@v1
```

## Configuration

`ghup` can be configured through:

1. Command-line flags
2. Environment variables (`GHUP_*`, with various fallbacks for CI tools)
3. Configuration files (various formats supported)

When run from a git repository, `ghup` automatically detects:

- Repository owner and name from the GitHub remote
- Current branch
- Git user information for commit trailers

See [full documentation](docs/cmd/ghup.md) for details on configuration options.

## Basic Usage

### Managing Content

```sh
# Update a file
ghup content -b feature-branch -u local/file.txt:remote/path.txt

# Create a pull request with changes
ghup content -b new-feature -u config.json --pr-title "Update configuration"

# Create a pull request with auto-merge enabled (if repository supports it)
ghup content -b feature-branch -u config.json --pr-title "Auto-merge update" --auto-merge

# Add, update, and delete files in one commit
ghup content -b updates \
  -u local/new-file.txt:new-file.txt \
  -d old-file.txt \
  -c main:existing-file.txt:new-location.txt
```

See [content command documentation](docs/cmd/ghup_content.md) for more examples.

### Creating and Managing Tags

```sh
# Create an annotated tag
ghup tag v1.0.0 --commitish main

# Create a lightweight tag
ghup tag v1.0.0 --lightweight
```

See [tag command documentation](docs/cmd/ghup_tag.md) for more examples.

### Updating References

```sh
# Fast-forward a branch to match another
ghup update-ref -s main refs/heads/production

# Update GitHub Actions-style tags after a release
ghup update-ref -s tags/v1.2.3 tags/v1.2 tags/v1
```

See [update-ref command documentation](docs/cmd/ghup_update-ref.md) for more examples.

### Resolving Commits

```sh
# Resolve a branch to its SHA
ghup resolve main

# Find all tags pointing to a specific commit
ghup resolve abc123 --tags
```

See [resolve command documentation](docs/cmd/ghup_resolve.md) for more examples.

### Debugging

```sh
# View configuration and environment information
ghup debug
```

See [debug command documentation](docs/cmd/ghup_debug.md) for more details.

## GitHub Actions

`ghup` offers ready-to-use GitHub Actions to simplify integration within your workflows:

### Setup Action

The [`nexthink-oss/ghup/actions/setup`](actions/setup/README.md) action installs `ghup` and makes it available in your workflow:

```yaml
- uses: nexthink-oss/ghup/actions/setup@main
  with:
    version: v0.12.0 # optional, defaults to 'latest'
```

### Fast-Forward Action

The [`nexthink-oss/ghup/actions/fast-forward`](actions/fast-forward/README.md) action updates refs to match a source commit:

```yaml
- uses: nexthink-oss/ghup/actions/fast-forward@main
  with:
    source: main
    target: refs/heads/production
    # force: false # optional, defaults to false
  env:
    GITHUB_TOKEN: ${{ github.token }}
```

Use this action to implement true fast-forward merges or to create/update tags from specific commits.

See individual action READMEs for detailed usage examples and parameters.

## Documentation

Detailed documentation for all commands is available in the [docs/cmd](docs/cmd) directory:

- [`ghup`](docs/cmd/ghup.md): General command usage and configuration
- [`ghup content`](docs/cmd/ghup_content.md): Managing repository content
- [`ghup tag`](docs/cmd/ghup_tag.md): Creating and managing tags
- [`ghup update-ref`](docs/cmd/ghup_update-ref.md): Updating git refs
- [`ghup resolve`](docs/cmd/ghup_resolve.md): Resolving commit-ish references
- [`ghup debug`](docs/cmd/ghup_debug.md): Debugging configuration and environment

## Contributing

All contributions in the spirit of the project are welcome! Open an issue or pull request to get started.
