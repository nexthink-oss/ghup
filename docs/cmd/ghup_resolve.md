# ghup resolve

Resolve a commit-ish to a SHA and optionally find matching refs.

## Synopsis

Resolve a commit-ish to a SHA, optionally finding matching branches and/or tags.

```
ghup resolve [<commit-ish>] [flags]
```

## Description

The `resolve` command resolves a commit-ish (branch name, tag name, SHA, etc.) to its full SHA hash.
It can also find all branches and/or tags that point to the same commit.

This is useful for:
- Verifying if a commit exists in a repository
- Finding the full SHA when you only have a short SHA or ref name
- Discovering all branches or tags that point to a specific commit

If the commit-ish doesn't exist in the repository, the command will return an error.

## Options

```
      --commitish string   commitish to match (default "HEAD")
  -b, --branches           list matching branches/heads
  -t, --tags               list matching tags
  -h, --help               help for resolve
```

## Examples

```bash
# Resolve a branch name to a SHA
ghup resolve main

# Resolve a tag to a SHA
ghup resolve v1.0.0

# Resolve a short SHA to its full form
ghup resolve abc123

# Find all branches that point to a specific commit
ghup resolve abc123 --branches

# Find all tags that point to a specific commit
ghup resolve v1.0.0 --tags

# Find all refs that point to a specific commit
ghup resolve main --branches --tags
```

## Output

The command returns a JSON (or YAML) object with the following structure:

```json
{
  "repository": "owner/repo",
  "commitish": "main",
  "sha": "full-sha-hash",
  "branches": [
    "main",
    "feature/branch"
  ],
  "tags": [
    "v1.0.0",
    "stable"
  ]
}
```

The `branches` and `tags` arrays will only be present if the corresponding flags were provided.
If the commit-ish doesn't exist, the command will return an error and the output will contain an error message.
