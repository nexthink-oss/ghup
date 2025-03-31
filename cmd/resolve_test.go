//go:build acceptance
// +build acceptance

package cmd_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-github/v70/github"

	"github.com/nexthink-oss/ghup/cmd"
	"github.com/nexthink-oss/ghup/internal/util"
)

type resolveTestArgs struct {
	Commitish string
	Branches  bool
	Tags      bool
}

func (s *resolveTestArgs) Slice() []string {
	args := []string{"resolve"}

	if s.Commitish != "" {
		args = append(args, s.Commitish)
	}

	if s.Branches {
		args = append(args, "--branches")
	}

	if s.Tags {
		args = append(args, "--tags")
	}

	return args
}

func TestAccResolveCmd(t *testing.T) {
	client, resources := setupTestResources(t)

	// Generate a unique branch name for our tests
	testBranch := "test-content-" + testRandomString(8)
	testTag := "test-tag-" + testRandomString(8)
	// Add to resources for cleanup
	resources.AddBranch(testBranch)
	resources.AddTag(testTag)

	mainSha, err := client.ResolveCommitish("main")
	if err != nil {
		t.Fatalf("failed to resolve main commitish: %v", err)
	}

	// Create a new branch from main
	_, _, err = client.UpdateRefName(
		testBranch,
		&github.Reference{
			Ref:    github.String("refs/heads/" + testBranch),
			Object: &github.GitObject{SHA: github.Ptr(mainSha)},
		},
		false,
	)

	// Create a new tag from main
	_, _, err = client.UpdateRefName(
		testTag,
		&github.Reference{
			Ref:    github.String("refs/tags/" + testTag),
			Object: &github.GitObject{SHA: github.Ptr(mainSha)},
		},
		false,
	)

	tests := []struct {
		name      string
		args      resolveTestArgs
		wantError bool
		checkJson bool
	}{
		{
			name: "Resolve plain commitish to SHA only",
			args: resolveTestArgs{
				Commitish: "main",
			},
			checkJson: true,
		},
		{
			name: "Resolve relative commitish to SHA only",
			args: resolveTestArgs{
				Commitish: "main~1",
			},
			checkJson: true,
		},
		{
			name: "Resolve commitish to SHA and list matching branches",
			args: resolveTestArgs{
				Commitish: "main",
				Branches:  true,
			},
			checkJson: true,
		},
		{
			name: "Resolve commitish to SHA and list matching tags",
			args: resolveTestArgs{
				Commitish: "main",
				Tags:      true,
			},
			checkJson: true,
		},
		{
			name: "Resolve non-existing commitish",
			args: resolveTestArgs{
				Commitish: "thisdefinitelydoesnotexist",
			},
			wantError: true,
			checkJson: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			spec := testCmdSpec{
				Args: append([]string{"-vvvv"}, test.args.Slice()...),
			}

			tt.Logf("args: %+v", spec.Args)

			stdoutBuf, stderrBuf, err := testExecuteCmd(tt, spec)

			stdout := stdoutBuf.Bytes()
			stderr := stderrBuf.Bytes()

			if os.Getenv("TEST_GHUP_LOG_OUTPUT") != "" {
				tt.Logf("stdout:\n%s", string(stdout))
				tt.Logf("stderr:\n%s", string(stderr))
			}

			if (err != nil) != test.wantError {
				tt.Errorf("gotErr=%v, wantError=%v", err, test.wantError)
			}

			if test.checkJson {
				var output cmd.ResolveOutput
				unmarshalErr := json.Unmarshal(stdout, &output)
				if unmarshalErr != nil {
					tt.Errorf("failed to unmarshal output: %v", unmarshalErr)
				} else {
					if output.Commitish != test.args.Commitish {
						tt.Errorf("unexpected commitish: got %q, want %q", output.Commitish, test.args.Commitish)
					}
					if test.wantError && output.ErrorMessage == "" {
						tt.Errorf("expected error message, got none")
					}
					if !test.wantError && !util.IsCommitHash(output.SHA) {
						tt.Errorf("unexpected SHA: got %q", output.SHA)
					}
					if test.args.Branches && len(output.Branches) == 0 {
						tt.Errorf("expected branches, got none")
					}
					if test.args.Tags && len(output.Tags) == 0 {
						tt.Errorf("expected tags, got none")
					}
				}
			}
		})
	}
}
