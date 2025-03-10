# `fast-forward` action

The `nexthink-oss/ghup/actions/fast-forward` action will fast-forward one or more `target` refs (heads or tags) to match `source` commit-ish.

It can be used, for example, to create/update tags or to implement true fast-forward merge for GitHub PRs.

## Inputs

### `source` input

**Required** a commit-ish from which to source the target commit.

### `target` input

**Required** a whitespace-separated list of targets to update to the resolved source commit.

### `force` input

**Optional** force update target heads even if fast-forward is not possible. Default: `false`

### `version` input

**Optional** version of `ghup` to install (default: `latest`)

## Outputs

### `source` output

Resolved source details.

### `target` output

Target update details.

### `version`

The version of `ghup` actually installed.

## Example usage

```yaml
name: release-to-environment

on:
  workflow_dispatch:
    inputs:
      revision:
        description: 'Ref-or-commit to release'
        type: string
        default: heads/main
      environment:
        description: 'Environment to update'
        type: environment
        required: true

permissions:
  contents: write

jobs:
  release-to-environment:
    runs-on: ubuntu-latest
    environment: ${{ inputs.environment }}
    steps:
      - name: Update Environment Tag
        env:
          GITHUB_TOKEN: ${{ github.token }}
        uses: nexthink-oss/ghup/actions/fast-forward
        with:
          source: ${{ inputs.revision }}
          target: tags/${{ inputs.environment }}
```
