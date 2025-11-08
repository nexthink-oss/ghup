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

type deploymentTestArgs struct {
	Environment    string
	Commitish      string
	State          string
	Transient      bool
	Production     bool
	Description    string
	EnvironmentURL string
	DryRun         bool
}

func (s *deploymentTestArgs) Slice() []string {
	args := []string{"deployment"}

	if s.Environment != "" {
		args = append(args, "--environment", s.Environment)
	}

	if s.Commitish != "" {
		args = append(args, "--commitish", s.Commitish)
	}

	if s.State != "" {
		args = append(args, "--state", s.State)
	}

	if s.Transient {
		args = append(args, "--transient")
	}

	if s.Production {
		args = append(args, "--production")
	}

	if s.Description != "" {
		args = append(args, "--description", s.Description)
	}

	if s.EnvironmentURL != "" {
		args = append(args, "--environment-url", s.EnvironmentURL)
	}

	if s.DryRun {
		args = append(args, "--dry-run")
	}

	return args
}

func TestAccDeploymentCmd(t *testing.T) {
	client, _ := setupTestResources(t)

	// Generate unique environment names for our tests
	testSuffix := testRandomString(6)
	prodEnv := "prod-" + testSuffix
	stagingEnv := "staging-" + testSuffix

	// Get main branch SHA for tests
	mainSha, err := client.ResolveCommitish("main")
	if err != nil {
		t.Fatalf("failed to resolve main commitish: %v", err)
	}

	tests := []struct {
		name        string
		args        deploymentTestArgs
		wantError   bool
		wantStdout  *regexp.Regexp
		wantStderr  *regexp.Regexp
		checkJson   bool
		jsonCreated bool
	}{
		{
			name:       "no args - missing environment",
			wantStderr: regexp.MustCompile(`Error: environment is required`),
			wantError:  true,
		},
		{
			name: "invalid state",
			args: deploymentTestArgs{
				Environment: stagingEnv,
				State:       "invalid",
			},
			wantStderr: regexp.MustCompile(`Error: invalid argument "invalid" for "--state" flag: valid choices are: \[success, pending, failure, error, in_progress, queued, inactive\]`),
			wantError:  true,
		},
		{
			name: "create deployment with success state",
			args: deploymentTestArgs{
				Environment: stagingEnv,
				State:       "success",
				Description: "Test deployment",
			},
			checkJson:   true,
			jsonCreated: true,
		},
		{
			name: "create deployment with failure state",
			args: deploymentTestArgs{
				Environment: prodEnv,
				State:       "failure",
				Production:  true,
				Description: "Production deployment",
			},
			checkJson:   true,
			jsonCreated: true,
		},
		{
			name: "create transient deployment",
			args: deploymentTestArgs{
				Environment: "temp-env-" + testSuffix,
				State:       "success",
				Transient:   true,
				Description: "Temporary environment",
			},
			checkJson:   true,
			jsonCreated: true,
		},
		{
			name: "deployment with specific commitish",
			args: deploymentTestArgs{
				Environment: "commit-env-" + testSuffix,
				Commitish:   "main",
				State:       "success",
			},
			checkJson:   true,
			jsonCreated: true,
		},
		{
			name: "deployment with all optional fields",
			args: deploymentTestArgs{
				Environment:    "full-env-" + testSuffix,
				State:          "in_progress",
				Description:    "Full deployment test",
				EnvironmentURL: "https://staging.example.com",
			},
			checkJson:   true,
			jsonCreated: true,
		},
		{
			name: "dry-run deployment",
			args: deploymentTestArgs{
				Environment: "dryrun-env-" + testSuffix,
				State:       "success",
				Description: "Dry run test",
				DryRun:      true,
			},
			checkJson:   true,
			jsonCreated: true,
		},
		{
			name: "dry-run production deployment",
			args: deploymentTestArgs{
				Environment: "dryrun-prod-" + testSuffix,
				State:       "error",
				Production:  true,
				Description: "Dry run production test",
				DryRun:      true,
			},
			checkJson:   true,
			jsonCreated: true,
		},
		{
			name: "inactive deployment state",
			args: deploymentTestArgs{
				Environment: "inactive-env-" + testSuffix,
				State:       "inactive",
				Description: "Inactive deployment test",
			},
			checkJson:   true,
			jsonCreated: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// Set default commitish to main
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

			if test.wantStderr != nil && !test.wantStderr.Match(stderr) {
				tt.Errorf("unexpected stderr: got %q, want %q", string(stderr), test.wantStderr)
			}

			if test.wantStdout != nil && !test.wantStdout.Match(stdout) {
				tt.Errorf("unexpected stdout: got %q, want %q", string(stdout), test.wantStdout)
			}

			// Validate the structured output
			if test.checkJson && !test.wantError {
				var output cmd.DeploymentOutput
				unmarshalErr := json.Unmarshal(stdout, &output)
				if unmarshalErr != nil {
					tt.Errorf("failed to unmarshal output: %v", unmarshalErr)
				} else {
					if output.Environment != test.args.Environment {
						tt.Errorf("unexpected environment: got %q, want %q", output.Environment, test.args.Environment)
					}
					if test.args.State != "" && output.State != test.args.State {
						tt.Errorf("unexpected state: got %q, want %q", output.State, test.args.State)
					}
					if !util.IsCommitHash(output.SHA) {
						tt.Errorf("unexpected SHA: got %q", output.SHA)
					}
					if output.URL != client.GetCommitURL(output.SHA) {
						tt.Errorf("unexpected URL: got %q, want %q", output.URL, client.GetCommitURL(output.SHA))
					}
					if output.DeploymentID == 0 {
						tt.Errorf("expected deployment ID to be set")
					}
					if output.StatusID == 0 {
						tt.Errorf("expected status ID to be set")
					}
					if output.Created != test.jsonCreated {
						tt.Errorf("unexpected created status: got %v, want %v", output.Created, test.jsonCreated)
					}
					if test.args.Commitish != "" {
						if output.Commitish != test.args.Commitish {
							tt.Errorf("unexpected commitish: got %q, want %q", output.Commitish, test.args.Commitish)
						}
					}
					if output.SHA != mainSha {
						tt.Errorf("unexpected SHA: got %q, want %q", output.SHA, mainSha)
					}
				}
			}
		})
	}
}
