//go:build acceptance
// +build acceptance

package cmd_test

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/google/go-github/v72/github"
	"github.com/nexthink-oss/ghup/cmd"
	"github.com/nexthink-oss/ghup/internal/util"
)

type updateRefTestArgs struct {
	Source         string
	SourceType     string
	TargetType     string
	Targets        []string
	Force          bool
	AdditionalArgs []string
}

func (s *updateRefTestArgs) Slice() []string {
	args := []string{"update-ref"}

	if s.Source != "" {
		args = append(args, "--source", s.Source)
	}

	if s.SourceType != "" {
		args = append(args, "--source-type", s.SourceType)
	}

	if s.TargetType != "" {
		args = append(args, "--target-type", s.TargetType)
	}

	if s.Force {
		args = append(args, "--force")
	}

	args = append(args, s.Targets...)
	args = append(args, s.AdditionalArgs...)

	return args
}

func TestAccUpdateRefCmd(t *testing.T) {
	client, resources := setupTestResources(t)

	// Get default branch for tests
	repoInfo, err := client.GetRepositoryInfo("")
	if err != nil {
		t.Fatalf("failed to get repository info: %v", err)
	}
	defaultBranch := repoInfo.DefaultBranch.Name
	defaultSHA := repoInfo.DefaultBranch.Commit

	// Generate unique ref names for our tests
	testTag1 := "test-update-ref-tag-" + testRandomString(8)
	testTag2 := "test-update-ref-tag-" + testRandomString(8)
	testBranch1 := "test-update-ref-branch-" + testRandomString(8)
	testBranch2 := "test-update-ref-branch-" + testRandomString(8)

	// Register resources for cleanup
	resources.AddTag(testTag1)
	resources.AddTag(testTag2)
	resources.AddBranch(testBranch1)
	resources.AddBranch(testBranch2)

	// First we need to create some refs to update later
	// These would typically point to existing commits
	t.Logf("Creating initial test refs")

	// Pre-create test branch and tag pointing to default branch
	testBranch1Ref := "refs/heads/" + testBranch1
	testTag1Ref := "refs/tags/" + testTag1
	createTestRefs := []string{testBranch1Ref, testTag1Ref}
	for _, refName := range createTestRefs {
		ref := &github.Reference{
			Ref: github.Ptr(refName),
			Object: &github.GitObject{
				SHA: github.Ptr(string(defaultSHA)),
			},
		}

		_, err := client.CreateRef(ref)
		if err != nil {
			t.Fatalf("failed to create test ref %s: %v", refName, err)
		}
		t.Logf("Created test ref %s pointing to %s", refName, defaultSHA)
	}

	tests := []struct {
		name          string
		args          updateRefTestArgs
		wantError     bool
		wantStdout    *regexp.Regexp
		wantStderr    *regexp.Regexp
		checkJson     bool
		expectUpdated []bool // Whether each target in the test should be updated
	}{
		{
			name: "Idempotent branch update",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{testBranch1Ref},
			},
			checkJson:     true,
			expectUpdated: []bool{false}, // Should be false as target already points to the same commit
		},
		{
			name: "Idempotent tag update",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{testTag1Ref},
			},
			checkJson:     true,
			expectUpdated: []bool{false}, // Should be false as target already points to the same commit
		},
		{
			name: "New branch pointing to default branch",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{"refs/heads/" + testBranch2},
			},
			checkJson:     true,
			expectUpdated: []bool{true}, // New branch should be created
		},
		{
			name: "New tag pointing to default branch",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{"refs/tags/" + testTag2},
			},
			checkJson:     true,
			expectUpdated: []bool{true}, // New tag should be created
		},
		{
			name: "Update multiple refs at once",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{testTag1Ref, testBranch1Ref},
			},
			checkJson:     true,
			expectUpdated: []bool{false, false},
		},
		{
			name: "Update using a commit SHA as source",
			args: updateRefTestArgs{
				Source:  string(defaultSHA),
				Targets: []string{testTag1Ref},
				Force:   true,
			},
			checkJson:     true,
			expectUpdated: []bool{false},
		},
		{
			name: "Error - missing source",
			args: updateRefTestArgs{
				Source:  "",
				Targets: []string{testTag1Ref},
			},
			wantError:  true,
			wantStderr: regexp.MustCompile(`no source ref specified`),
		},
		{
			name: "Error - missing targets",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{},
			},
			wantError:  true,
			wantStderr: regexp.MustCompile(`no target refs specified`),
		},
		{
			name: "Error - non-existent source",
			args: updateRefTestArgs{
				Source:  "refs/heads/this-branch-does-not-exist-" + testRandomString(8),
				Targets: []string{testTag1Ref},
			},
			wantError: true,
			checkJson: true,
		},
		{
			name: "Source using different format (unqualified)",
			args: updateRefTestArgs{
				Source:     defaultBranch,
				SourceType: "heads",
				Targets:    []string{testTag1Ref},
				Force:      true,
			},
			checkJson:     true,
			expectUpdated: []bool{false},
		},
		{
			name: "Target using different format (unqualified)",
			args: updateRefTestArgs{
				Source:     defaultBranch,
				TargetType: "tags",
				Targets:    []string{testTag1}, // Unqualified tag name
				Force:      true,
			},
			checkJson:     true,
			expectUpdated: []bool{false},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			spec := testCmdSpec{
				Args: append([]string{"-vvvv"}, test.args.Slice()...),
			}

			tt.Logf("Running command with args: %v", spec.Args)

			stdout, stderr, err := testExecuteCmd(tt, spec)

			if os.Getenv("TEST_GHUP_LOG_OUTPUT") != "" {
				tt.Logf("stdout:\n%s", stdout.String())
				tt.Logf("stderr:\n%s", stderr.String())
			}

			if (err != nil) != test.wantError {
				tt.Errorf("gotErr=%v, wantError=%v", err, test.wantError)
			}

			if test.wantStdout != nil && !test.wantStdout.MatchString(stdout.String()) {
				tt.Errorf("stdout did not match expected pattern:\ngot: %q\nwant pattern: %q",
					stdout.String(), test.wantStdout)
			}

			if test.wantStderr != nil && !test.wantStderr.MatchString(stderr.String()) {
				tt.Errorf("stderr did not match expected pattern:\ngot: %q\nwant pattern: %q",
					stderr.String(), test.wantStderr)
			}

			if test.checkJson {
				var output cmd.UpdateRefOutput
				err := json.Unmarshal(stdout.Bytes(), &output)
				if err != nil {
					tt.Errorf("failed to unmarshal JSON output: %v", err)
				} else {
					// Verify source info
					if !test.wantError && output.Source.Commitish != test.args.Source {
						tt.Errorf("expected Source.Commitish=%q, got %q", test.args.Source, output.Source.Commitish)
					}

					if !test.wantError && !util.IsCommitHash(output.Source.SHA) {
						tt.Errorf("expected Source.SHA to be a commit hash, got %q", output.Source.SHA)
					}

					// Verify target info
					if !test.wantError && len(output.Target) != len(test.args.Targets) {
						tt.Errorf("expected %d targets in output, got %d", len(test.args.Targets), len(output.Target))
					}

					// Check if the targets were updated as expected
					for i, target := range output.Target {
						if i < len(test.expectUpdated) && test.expectUpdated[i] != target.Updated {
							tt.Errorf("target %d: expected Updated=%v, got %v", i, test.expectUpdated[i], target.Updated)
						}

						if !util.IsCommitHash(target.SHA) {
							tt.Errorf("target %d: expected SHA to be a commit hash, got %q", i, target.SHA)
						}
					}

					// Check error handling
					if test.wantError && output.Source.Error == "" && (len(output.Target) == 0 || output.Target[0].Error == "") {
						tt.Errorf("expected error message in JSON output, but got none")
					}
				}
			}
		})
	}
}
