package gitlab

// The twin-pattern contract body (docs/testing.md): the same assertions run
// against forgemock (always-on, integration tag) and against the real
// gitlab.com API (opt-in, contract tag). Shape and protocol only, no data.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/forge/linkheader"
)

// runGitLabForgeContract asserts the protocol surface the adapter relies
// on: 200 + JSON array with path_with_namespace/http_url_to_repo (with
// include_subgroups accepted), a parseable pagination Link header, a
// differing second page, and 401 for a bad token.
func runGitLabForgeContract(t *testing.T, baseURL, group, token string) {
	t.Helper()
	listURL := fmt.Sprintf("%s/groups/%s/projects?per_page=2&include_subgroups=true",
		baseURL, url.PathEscape(group))

	resp, page1 := getProjectPage(t, listURL, token)
	if len(page1) == 0 {
		t.Fatalf("group %q returned no projects; the contract needs a populated group", group)
	}
	for i, item := range page1 {
		if asString(item["path_with_namespace"]) == "" {
			t.Errorf("item %d: field 'path_with_namespace' missing or empty: %v", i, item)
		}
		if asString(item["http_url_to_repo"]) == "" {
			t.Errorf("item %d: field 'http_url_to_repo' missing or empty: %v", i, item)
		}
	}

	next := linkheader.Next(resp.Header.Get("Link"))
	if next == "" {
		t.Fatalf("no rel=\"next\" in Link header %q; the contract needs a group with >2 projects", resp.Header.Get("Link"))
	}
	_, page2 := getProjectPage(t, next, token)
	if len(page2) == 0 {
		t.Fatal("second page is empty")
	}
	if asString(page1[0]["path_with_namespace"]) == asString(page2[0]["path_with_namespace"]) {
		t.Error("second page does not differ from the first")
	}

	badResp := doGet(t, listURL, "definitely-not-a-valid-token")
	defer badResp.Body.Close()
	if badResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("bad token: status = %d, want 401", badResp.StatusCode)
	}
}

func getProjectPage(t *testing.T, url, token string) (*http.Response, []map[string]any) {
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
	if token != "" {
		req.Header.Set("PRIVATE-TOKEN", token)
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
