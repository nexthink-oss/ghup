package remote

import (
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
)

func CommittableBranch(owner string, repo string, branch string) githubv4.CommittableBranch {
	return githubv4.CommittableBranch{
		RepositoryNameWithOwner: githubv4.NewString(githubv4.String(fmt.Sprintf("%s/%s", owner, repo))),
		BranchName:              githubv4.NewString(githubv4.String(branch)),
	}
}

func CommitMessage(message string) (commitMessage githubv4.CommitMessage) {
	split := strings.SplitN(message, "\n", 2)
	switch {
	case len(split) < 1:
		return
	case len(split) == 2:
		return githubv4.CommitMessage{
			Headline: githubv4.String(split[0]),
			Body:     githubv4.NewString(githubv4.String(split[1])),
		}
	default:
		return githubv4.CommitMessage{
			Headline: githubv4.String(split[0]),
		}
	}
}
