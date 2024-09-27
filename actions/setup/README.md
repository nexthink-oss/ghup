# `setup` action

The `nexthink-oss/ghup/actions/setup` action is designed to make the `ghup` tool available within your workflows.

Ensure that any steps that use the tool are passed appropriate credentials, i.e. by setting the `GITHUB_TOKEN` environment variable.

## Inputs

### `version` input

**Optional** version of `ghup` to install (default: `latest`)

## Outputs

### `version` output

The version of `ghup` actually installed.

## Example usage

```yaml
name: autobuild

on:
  pull_request:
    branches: [main]

permissions:
  contents: write

jobs:
  autobuild:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - uses: nexthink-oss/ghup/actions/setup@main
        with:
          version: v0.10.0 # default: latest

      - name: Build
        run: npm ci && npm run build

      - name: Idempotently commit updated artifacts
        env:
          GITHUB_TOKEN: ${{ github.token }}
          GHUP_BRANCH: ${{ github.head_ref }}
          GHUP_MESSAGE: "build: autobuild #${{ github.event.pull_request.number }}"
          GHUP_TRAILER: '{"Build-Logs": "https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}"}'
        run: ghup content dist/*
```
