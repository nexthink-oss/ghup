package cmd

import (
	"errors"
	"fmt"

	"github.com/apex/log"
	"github.com/google/go-github/v69/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nexthink-oss/ghup/internal/remote"
	"github.com/nexthink-oss/ghup/internal/util"
)

type TagOutput struct {
	Tag          string `json:"tag" yaml:"tag"`
	Commitish    string `json:"commitish" yaml:"commitish"`
	SHA          string `json:"sha" yaml:"sha"`
	URL          string `json:"url" yaml:"url"`
	Updated      bool   `json:"updated" yaml:"updated"`
	Error        error  `json:"-" yaml:"-"`
	ErrorMessage string `json:"error,omitempty" yaml:"error,omitempty"`
}

func (o *TagOutput) GetError() error {
	return o.Error
}

func (o *TagOutput) SetError(err error) {
	o.Error = err
	if err != nil {
		o.ErrorMessage = err.Error()
	}
}

func cmdTag() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag [flags] [<name>]",
		Short: "Create or update lightweight or annotated tags.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runTagCmd,
	}

	flags := cmd.Flags()
	flags.String("tag", "", "tag `name`")
	flags.BoolP("lightweight", "l", false, "force lightweight tag")
	flags.StringP("commitish", "c", localRepo.Branch, "target `commitish`")
	addBranchFlag(flags)
	flags.MarkDeprecated("branch", "pass commitish via -c/--commitish instead")
	flags.MarkHidden("branch")
	addCommitMessageFlags(flags)
	addForceFlag(flags)

	flags.SetNormalizeFunc(normalizeFlags)
	flags.SortFlags = false

	return cmd
}

func runTagCmd(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

	tagName := viper.GetString("tag")

	if len(args) == 1 {
		tagName = args[0]
	}

	if tagName == "" {
		return fmt.Errorf("tag is required")
	}

	repo := remote.Repo{
		Owner: viper.GetString("owner"),
		Name:  viper.GetString("repo"),
	}

	lightweight := viper.GetBool("lightweight")
	force := viper.GetBool("force")
	update := false

	client, err := remote.NewClient(ctx, &repo)
	if err != nil {
		return fmt.Errorf("NewClient(%s): %w", repo, err)
	}

	repoInfo, err := client.GetRepositoryInfo("")
	if err != nil {
		return fmt.Errorf("GetRepositoryInfo(%s): %w", repo, err)
	}

	if repoInfo.IsEmpty {
		return errors.New("cannot tag empty repository")
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
		commitish = repoInfo.DefaultBranch.Name
		targetSha = string(repoInfo.DefaultBranch.Commit)
	}

	output := &TagOutput{
		Tag:       tagName,
		Commitish: commitish,
		SHA:       targetSha,
		URL:       client.GetCommitURL(targetSha),
	}

	tagRefName, err := util.QualifiedRefName(tagName, "tags")
	if err != nil {
		return fmt.Errorf("Invalid tag reference: %s: %w", tagRefName, err)
	}

	log.Infof("checking tag reference: %s", tagRefName)

	tagRef := &github.Reference{
		Ref:    &tagRefName,
		Object: &github.GitObject{SHA: github.Ptr(targetSha)},
	}

	tagObj, err := client.GetTagObj(tagRefName)
	if err != nil {
		// tag does not exist, or other error
		if !errors.Is(err, remote.NoMatchingObjectError) {
			return fmt.Errorf("GetTagObj(%s): %w", tagRefName, err)
		}
		log.Debug("tag does not exist")
		// fallthrough to create non-existent tag
	} else {
		// tag already exists
		log.Debugf("tag exists: %+v", tagObj)
		if targetSha == tagObj.Commit.SHA && lightweight == tagObj.Lightweight {
			// matching tag already exists
			log.Infof("tag exists: idempotent")
			return cmdOutput(cmd, output)
		} else if !force {
			log.Infof("tag exists: wrong type without force")
			// tag exists but points to a different commit
			if tagObj.Lightweight {
				err = fmt.Errorf("lightweight tag already exists, targeting %s", tagObj.Commit.SHA)
			} else {
				err = fmt.Errorf("annotated tag already exists, targeting %s", tagObj.Commit.SHA)
			}
			output.SetError(err)
			return cmdOutput(cmd, output)
		} else {
			log.Infof("tag exists: forcing update")
			update = true
		}
	}

	if !lightweight {
		message := util.BuildCommitMessage()
		log.Debugf("creating tag object: %s", tagName)
		tag, err := client.CreateTag(tagName, message, targetSha)
		if err != nil {
			output.SetError(fmt.Errorf("creating tag object: %w", err))
			return cmdOutput(cmd, output)
		}

		log.Debugf("created tag object: %+v", tag)

		tagRef.Object = &github.GitObject{SHA: tag.SHA}
	}

	if update {
		log.Infof("updating tag reference: %s", tagRefName)
		_, err = client.UpdateRef(tagRef, true)
		if err != nil {
			err = fmt.Errorf("updating tag ref: %w", err)
		}
	} else {
		log.Infof("creating tag reference: %s", tagRefName)
		_, err = client.CreateRef(tagRef)
		if err != nil {
			err = fmt.Errorf("updating tag ref: %w", err)
		}
	}

	if err == nil {
		output.Updated = true
	} else {
		output.SetError(err)
	}

	return cmdOutput(cmd, output)
}
