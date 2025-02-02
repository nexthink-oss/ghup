package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nexthink-oss/ghup/internal/util"
)

var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "Preview commit message",
	Args:  cobra.NoArgs,
	Run:   runMessageCmd,
}

func init() {
	defaultsOnce.Do(loadDefaults)

	flags := messageCmd.Flags()

	addCommitMessageFlags(flags)

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	rootCmd.AddCommand(messageCmd)
}

func runMessageCmd(cmd *cobra.Command, args []string) {
	message := util.BuildCommitMessage()

	fmt.Println(message)
}
