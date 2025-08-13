//go:build acceptance
// +build acceptance

package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	NoCreateBranch bool
	BaseBranch     string
	PRTitle        string
	PRBody         string
	PRDraft        bool
	PRAutoMerge    string // New choice flag
	Force          bool
	DryRun         bool
	AllowEmpty     bool
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

	if s.NoCreateBranch {
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

	if s.PRAutoMerge != "" {
		args = append(args, "--pr-auto-merge", s.PRAutoMerge)
	}

	if s.Force {
		args = append(args, "--force")
	}

	if s.DryRun {
		args = append(args, "--dry-run")
	}

	if s.AllowEmpty {
		args = append(args, "--allow-empty")
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
	noChangeTestBranch := "test-noop-branch-" + testRandomString(8)
	emptyBranch := "test-empty-branch-" + testRandomString(8)
	emptyPRBranch := "test-empty-pr-" + testRandomString(8)
	// Add to resources for cleanup
	resources.AddBranch(testBranch)
	resources.AddBranch(testBranch + "-squash")
	resources.AddBranch(testBranch + "-rebase")
	resources.AddBranch(testBranch + "-off")
	resources.AddBranch(testBranch + "-compat")
	resources.AddBranch(emptyBranch)
	resources.AddBranch(emptyPRBranch)

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
		checkJson     bool
		expectUpdated bool
		expectPR      bool
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
			checkJson:     true,
			expectUpdated: false, // Files have not changed, so expect no update
		},
		{
			name: "Update a file with new content",
			args: contentTestArgs{
				Branch: testBranch,
				Updates: []string{
					file3 + ":test-path/file2.txt",
				},
				Message: "Updating file2 with new content",
			},
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
			checkJson:     true,
			expectUpdated: true,
			expectPR:      true,
		},
		{
			name: "PR creation is idempotent",
			args: contentTestArgs{
				Branch: testBranch,
				Updates: []string{
					file1 + ":test-path/update-for-pr.txt",
				},
				PRTitle: "Test PR from Content Command",
				PRBody:  "This PR was created by the acceptance test",
				Message: "Update for PR creation",
			},
			checkJson:     true,
			expectUpdated: false, // No changes, so no update
			expectPR:      true,
		},
		{
			name: "Create a PR with auto-merge enabled",
			args: contentTestArgs{
				Branch: testBranch + "-automerge",
				Updates: []string{
					file1 + ":test-path/automerge-test.txt",
				},
				PRTitle:     "Test PR with Auto-merge",
				PRBody:      "This PR was created with auto-merge enabled",
				PRAutoMerge: "merge",
				Message:     "Update for auto-merge PR test",
			},
			checkJson:     true,
			expectUpdated: true,
			expectPR:      true,
		},
		{
			name: "Create a PR with squash auto-merge",
			args: contentTestArgs{
				Branch: testBranch + "-squash",
				Updates: []string{
					file1 + ":test-path/squash-test.txt",
				},
				PRTitle:     "Test PR with Squash Auto-merge",
				PRBody:      "This PR was created with squash auto-merge enabled",
				PRAutoMerge: "squash",
				Message:     "Update for squash auto-merge PR test",
			},
			checkJson:     true,
			expectUpdated: true,
			expectPR:      true,
		},
		{
			name: "Create a PR with rebase auto-merge",
			args: contentTestArgs{
				Branch: testBranch + "-rebase",
				Updates: []string{
					file1 + ":test-path/rebase-test.txt",
				},
				PRTitle:     "Test PR with Rebase Auto-merge",
				PRBody:      "This PR was created with rebase auto-merge enabled",
				PRAutoMerge: "rebase",
				Message:     "Update for rebase auto-merge PR test",
			},
			checkJson:     true,
			expectUpdated: true,
			expectPR:      true,
		},
		{
			name: "Create a PR with auto-merge off explicitly",
			args: contentTestArgs{
				Branch: testBranch + "-off",
				Updates: []string{
					file1 + ":test-path/off-test.txt",
				},
				PRTitle:     "Test PR with Auto-merge Off",
				PRBody:      "This PR was created with auto-merge explicitly disabled",
				PRAutoMerge: "off",
				Message:     "Update for auto-merge off PR test",
			},
			checkJson:     true,
			expectUpdated: true,
			expectPR:      true,
		},
		{
			name: "No PR when there are no changes",
			args: contentTestArgs{
				BaseBranch: testBranch,
				Branch:     noChangeTestBranch,
				Updates: []string{
					file1 + ":test-path/file1.txt",
				},
				PRTitle: "Test PR from Content Command",
			},
			checkJson:     true,
			expectUpdated: false, // No changes, so no update
		},
		{
			name: "Try to update non-existent branch with create-branch=false flag",
			args: contentTestArgs{
				Branch: "non-existent-branch-" + testRandomString(6),
				Updates: []string{
					file1 + ":test-path/any-file.txt",
				},
				NoCreateBranch: true,
			},
			wantError: true,
			checkJson: true,
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
			checkJson:     true,
			expectUpdated: true, // Force should cause an update even with identical content
		},
		{
			name: "No changes without allow-empty should not create commit",
			args: contentTestArgs{
				Branch:  testBranch,
				Message: "This should not create a commit",
			},
			checkJson:     true,
			expectUpdated: false, // No changes and no allow-empty, so no update
		},
		{
			name: "Empty commit with allow-empty flag should create commit",
			args: contentTestArgs{
				Branch:     testBranch,
				AllowEmpty: true,
				Message:    "This is an empty commit",
			},
			checkJson:     true,
			expectUpdated: true, // Empty commit with allow-empty should create update
		},
		{
			name: "Empty commit on new branch with allow-empty",
			args: contentTestArgs{
				Branch:     emptyBranch,
				AllowEmpty: true,
				Message:    "Empty commit on new branch",
			},
			checkJson:     true,
			expectUpdated: true, // Should create new branch and empty commit
		},
		{
			name: "Empty commit with PR creation",
			args: contentTestArgs{
				Branch:     emptyPRBranch,
				AllowEmpty: true,
				PRTitle:    "Test PR with Empty Commit",
				PRBody:     "This PR was created with an empty commit",
				Message:    "Empty commit for PR test",
			},
			checkJson:     true,
			expectUpdated: true,
			expectPR:      true, // Should create PR even with empty commit
		},
		{
			name: "Dry run with allow-empty should not create commit",
			args: contentTestArgs{
				Branch:     testBranch,
				AllowEmpty: true,
				DryRun:     true,
				Message:    "Dry run empty commit",
			},
			checkJson:     true,
			expectUpdated: true, // Dry run returns true for updated but doesn't actually commit
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

			if test.checkJson {
				var output cmd.ContentOutput
				err := json.Unmarshal(stdout.Bytes(), &output)
				if err != nil {
					tt.Errorf("failed to unmarshal JSON output: %v", err)
				} else {
					if output.Updated != test.expectUpdated {
						tt.Errorf("expected Updated=%v, got %v", test.expectUpdated, output.Updated)
					}

					if test.wantError && output.ErrorMessage == "" {
						tt.Errorf("expected error message in JSON output, but got none")
					}

					if !test.wantError && output.ErrorMessage != "" {
						tt.Errorf("unexpected error message in JSON output: %s", output.ErrorMessage)
					}

					if test.expectPR && output.PullRequest == nil {
						tt.Errorf("expected pull request info in output, but got none")
					}
				}
			}
		})
	}
}
