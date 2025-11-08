package cmd

import (
	"fmt"
	"strconv"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/pkg/choiceflag"
)

type DeploymentOutput struct {
	DeploymentID int64  `json:"deployment_id" yaml:"deployment_id"`
	StatusID     int64  `json:"status_id" yaml:"status_id"`
	Environment  string `json:"environment" yaml:"environment"`
	Commitish    string `json:"commitish" yaml:"commitish"`
	SHA          string `json:"sha" yaml:"sha"`
	State        string `json:"state" yaml:"state"`
	URL          string `json:"url" yaml:"url"`
	Created      bool   `json:"created" yaml:"created"`
	Error        error  `json:"-" yaml:"-"`
	ErrorMessage string `json:"error,omitempty" yaml:"error,omitempty"`
}

func (o *DeploymentOutput) GetError() error {
	return o.Error
}

func (o *DeploymentOutput) SetError(err error) {
	o.Error = err
	if err != nil {
		o.ErrorMessage = err.Error()
	}
}

func cmdDeployment() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment [flags] [<environment>]",
		Short: "Update deployment status for a specific environment.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runDeploymentCmd,
	}

	flags := cmd.Flags()
	flags.StringP("commitish", "c", localRepo.Branch, "target `commitish`")
	flags.StringP("environment", "e", "", "deployment environment `name`")

	states := []string{"success", "pending", "failure", "error", "in_progress", "queued", "inactive"}
	defaultState := choiceflag.NewChoiceFlag(states)
	_ = defaultState.Set("success")
	flags.VarP(defaultState, "state", "s", "deployment state")

	flags.BoolP("transient", "T", false, "transient environment")
	flags.BoolP("production", "P", false, "production environment")
	flags.String("description", "", "deployment description")
	flags.String("environment-url", "", "environment URL")

	addDryRunFlag(flags)

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	return cmd
}

func runDeploymentCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	environment := viper.GetString("environment")

	if len(args) == 1 {
		environment = args[0]
	}

	if environment == "" {
		return fmt.Errorf("environment is required")
	}

	state := viper.GetString("state")

	dryRun := viper.GetBool("dry-run")

	repo := remote.Repo{
		Owner: viper.GetString("owner"),
		Name:  viper.GetString("repo"),
	}

	client, err := remote.NewClient(ctx, &repo)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	commitish := viper.GetString("commitish")
	var targetSha string
	if commitish != "" {
		targetSha, err = client.ResolveCommitish(commitish)
		if err != nil {
			return fmt.Errorf("ResolveCommitish(%s, %s): %w", repo, commitish, err)
		}
		if targetSha == "" {
			return fmt.Errorf("commitish %q not found", commitish)
		}
	} else {
		repoInfo, err := client.GetRepositoryInfo("")
		if err != nil {
			return fmt.Errorf("GetRepositoryInfo(%s): %w", repo, err)
		}
		commitish = repoInfo.DefaultBranch.Name
		targetSha = string(repoInfo.DefaultBranch.Commit)
	}

	output := &DeploymentOutput{
		Environment: environment,
		Commitish:   commitish,
		SHA:         targetSha,
		State:       state,
		URL:         client.GetCommitURL(targetSha),
	}

	// Check if deployment already exists
	var deployment *remote.DeploymentInfo
	var deploymentID string = "1" // Default for dry-run mode

	if !dryRun {
		deployments, err := client.ListDeploymentsV3(targetSha, environment)
		if err != nil {
			output.SetError(fmt.Errorf("listing deployments: %w", err))
			return cmdOutput(cmd, output)
		}

		if len(deployments) > 0 {
			// Use existing deployment
			deployment = &deployments[0]
			deploymentID = string(deployment.ID)
			log.Infof("using existing deployment: %s", deploymentID)
		} else {
			// Create new deployment
			transient := viper.GetBool("transient")
			production := viper.GetBool("production")
			description := viper.GetString("description")

			log.Infof("creating deployment for %s in %s environment", commitish, environment)
			deployment, err = client.CreateDeploymentV3(commitish, environment, description, transient, production)
			if err != nil {
				output.SetError(fmt.Errorf("creating deployment: %w", err))
				return cmdOutput(cmd, output)
			}

			deploymentID = string(deployment.ID)
			output.Created = true
			log.Infof("created deployment: %s", deploymentID)
		}
	} else {
		log.Infof("dry-run: skipping deployment creation/lookup for %s in %s environment", commitish, environment)
	}

	// Convert deploymentID to int64 for output compatibility
	if deploymentIDInt, err := strconv.ParseInt(deploymentID, 10, 64); err == nil {
		output.DeploymentID = deploymentIDInt
	} else {
		// For GraphQL node IDs that can't be parsed as int64, use 0
		output.DeploymentID = 0
	}

	// Create deployment status
	description := viper.GetString("description")
	environmentURL := viper.GetString("environment-url")

	if !dryRun {
		log.Infof("creating deployment status: %s", state)
		statusID, err := client.CreateDeploymentStatusV3(deploymentID, state, description, environment, environmentURL)
		if err != nil {
			output.SetError(fmt.Errorf("creating deployment status: %w", err))
			return cmdOutput(cmd, output)
		}

		output.StatusID = statusID
		log.Infof("created deployment status: %d", statusID)
	} else {
		log.Infof("dry-run: skipping creation of deployment status: %s", state)
	}

	return cmdOutput(cmd, output)
}
