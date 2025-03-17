# ghup debug

Dump contextual information to aid debugging.

## Synopsis

Dump contextual information about the current environment, repository, and configuration to aid debugging.

```
ghup debug [flags]
```

## Description

The `debug` command provides detailed information about the current `ghup` configuration, environment, and repository context. This is useful for troubleshooting issues with the tool or understanding the environment in which it's running.

The command displays information such as:
- Repository details
- Current branch and commit
- Authentication status
- Commit message and trailers that would be used
- Local repository status

This command is particularly helpful when:
- Diagnosing issues with authentication
- Confirming environment variables are being picked up correctly
- Verifying the correct repository is being targeted
- Testing commit message formatting before making actual commits

## Options

```
  -b, --branch string           target branch name
  -m, --message string          commit message (default "Commit via API")
      --user-trailer string     key for commit author trailer (blank to disable) (default "Co-Authored-By")
      --user-name string        name for commit author trailer
      --user-email string       email for commit author trailer
      --trailer stringToString  extra key=value commit trailers
  -h, --help                    help for debug
```

## Examples

```bash
# Show basic debug information
ghup debug

# Show debug information for a specific branch
ghup debug -b feature-branch

# Preview commit message formatting
ghup debug -m "Testing commit message" --trailer "Reviewed-By=Jane Doe"
```

## Output

The command returns a JSON (or YAML) object with the following structure:

```json
{
  "remote": {
    "owner": "example-org",
    "name": "example-repo"
  },
  "has_token": true,
  "branch": "main",
  "commit": "current-head-commit-sha",
  "clean": true,
  "message": {
    "headline": "Testing commit message",
    "body": ""
  },
  "trailers": [
    "Co-Authored-By: Jane Doe <jane@example.com>",
    "Reviewed-By: Jane Doe"
  ]
}
```

- `remote`: Information about the target repository
- `has_token`: Whether a GitHub token is available
- `branch`: The current or specified branch
- `commit`: The current HEAD commit SHA (if in a git repository)
- `clean`: Whether the working directory is clean (no uncommitted changes)
- `message`: The formatted commit message that would be used
- `trailers`: The commit trailers that would be included

If an error occurs, the output will include an error message explaining the issue.
