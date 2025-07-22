# ghup content

Manage repository content via the GitHub API.

## Synopsis

Directly manage repository content via the GitHub API, ensuring verified commits from CI systems.

```
ghup content [flags] [<file-spec> ...]
```

## Description

The `content` command allows you to add, update, or delete files in a GitHub repository through the GitHub API. This is especially useful in CI/CD pipelines where you need to make changes with proper commit verification status.

When executing this command, `ghup` will:

1. Create the target branch if it doesn't exist (unless `--create-branch=false`)
2. Apply the specified file operations (additions, updates, deletions)
3. Create a commit with the changes
4. Optionally create a pull request if `--pr-title` is specified

File operations are idempotent by default - if a file already has the target content, no changes will be made unless `--force` is specified.

## Options

```
      --tracked                 commit changes to tracked files
      --staged                  commit staged changes
  -c, --copy strings            remote file-spec to copy ([src-branch<separator>]src-path[<separator>dst-path]); non-binary files only!
  -u, --update strings          file-spec to update (local-path[<separator>remote-path])
  -d, --delete strings          remote-path to delete
  -s, --separator string        file-spec separator (default ":")
  -m, --message string          commit message (default "Commit via API")
      --user-trailer string     key for commit author trailer (blank to disable) (default "Co-Authored-By")
      --user-name string        name for commit author trailer
      --user-email string       email for commit author trailer
      --trailer stringToString  extra key=value commit trailers
  -b, --branch string           target branch name
      --create-branch           create missing target branch (default true)
      --base-branch string      base branch name (default: "[remote-default-branch]")
      --pr-title string         pull request title
      --pr-body string          pull request body
      --pr-draft                create pull request in draft mode
      --pr-auto-merge string    auto-merge method for pull request (off|merge|squash|rebase) (default "off")
  -n, --dry-run                 dry-run mode
  -f, --force                   force operation
  -h, --help                    help for content
```

## Examples

```bash
# Update a single file
ghup content -b feature-branch -u local/path/to/file.txt:remote/path/to/file.txt

# Add multiple files and create a pull request
ghup content -b new-feature \
  -u local/config.json:config.json \
  -u local/README.md:README.md \
  --pr-title "Update configuration and documentation"

# Create a pull request with auto-merge enabled using merge method
ghup content -b feature-branch \
  -u local/file.txt:remote/file.txt \
  --pr-title "Auto-merge PR with merge commit" \
  --pr-auto-merge merge

# Create a pull request with squash auto-merge
ghup content -b feature-branch \
  -u local/file.txt:remote/file.txt \
  --pr-title "Auto-merge PR with squash" \
  --pr-auto-merge squash

# Create a pull request with rebase auto-merge
ghup content -b feature-branch \
  -u local/file.txt:remote/file.txt \
  --pr-title "Auto-merge PR with rebase" \
  --pr-auto-merge rebase

# Copy a file from another branch
ghup content -b feature-branch -c main:existing/file.txt:new/location/file.txt

# Delete a file
ghup content -b cleanup-branch -d obsolete/file.txt

# Combining operations in one commit
ghup content -b feature-branch \
  -u local/new-file.txt:new-file.txt \
  -d old-file.txt \
  -c main:move-me.txt:new-location.txt

# Using a different path separator
ghup content -b feature-branch -s "|" -u "local/file.txt|remote/file.txt"

# Commit all tracked changes from local repository
ghup content -b feature-branch --tracked -m "Sync all tracked changes"

# Only commit staged changes from local repository
ghup content -b feature-branch --staged -m "Apply staged changes"
```

## Output

The command returns a JSON (or YAML) object with the following structure:

```json
{
  "repository": "owner/repo",
  "sha": "commit-sha-if-created",
  "updated": true,
  "pullrequest": {
    "url": "https://github.com/owner/repo/pull/123",
    "number": 123
  }
}
```

If there were no changes to commit (idempotent operation), `updated` will be `false`.
