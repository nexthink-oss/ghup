//go:build acceptance
// +build acceptance

package cmd_test

import (
	"bytes"
	"cmp"
	"context"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/nexthink-oss/ghup/cmd"
	"github.com/nexthink-oss/ghup/internal/remote"
)

// testResources tracks resources created during tests to help with cleanup
type testResources struct {
	client   *remote.Client
	branches []string
	tags     []string
}

// CleanupResources cleans up all tracked resources
func (r *testResources) CleanupResources(t *testing.T) {
	if r.client == nil {
		t.Error("cleanupResources: no client")
		return
	}

	t.Logf("cleanupResources for %s", t.Name())

	for _, tag := range r.tags {
		if err := r.client.DeleteRef("tags/" + tag); err != nil {
			t.Logf("Warning: Failed to clean up tag %q: %v", tag, err)
		} else {
			t.Logf("Cleaned up tag %q", tag)
		}
	}

	for _, branch := range r.branches {
		if err := r.client.DeleteRef("heads/" + branch); err != nil {
			t.Logf("Warning: Failed to clean up branch %q: %v", branch, err)
		} else {
			t.Logf("Cleaned up branch %q", branch)
		}
	}
}

// AddTag adds a tag to be cleaned up
func (r *testResources) AddTag(tag string) {
	r.tags = append(r.tags, tag)
}

// AddBranch adds a branch to be cleaned up
func (r *testResources) AddBranch(branch string) {
	r.branches = append(r.branches, branch)
}

// setupTestEnvironment sets up the test environment and returns resources to track cleanup
func setupTestEnvironment(t *testing.T) *remote.Client {
	setupIssues := make([]string, 0)

	testToken := os.Getenv("TEST_GHUP_TOKEN")
	if testToken == "" {
		setupIssues = append(setupIssues, "TEST_GHUP_TOKEN is not set")
	}

	testOwner := os.Getenv("TEST_GHUP_OWNER")
	if testOwner == "" {
		setupIssues = append(setupIssues, "TEST_GHUP_OWNER is not set")
	}

	testRepo := os.Getenv("TEST_GHUP_REPO")
	if testRepo == "" {
		setupIssues = append(setupIssues, "TEST_GHUP_REPO is not set")
	}

	if len(setupIssues) > 0 {
		t.Fatalf("setup issues: \n%s", strings.Join(setupIssues, "\n"))
	}

	testBranch := cmp.Or(os.Getenv("TEST_GHUP_BRANCH"), "main")

	t.Setenv("GHUP_TOKEN", testToken)
	t.Setenv("GHUP_OWNER", testOwner)
	t.Setenv("GHUP_REPO", testRepo)
	t.Setenv("GHUP_BRANCH", testBranch)

	repo := remote.Repo{
		Owner: testOwner,
		Name:  testRepo,
	}

	client, err := remote.NewClient(context.Background(), &repo)
	if err != nil {
		t.Fatalf("NewClient(%s): %v", repo, err)
	}

	return client
}

// Create test resource manager and register cleanup function
func setupTestResources(t *testing.T) (*remote.Client, *testResources) {
	client := setupTestEnvironment(t)

	resources := &testResources{
		client: client,
	}

	// Register cleanup function to run at the end of the test
	t.Cleanup(func() {
		resources.CleanupResources(t)
	})

	return client, resources
}

type testArgSpec interface {
	GetArgs() []string
}

type testCmdSpec struct {
	Env  map[string]string
	Args []string
}

func testExecuteCmd(t *testing.T, spec testCmdSpec) (stdout, stderr bytes.Buffer, err error) {
	cmd := cmd.New()

	cmd.SetArgs(spec.Args)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	// cmd.SetContext(t.Context())

	for k, v := range spec.Env {
		t.Setenv(k, v)
	}

	return stdout, stderr, cmd.Execute()
}

func testRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
