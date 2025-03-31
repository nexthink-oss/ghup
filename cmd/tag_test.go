//go:build acceptance
// +build acceptance

package cmd_test

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/nexthink-oss/ghup/cmd"
	"github.com/nexthink-oss/ghup/internal/util"
	"github.com/spf13/viper"
)

type tagTestArgs struct {
	Tag         string
	Commitish   string
	Lightweight bool
	Force       bool
}

func (s *tagTestArgs) Slice() []string {
	args := []string{"tag"}

	if s.Tag != "" {
		args = append(args, s.Tag)
	}

	if s.Commitish != "" {
		args = append(args, "--commitish", s.Commitish)
	}

	if s.Lightweight == true {
		args = append(args, "--lightweight")
	}

	if s.Force {
		args = append(args, "--force")
	}

	return args
}

func TestAccTagCmd(t *testing.T) {
	client, resources := setupTestResources(t)

	testSuffix := testRandomString(6)
	tagName := "tag-" + testSuffix
	lightTagName := "light-tag-" + testSuffix

	// Register tags for cleanup
	resources.AddTag(tagName)
	resources.AddTag(lightTagName)

	tests := []struct {
		name        string
		args        tagTestArgs
		wantError   bool
		wantStdout  *regexp.Regexp
		wantStderr  *regexp.Regexp
		checkJson   bool
		jsonUpdated bool
	}{
		{
			name:       "no args",
			wantStderr: regexp.MustCompile(`Error: tag is required`),
			wantError:  true,
		},
		{
			name: "create annotated tag",
			args: tagTestArgs{
				Tag: tagName,
			},
			checkJson:   true,
			jsonUpdated: true,
		},
		{
			name: "create annotated tag is idempotent",
			args: tagTestArgs{
				Tag: tagName,
			},
			checkJson:   true,
			jsonUpdated: false,
		},
		{
			name: "create lightweight tag with clashing annotated tag",
			args: tagTestArgs{
				Tag:         tagName,
				Lightweight: true,
			},
			wantError:   true,
			checkJson:   true,
			jsonUpdated: false,
		},
		{
			name: "force replace annotated tag with lightweight",
			args: tagTestArgs{
				Tag:         tagName,
				Lightweight: true,
				Force:       true,
			},
			checkJson:   true,
			jsonUpdated: true,
		},
		{
			name: "create lightweight tag",
			args: tagTestArgs{
				Tag:         lightTagName,
				Lightweight: true,
			},
			checkJson:   true,
			jsonUpdated: true,
		},
		{
			name: "create lightweight tag is idempotent",
			args: tagTestArgs{
				Tag:         lightTagName,
				Lightweight: true,
			},
			checkJson:   true,
			jsonUpdated: false,
		},
		{
			name: "create annotated tag with clashing lightweight tag",
			args: tagTestArgs{
				Tag: lightTagName,
			},
			wantError:   true,
			checkJson:   true,
			jsonUpdated: false,
		},
		{
			name: "force replace lightweight tag with annotated",
			args: tagTestArgs{
				Tag:   lightTagName,
				Force: true,
			},
			checkJson:   true,
			jsonUpdated: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// Set the target commit-ish to main by default
			viper.Reset()
			tt.Setenv("GHUP_COMMITISH", "main")

			spec := testCmdSpec{
				Args: append([]string{"-vvvv"}, test.args.Slice()...),
			}

			tt.Logf("args: %+v", spec.Args)

			stdoutBuf, stderrBuf, executeErr := testExecuteCmd(tt, spec)

			stdout := stdoutBuf.Bytes()
			stderr := stderrBuf.Bytes()

			if os.Getenv("TEST_GHUP_LOG_OUTPUT") != "" {
				tt.Logf("stdout:\n%s", string(stdout))
				tt.Logf("stderr:\n%s", string(stderr))
			}

			if (executeErr != nil) != test.wantError {
				tt.Errorf("unexpected error: got %v", executeErr)
			}

			if test.wantStderr != nil && !test.wantStderr.MatchString(string(stderr)) {
				tt.Errorf("unexpected stderr: got %q, want %q", string(stderr), test.wantStderr)
			}

			if test.wantStdout != nil && !test.wantStdout.MatchString(string(stdout)) {
				tt.Errorf("unexpected stdout: got %q, want %q", string(stdout), test.wantStdout)
			}

			// Validate the structured output
			if test.checkJson {
				var output cmd.TagOutput
				unmarshalErr := json.Unmarshal(stdout, &output)
				if unmarshalErr != nil {
					tt.Errorf("failed to unmarshal output: %v", unmarshalErr)
				} else {
					if output.Tag != test.args.Tag {
						tt.Errorf("unexpected tag name: got %q, want %q", output.Tag, test.args.Tag)
					}
					if !util.IsCommitHash(output.SHA) {
						tt.Errorf("unexpected SHA: got %q", output.SHA)
					}
					if output.URL != client.GetCommitURL(output.SHA) {
						tt.Errorf("unexpected URL: got %q, want %q", output.URL, client.GetCommitURL(output.SHA))
					}
					if output.Updated != test.jsonUpdated {
						tt.Errorf("unexpected updated status: got %v, want %v", output.Updated, test.jsonUpdated)
					}
				}
			}
		})
	}
}
