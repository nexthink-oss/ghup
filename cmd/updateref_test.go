//go:build acceptance
// +build acceptance

package cmd_test

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/google/go-github/v70/github"
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

	// Get default branch SHA
	defaultSHA, err := client.ResolveCommitish(defaultBranch)
	if err != nil {
		t.Fatalf("failed to resolve default branch commitish: %v", err)
	}

	// Create test branch and tag pointing to default branch
	testBranch1Ref := "refs/heads/" + testBranch1
	testTag1Ref := "refs/tags/" + testTag1
	createTestRefs := []string{testBranch1Ref, testTag1Ref}
	for _, refName := range createTestRefs {
		ref := &github.Reference{
			Ref: github.Ptr(refName),
			Object: &github.GitObject{
				SHA: github.Ptr(defaultSHA),
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
			name: "Update a tag to match default branch",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{testTag1Ref},
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: []bool{false}, // Should be false as target already points to the same commit
		},
		{
			name: "Force update a tag to match default branch",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{testTag1Ref},
				Force:   true,
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: []bool{true}, // With force, it should update even if already pointing to the same commit
		},
		{
			name: "Create a new tag pointing to default branch",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{"refs/tags/" + testTag2},
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: []bool{true}, // New tag should be created
		},
		{
			name: "Create a new branch pointing to default branch",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{"refs/heads/" + testBranch2},
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: []bool{true}, // New branch should be created
		},
		{
			name: "Update multiple refs at once",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{testTag1Ref, testBranch1Ref},
				Force:   true,
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: []bool{true, true}, // Both should be updated with force flag
		},
		{
			name: "Update using a commit SHA as source",
			args: updateRefTestArgs{
				Source:  defaultSHA,
				Targets: []string{testTag1Ref},
				Force:   true,
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: []bool{true},
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
			name: "Error - invalid target ref",
			args: updateRefTestArgs{
				Source:  defaultBranch,
				Targets: []string{"invalid/ref/format"},
			},
			wantError: true,
		},
		{
			name: "Source using different format (unqualified)",
			args: updateRefTestArgs{
				Source:     defaultBranch,
				SourceType: "heads",
				Targets:    []string{testTag1Ref},
				Force:      true,
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: []bool{true},
		},
		{
			name: "Target using different format (unqualified)",
			args: updateRefTestArgs{
				Source:     defaultBranch,
				TargetType: "tags",
				Targets:    []string{testTag1}, // Unqualified tag name
				Force:      true,
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: []bool{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := testCmdSpec{
				Args: append([]string{"-vvvv"}, tt.args.Slice()...),
			}

			t.Logf("Running command with args: %v", spec.Args)

			stdout, stderr, err := testExecuteCmd(t, spec)

			if os.Getenv("TEST_GHUP_LOG_OUTPUT") != "" {
				t.Logf("stdout:\n%s", stdout.String())
				t.Logf("stderr:\n%s", stderr.String())
			}

			if (err != nil) != tt.wantError {
				t.Errorf("gotErr=%v, wantError=%v", err, tt.wantError)
			}

			if tt.wantStdout != nil && !tt.wantStdout.MatchString(stdout.String()) {
				t.Errorf("stdout did not match expected pattern:\ngot: %q\nwant pattern: %q",
					stdout.String(), tt.wantStdout)
			}

			if tt.wantStderr != nil && !tt.wantStderr.MatchString(stderr.String()) {
				t.Errorf("stderr did not match expected pattern:\ngot: %q\nwant pattern: %q",
					stderr.String(), tt.wantStderr)
			}

			if tt.checkJson {
				var output cmd.UpdateRefOutput
				err := json.Unmarshal(stdout.Bytes(), &output)
				if err != nil {
					t.Errorf("failed to unmarshal JSON output: %v", err)
				} else {
					// Verify source info
					if !tt.wantError && output.Source.Commitish != tt.args.Source {
						t.Errorf("expected Source.Commitish=%q, got %q", tt.args.Source, output.Source.Commitish)
					}

					if !tt.wantError && !util.IsCommitHash(output.Source.SHA) {
						t.Errorf("expected Source.SHA to be a commit hash, got %q", output.Source.SHA)
					}

					// Verify target info
					if !tt.wantError && len(output.Target) != len(tt.args.Targets) {
						t.Errorf("expected %d targets in output, got %d", len(tt.args.Targets), len(output.Target))
					}

					// Check if the targets were updated as expected
					for i, target := range output.Target {
						if i < len(tt.expectUpdated) && tt.expectUpdated[i] != target.Updated {
							t.Errorf("target %d: expected Updated=%v, got %v", i, tt.expectUpdated[i], target.Updated)
						}

						if !util.IsCommitHash(target.SHA) {
							t.Errorf("target %d: expected SHA to be a commit hash, got %q", i, target.SHA)
						}
					}

					// Check error handling
					if tt.wantError && output.Source.Error == "" && (len(output.Target) == 0 || output.Target[0].Error == "") {
						t.Errorf("expected error message in JSON output, but got none")
					}
				}
			}
		})
	}
}
