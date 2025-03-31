# ghup tag

Create or update lightweight or annotated tags.

## Synopsis

Create or update lightweight or annotated tags in a GitHub repository.

```
ghup tag [flags] [<name>]
```

## Description

The `tag` command allows you to create or update Git tags in a GitHub repository.
By default, it creates annotated tags that include a message, timestamp, and tagger
information. With the `--lightweight` flag, it creates lightweight tags that are
simply pointers to specific commits.

The command is idempotent - if a tag with the same name and target already exists,
no action is taken. If the tag exists but points to a different target, you must use
the `--force` flag to update it.

This command is particularly useful for:
- Creating release tags from CI pipelines
- Ensuring tags have the proper verification status
- Managing tags without needing a git checkout

## Options

```
      --tag string              tag name
  -l, --lightweight             force lightweight tag
  -c, --commitish string        target commitish (default is the default branch)
  -m, --message string          commit message (default "Commit via API")
      --user-trailer string     key for commit author trailer (blank to disable) (default "Co-Authored-By")
      --user-name string        name for commit author trailer
      --user-email string       email for commit author trailer
      --trailer stringToString  extra key=value commit trailers
  -f, --force                   force update if tag already exists
  -h, --help                    help for tag
```

## Examples

```bash
# Create an annotated tag on the default branch
ghup tag v1.0.0

# Create a lightweight tag
ghup tag v1.0.0 --lightweight

# Tag a specific commit
ghup tag v1.0.0 --commitish abc123def456

# Tag a specific branch
ghup tag release-candidate --commitish feature/new-feature

# Force update an existing tag
ghup tag v1.0.0 --commitish main --force

# Create an annotated tag with a custom message
ghup tag v1.0.0 --message "Release version 1.0.0"

# Create a tag with custom trailers
ghup tag v1.0.0 --trailer "Reviewed-By=Jane Doe" --trailer "Fixed-Issue=123"
```

## Output

The command returns a JSON (or YAML) object with the following structure:

```json
{
  "tag": "v1.0.0",
  "commitish": "main",
  "sha": "full-sha-hash",
  "url": "https://github.com/owner/repo/commit/full-sha-hash",
  "updated": true
}
```

- `tag`: The name of the tag that was created or updated
- `commitish`: The original commitish that was specified
- `sha`: The full SHA hash of the commit the tag points to
- `url`: The URL to view the commit on GitHub
- `updated`: Whether the tag was created/updated (`true`) or already existed with the same target (`false`)

If an error occurs, the output will include an error message explaining the issue.
