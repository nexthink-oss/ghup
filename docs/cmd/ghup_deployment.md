# ghup deployment

Create a deployment and deployment status on GitHub.

## Synopsis

Create GitHub deployments and deployment statuses to track application deployments across different environments.

```
ghup deployment [flags]
```

## Description

The `deployment` command creates GitHub deployments and their corresponding deployment statuses. This is useful for tracking deployments in CI/CD pipelines and integrating with GitHub's deployment protection rules and environment management features.

When executing this command, `ghup` will:
1. Resolve the target commit (from `--commitish` or default branch)
2. Check for existing deployments for the same SHA and environment
3. Create a new deployment if none exists, or reuse an existing one
4. Create a deployment status with the specified state

Deployments are environment-specific and can be marked as production or transient environments. The deployment status indicates the current state of the deployment (success, failure, in progress, etc.).

## Options

```
-c, --commitish commitish                                               target commitish (default HEAD)
-e, --environment string                                                deployment environment (required)
-s, --state success|pending|failure|error|in_progress|queued|inactive   deployment state (default success)
-T, --transient                                                         transient environment
-P, --production                                                        production environment
    --description string                                                deployment description
    --environment-url string                                            environment URL
-n, --dry-run                                                           dry-run mode
-h, --help                                                              help for deployment
```

## Deployment States

The `--state` flag accepts the following values:

- `success` - Deployment completed successfully (default)
- `pending` - Deployment is pending
- `failure` - Deployment failed
- `error` - Deployment encountered an error
- `in_progress` - Deployment is currently in progress
- `queued` - Deployment is queued
- `inactive` - Deployment is inactive

## Examples

```bash
# Create a successful deployment to staging environment
ghup deployment --environment staging --state success

# Create a production deployment with description
ghup deployment --environment production \
  --state success \
  --production \
  --description "Release v1.2.3"

# Create a deployment for a specific commit
ghup deployment --environment staging \
  --commitish v1.2.3 \
  --state success

# Create a transient environment deployment
ghup deployment --environment pr-123 \
  --transient \
  --state in_progress \
  --description "PR #123 preview environment"

# Create a deployment with environment URL
ghup deployment --environment production \
  --state success \
  --environment-url "https://app.example.com" \
  --description "Production deployment"

# Mark a deployment as failed
ghup deployment --environment staging \
  --state failure \
  --description "Deployment failed due to test failures"

# Dry run to see what would be created
ghup deployment --environment staging \
  --state success \
  --dry-run
```

## Output

The command returns a JSON (or YAML) object with the following structure:

```json
{
  "deployment_id": 12345,
  "status_id": 67890,
  "environment": "staging",
  "commitish": "main",
  "sha": "abc123def456",
  "state": "success",
  "created": true
}
```

Fields:
- `deployment_id`: The ID of the GitHub deployment
- `status_id`: The ID of the deployment status
- `environment`: The deployment environment name
- `commitish`: The commitish that was resolved
- `sha`: The full SHA of the target commit
- `state`: The deployment state
- `created`: Whether a new deployment was created (false if reusing existing)

## Integration with GitHub Features

GitHub deployments integrate with several platform features:

- **Environment Protection Rules**: Require reviews or checks before deployments
- **Deployment Branches**: Restrict which branches can deploy to specific environments
- **Status Checks**: Link deployment status to required status checks
- **Pull Request Integration**: Show deployment status in pull request interfaces

## Use Cases

- **CI/CD Pipelines**: Track deployment progress and outcomes
- **Environment Management**: Organize deployments by environment (staging, production, etc.)
- **Release Tracking**: Associate deployments with specific commits or tags
- **Preview Environments**: Create transient environments for pull requests
- **Rollback Tracking**: Mark deployments as inactive when rolling back
