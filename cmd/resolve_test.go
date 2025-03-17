//go:build acceptance
// +build acceptance

package cmd_test

import (
	"encoding/json"
	"os"
	"testing"

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
	setupTestResources(t) // Ensures GHUP_TOKEN, GHUP_OWNER, GHUP_REPO, etc. are set.

	tests := []struct {
		name      string
		args      resolveTestArgs
		wantError bool
	}{
		{
			name: "Resolve plain commitish to SHA only",
			args: resolveTestArgs{Commitish: "main"},
		},
		{
			name: "Resolve relative commitish to SHA only",
			args: resolveTestArgs{Commitish: "main~1"},
		},
		{
			name: "Resolve commitish to SHA and list matching branches",
			args: resolveTestArgs{Commitish: "main", Branches: true},
		},
		{
			name: "Resolve commitish to SHA and list matching tags",
			args: resolveTestArgs{Commitish: "main", Tags: true},
		},
		{
			name:      "Resolve non-existing commitish",
			args:      resolveTestArgs{Commitish: "thisdefinitelydoesnotexist"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := testCmdSpec{
				Args: append([]string{"-vvvv"}, tt.args.Slice()...),
			}

			t.Logf("args: %+v", spec.Args)

			stdoutBuf, stderrBuf, err := testExecuteCmd(t, spec)

			stdout := stdoutBuf.Bytes()
			stderr := stderrBuf.Bytes()

			if os.Getenv("TEST_GHUP_LOG_OUTPUT") != "" {
				t.Logf("stdout:\n%s", string(stdout))
				t.Logf("stderr:\n%s", string(stderr))
			}

			if (err != nil) != tt.wantError {
				t.Errorf("gotErr=%v, wantError=%v", err, tt.wantError)
			}

			var output cmd.ResolveOutput
			unmarshalErr := json.Unmarshal(stdout, &output)
			if unmarshalErr != nil {
				t.Errorf("failed to unmarshal output: %v", unmarshalErr)
			} else {
				if output.Commitish != tt.args.Commitish {
					t.Errorf("unexpected commitish: got %q, want %q", output.Commitish, tt.args.Commitish)
				}
				if tt.wantError && output.ErrorMessage == "" {
					t.Errorf("expected error message, got none")
				}
				if !tt.wantError && !util.IsCommitHash(output.SHA) {
					t.Errorf("unexpected SHA: got %q", output.SHA)
				}
				if tt.args.Branches && len(output.Branches) == 0 {
					t.Errorf("expected branches, got none")
				}
				if tt.args.Tags && len(output.Tags) == 0 {
					t.Errorf("expected tags, got none")
				}
			}
		})
	}
}
