//go:build integration

package github

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
	mock.SeedGitHubOrg("fixture-org", []forgemock.Repo{
		{Path: "r1", CloneURL: "http://git.test/fixture-org/r1.git"},
		{Path: "r2", CloneURL: "http://git.test/fixture-org/r2.git"},
		{Path: "r3", CloneURL: "http://git.test/fixture-org/r3.git"},
		{Path: "r4", CloneURL: "http://git.test/fixture-org/r4.git"},
		{Path: "r5", CloneURL: "http://git.test/fixture-org/r5.git"},
	})
	return mock
}

// The always-on twin: if the mock cannot pass the contract body, the mock
// is broken.
func TestGitHubForgeContractAgainstMock(t *testing.T) {
	mock := seededMock(t)
	runGitHubForgeContract(t, mock.URL(), "fixture-org", "")
}

func TestEnumerateWalksAllPages(t *testing.T) {
	mock := seededMock(t)
	mock.SetMaxPageSize(2) // 5 repos → 3 pages

	repos, err := New(mock.URL(), "", slog.New(slog.DiscardHandler)).Enumerate(context.Background(), "fixture-org")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 5 {
		t.Fatalf("Enumerate returned %d repos, want 5 across all pages", len(repos))
	}
	if repos[0].Path != "r1" || repos[4].Path != "r5" {
		t.Errorf("repos out of order: %+v", repos)
	}
	if repos[0].CloneURL != "http://git.test/fixture-org/r1.git" {
		t.Errorf("clone URL = %q", repos[0].CloneURL)
	}
}

func TestEnumeratePrivateOrgTokenInjection(t *testing.T) {
	mock := seededMock(t)
	mock.SeedGitHubOrg("private-org", []forgemock.Repo{
		{Path: "secret", CloneURL: "http://git.test/private-org/secret.git"},
	})
	mock.RequireGitHubToken("private-org", "tok-123")
	ctx := context.Background()
	log := slog.New(slog.DiscardHandler)

	repos, err := New(mock.URL(), "tok-123", log).Enumerate(ctx, "private-org")
	if err != nil {
		t.Fatalf("with token: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("repos = %+v, want the private repo", repos)
	}

	_, err = New(mock.URL(), "", log).Enumerate(ctx, "private-org")
	if err == nil {
		t.Fatal("without token: want error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "auth") {
		t.Errorf("error %q should name the auth problem", err)
	}
}
