# ghup update-ref

Update target refs to match source commitish.

## Synopsis

Update target refs to match source commitish, enabling precise ref management via the GitHub API.

```
ghup update-ref [flags] -s <source-commitish> <target-ref> ...
```

## Description

The `update-ref` command enables you to update one or more references (branches, tags, or any ref) to point to a specific commit. This is a lower-level command that gives you direct control over repository references.

Common use cases include:
- Syncing multiple branches/tags to point to the same commit
- Creating or updating branches or tags programmatically
- Managing references that are not standard branches or tags

This command is particularly useful in CI/CD pipelines where you need to update multiple references as part of automated processes.

Source commitish may also be passed via the `GHUP_SOURCE` environment variable, and target refs via `GHUP_TARGETS` (space-delimited).

## Options

```
  -s, --source string        source commitish
  -f, --force                force update if ref exists
  -h, --help                 help for update-ref
```

## Examples

```bash
# Update a branch to match another branch
ghup update-ref -s main refs/heads/production

# Update a tag to point to the latest commit on main
ghup update-ref -s main refs/tags/latest

# Update multiple refs in one command
ghup update-ref -s v1.0.0 refs/heads/stable refs/tags/production-ready

# Force update an existing ref
ghup update-ref -s main refs/heads/production -f

# Update refs using environment variables
export GHUP_SOURCE=main
export GHUP_TARGETS="refs/heads/production refs/tags/latest"
ghup update-ref
```

## Output

The command returns a JSON (or YAML) object with the following structure:

```json
{
  "repository": "owner/repo",
  "source": {
    "commitish": "main",
    "sha": "full-sha-hash"
  },
  "target": [
    {
      "ref": "refs/heads/production",
      "old": "previous-sha-if-updated",
      "sha": "new-sha-hash",
      "updated": true
    },
    {
      "ref": "refs/tags/latest",
      "old": "previous-sha-if-updated",
      "sha": "new-sha-hash",
      "updated": true
    }
  ]
}
```

- `source`: Information about the source commitish
- `target`: Array of information about each target ref that was processed
  - `ref`: The full reference name
  - `old`: The previous SHA (only present if updated)
  - `sha`: The new SHA the ref points to
  - `updated`: Whether the ref was actually changed
  - `error`: Error message if updating this specific ref failed

If an error occurs with the source commitish, the output will include an error message in the source object.
