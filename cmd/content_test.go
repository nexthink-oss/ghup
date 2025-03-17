//go:build acceptance
// +build acceptance

package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/nexthink-oss/ghup/cmd"
)

type contentTestArgs struct {
	Branch         string
	Updates        []string
	Deletes        []string
	Copies         []string
	Staged         bool
	Tracked        bool
	CreateBranch   bool
	BaseBranch     string
	PRTitle        string
	PRBody         string
	PRDraft        bool
	Force          bool
	DryRun         bool
	Message        string
	AdditionalArgs []string
}

func (s *contentTestArgs) Slice() []string {
	args := []string{"content"}

	if s.Branch != "" {
		args = append(args, "--branch", s.Branch)
	}

	if s.BaseBranch != "" {
		args = append(args, "--base-branch", s.BaseBranch)
	}

	if s.Message != "" {
		args = append(args, "--message", s.Message)
	}

	for _, update := range s.Updates {
		args = append(args, "--update", update)
	}

	for _, delete := range s.Deletes {
		args = append(args, "--delete", delete)
	}

	for _, copy := range s.Copies {
		args = append(args, "--copy", copy)
	}

	if s.Staged {
		args = append(args, "--staged")
	}

	if s.Tracked {
		args = append(args, "--tracked")
	}

	if !s.CreateBranch {
		args = append(args, "--create-branch=false")
	}

	if s.PRTitle != "" {
		args = append(args, "--pr-title", s.PRTitle)
	}

	if s.PRBody != "" {
		args = append(args, "--pr-body", s.PRBody)
	}

	if s.PRDraft {
		args = append(args, "--pr-draft")
	}

	if s.Force {
		args = append(args, "--force")
	}

	if s.DryRun {
		args = append(args, "--dry-run")
	}

	args = append(args, s.AdditionalArgs...)

	return args
}

func TestAccContentCmd(t *testing.T) {
	client, resources := setupTestResources(t)

	// We'll create ephemeral files in a temp dir to test "content" updates
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	err := os.WriteFile(file1, []byte("file1 content"), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	file2 := filepath.Join(tmpDir, "file2.txt")
	err = os.WriteFile(file2, []byte("file2 content"), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	file3 := filepath.Join(tmpDir, "file3.txt")
	err = os.WriteFile(file3, []byte("file3 content for later updates"), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fileToDelete := filepath.Join(tmpDir, "delete_me.txt")
	err = os.WriteFile(fileToDelete, []byte("this file will be deleted"), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Generate a unique branch name for our tests
	testBranch := "test-content-" + testRandomString(8)
	// Add to resources for cleanup
	resources.AddBranch(testBranch)

	// Get default branch for tests
	repoInfo, err := client.GetRepositoryInfo("")
	if err != nil {
		t.Fatalf("failed to get repository info: %v", err)
	}
	defaultBranch := repoInfo.DefaultBranch.Name

	tests := []struct {
		name          string
		args          contentTestArgs
		wantError     bool
		wantStdout    *regexp.Regexp
		wantStderr    *regexp.Regexp
		checkJson     bool
		expectUpdated bool
	}{
		{
			name: "Create a new branch with content updates",
			args: contentTestArgs{
				Branch: testBranch,
				Updates: []string{
					file1 + ":test-path/file1.txt",
					file2 + ":test-path/file2.txt",
				},
				Message: "Initial content test commit",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: true,
		},
		{
			name: "Update existing files (idempotent operation)",
			args: contentTestArgs{
				Branch: testBranch,
				Updates: []string{
					file1 + ":test-path/file1.txt",
					file2 + ":test-path/file2.txt",
				},
				Message: "Same content should be idempotent",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: false, // Files have not changed, so expect no update
		},
		{
			name: "Update a file with new content",
			args: contentTestArgs{
				Branch: testBranch,
				Updates: []string{
					file3 + ":test-path/file3.txt",
				},
				Message: "Adding a third file",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: true,
		},
		{
			name: "Delete a file",
			args: contentTestArgs{
				Branch: testBranch,
				Deletes: []string{
					"test-path/file2.txt",
				},
				Message: "Deleting the second file",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: true,
		},
		{
			name: "Copy a file from the same branch",
			args: contentTestArgs{
				Branch: testBranch,
				Copies: []string{
					testBranch + ":test-path/file1.txt:test-path/file1-copy.txt",
				},
				Message: "Copying a file within the branch",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: true,
		},
		{
			name: "Copy a file from default branch",
			args: contentTestArgs{
				Branch: testBranch,
				Copies: []string{
					defaultBranch + ":README.md:test-path/readme-copy.md",
				},
				Message: "Copying a file from the default branch",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: true,
		},
		{
			name: "Update, copy, and delete in one operation",
			args: contentTestArgs{
				Branch: testBranch,
				Updates: []string{
					fileToDelete + ":test-path/new-file.txt",
				},
				Copies: []string{
					testBranch + ":test-path/file1.txt:test-path/another-copy.txt",
				},
				Deletes: []string{
					"test-path/file3.txt",
				},
				Message: "Combined operation",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: true,
		},
		{
			name: "Create a PR from our changes",
			args: contentTestArgs{
				Branch: testBranch,
				Updates: []string{
					file1 + ":test-path/update-for-pr.txt",
				},
				PRTitle: "Test PR from Content Command",
				PRBody:  "This PR was created by the acceptance test",
				Message: "Update for PR creation",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: true,
			wantStdout:    regexp.MustCompile(`"pullrequest"`),
		},
		{
			name: "Try to update non-existent branch without create flag",
			args: contentTestArgs{
				Branch: "non-existent-branch-" + testRandomString(6),
				Updates: []string{
					file1 + ":test-path/any-file.txt",
				},
				CreateBranch: false,
			},
			wantError:  true,
			checkJson:  true,
			wantStdout: regexp.MustCompile(`"error":`),
		},
		{
			name: "Dry run mode should not create changes",
			args: contentTestArgs{
				Branch: testBranch,
				Updates: []string{
					file1 + ":test-path/dry-run-file.txt",
				},
				DryRun:  true,
				Message: "This commit should not happen",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: true, // The command returns true for updated, but no actual change happens
		},
		{
			name: "Force update even if file content is the same",
			args: contentTestArgs{
				Branch: testBranch,
				Updates: []string{
					file1 + ":test-path/file1.txt", // Same content as before
				},
				Force:   true,
				Message: "Force update with same content",
			},
			wantError:     false,
			checkJson:     true,
			expectUpdated: true, // Force should cause an update even with identical content
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
				var output cmd.ContentOutput
				err := json.Unmarshal(stdout.Bytes(), &output)
				if err != nil {
					t.Errorf("failed to unmarshal JSON output: %v", err)
				} else {
					if output.Updated != tt.expectUpdated {
						t.Errorf("expected Updated=%v, got %v", tt.expectUpdated, output.Updated)
					}

					if tt.wantError && output.ErrorMessage == "" {
						t.Errorf("expected error message in JSON output, but got none")
					}

					if !tt.wantError && output.ErrorMessage != "" {
						t.Errorf("unexpected error message in JSON output: %s", output.ErrorMessage)
					}

					// Check PR output if it was created
					if tt.args.PRTitle != "" && output.PullRequest == nil {
						t.Errorf("expected pull request info in output, but got none")
					}
				}
			}
		})
	}
}
