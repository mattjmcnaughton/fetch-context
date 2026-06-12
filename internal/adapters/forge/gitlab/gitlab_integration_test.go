//go:build integration

package gitlab

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/testing/forgemock"
)

func seededMock(t *testing.T) *forgemock.Server {
	t.Helper()
	mock := forgemock.New()
	t.Cleanup(mock.Close)
	mock.SeedGitLabGroup("acme/platform", []forgemock.Repo{
		{Path: "top", CloneURL: "http://git.test/acme/platform/top.git"},
		{Path: "sub/nested", CloneURL: "http://git.test/acme/platform/sub/nested.git"},
		{Path: "other", CloneURL: "http://git.test/acme/platform/other.git"},
	})
	return mock
}

// The always-on twin: if the mock cannot pass the contract body, the mock
// is broken.
func TestGitLabForgeContractAgainstMock(t *testing.T) {
	mock := seededMock(t)
	runGitLabForgeContract(t, mock.URL(), "acme/platform", "")
}

func TestEnumerateRecursesSubgroupsAndWalksPages(t *testing.T) {
	mock := seededMock(t)
	mock.SetMaxPageSize(2) // 3 projects → 2 pages

	repos, err := New(mock.URL(), "", slog.New(slog.DiscardHandler)).Enumerate(context.Background(), "acme/platform")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 3 {
		t.Fatalf("Enumerate returned %d repos, want 3 across pages", len(repos))
	}
	paths := map[string]string{}
	for _, r := range repos {
		paths[r.Path] = r.CloneURL
	}
	if paths["sub/nested"] != "http://git.test/acme/platform/sub/nested.git" {
		t.Errorf("subgroup path not preserved: %+v", repos)
	}
}

func TestEnumeratePrivateGroupTokenInjection(t *testing.T) {
	mock := seededMock(t)
	mock.SeedGitLabGroup("private-group", []forgemock.Repo{
		{Path: "secret", CloneURL: "http://git.test/private-group/secret.git"},
	})
	mock.RequireGitLabToken("private-group", "glpat-123")
	ctx := context.Background()
	log := slog.New(slog.DiscardHandler)

	repos, err := New(mock.URL(), "glpat-123", log).Enumerate(ctx, "private-group")
	if err != nil {
		t.Fatalf("with token: %v", err)
	}
	if len(repos) != 1 || repos[0].Path != "secret" {
		t.Errorf("repos = %+v", repos)
	}

	_, err = New(mock.URL(), "", log).Enumerate(ctx, "private-group")
	if err == nil {
		t.Fatal("without token: want error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "auth") {
		t.Errorf("error %q should name the auth problem", err)
	}
}
