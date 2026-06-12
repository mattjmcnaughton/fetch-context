package github

// The twin-pattern contract body (docs/testing.md): the same assertions run
// against forgemock (always-on, integration tag) and against the real
// api.github.com (opt-in, contract tag). Shape and protocol only, no data.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/forge/linkheader"
)

// runGitHubForgeContract asserts the protocol surface the adapter relies
// on: 200 + JSON array with name/clone_url, a parseable pagination Link
// header, a differing second page, and 401 for a bad token.
func runGitHubForgeContract(t *testing.T, baseURL, org, token string) {
	t.Helper()
	listURL := fmt.Sprintf("%s/orgs/%s/repos?per_page=2", baseURL, org)

	resp, page1 := getRepoPage(t, listURL, token)
	if len(page1) == 0 {
		t.Fatalf("org %q returned no repos; the contract needs a populated org", org)
	}
	for i, item := range page1 {
		if asString(item["name"]) == "" {
			t.Errorf("item %d: field 'name' missing or empty: %v", i, item)
		}
		if asString(item["clone_url"]) == "" {
			t.Errorf("item %d: field 'clone_url' missing or empty: %v", i, item)
		}
	}

	next := linkheader.Next(resp.Header.Get("Link"))
	if next == "" {
		t.Fatalf("no rel=\"next\" in Link header %q; the contract needs an org with >2 repos", resp.Header.Get("Link"))
	}
	_, page2 := getRepoPage(t, next, token)
	if len(page2) == 0 {
		t.Fatal("second page is empty")
	}
	if asString(page1[0]["name"]) == asString(page2[0]["name"]) {
		t.Error("second page does not differ from the first")
	}

	badResp := doGet(t, listURL, "definitely-not-a-valid-token")
	defer badResp.Body.Close()
	if badResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("bad token: status = %d, want 401", badResp.StatusCode)
	}
}

func getRepoPage(t *testing.T, url, token string) (*http.Response, []map[string]any) {
	t.Helper()
	resp := doGet(t, url, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %s, want 200", url, resp.Status)
	}
	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("GET %s: response is not a JSON array: %v", url, err)
	}
	return resp, items
}

func doGet(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
