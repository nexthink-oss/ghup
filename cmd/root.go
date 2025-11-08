package cmd

import (
	"errors"

	"github.com/apex/log"
	"github.com/creasty/defaults"
	"github.com/spf13/cobra"

	"github.com/nexthink-oss/ghup/internal/local"
	"github.com/nexthink-oss/ghup/internal/util"
)

type OutputEncoder interface {
	Encode(any) error
}

type CommandOutput interface {
	GetError() error
	SetError(error)
}

var (
	localRepo local.Repository

	outputEncoder OutputEncoder
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "ghup",
		Short:             "Update GitHub content and tags via API",
		SilenceUsage:      true,
		PersistentPreRunE: commonSetup,
	}

	// load defaults from local repository context, if available
	if err := defaults.Set(&localRepo); err != nil {
		panic(err)
	}

	// override defaults from GitHub Actions environment variables
	// in case we're running without a checkout
	if actionsCtx := util.GithubActionsContext(); actionsCtx != nil {
		localRepo.Owner = actionsCtx.Owner
		localRepo.Name = actionsCtx.Name
		if actionsCtx.Branch != "" {
			localRepo.Branch = actionsCtx.Branch
		}
	}

	log.Debugf("local repository: %+v", localRepo)

	flags := cmd.Flags()
	persistentFlags := cmd.PersistentFlags()

	persistentFlags.StringSlice("config-path", []string{"."}, "configuration `name`")
	persistentFlags.StringP("config-name", "C", "", "configuration `name`")
	persistentFlags.String("token", "", "GitHub Token or path/to/token-file")
	persistentFlags.StringP("owner", "o", localRepo.Owner, "repository owner `name`")
	persistentFlags.StringP("repo", "r", localRepo.Name, "repository `name`")
	persistentFlags.Bool("no-cli-token", false, "disable fallback to GitHub CLI Token")
	persistentFlags.CountP("verbose", "v", "increase verbosity``")
	persistentFlags.StringP("output-format", "O", "json", "output `format` (json|j, yaml|y)")
	persistentFlags.Bool("compact", false, "compact output")

	flags.SortFlags = false
	persistentFlags.SortFlags = false

	cmd.AddCommand(
		cmdContent(),
		cmdDebug(),
		cmdDeployment(),
		cmdResolve(),
		cmdTag(),
		cmdUpdateRef(),
	)

	return cmd
}

func cmdOutput(cmd *cobra.Command, output CommandOutput) error {
	outputErr := output.GetError()

	encodeErr := outputEncoder.Encode(output)
	if encodeErr == nil {
		cmd.SilenceErrors = true
	}

	return errors.Join(outputErr, encodeErr)
}
