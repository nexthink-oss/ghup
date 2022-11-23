package cmd

import (
	"fmt"

	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type info struct {
	HasToken   bool `yaml:"hasToken"`
	Committer  string
	Owner      string
	Repository string
	Branch     string
	Commit     string
	IsClean    bool `yaml:"isClean"`
}

var infoCmd = &cobra.Command{
	Use:     "info [flags]",
	Short:   "Dump info on the current context",
	Args:    cobra.NoArgs,
	PreRunE: validateFlags,
	RunE:    runInfoCmd,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfoCmd(cmd *cobra.Command, args []string) (err error) {
	i := info{
		HasToken:   len(viper.GetString("token")) > 0,
		Committer:  util.BuildCommitter(),
		Owner:      owner,
		Repository: repo,
		Branch:     branch,
	}

	if localRepo != nil {
		i.Commit = localRepo.HeadCommit()

		status, err := localRepo.Status()
		if err == nil {
			i.IsClean = status.IsClean()
		}
	}

	m, err := yaml.Marshal(i)
	if err != nil {
		return err
	}

	fmt.Print(string(m))
	return
}
