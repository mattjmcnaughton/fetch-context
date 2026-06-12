//go:build contract

package github

import (
	"os"
	"testing"
)

// The opt-in twin of TestGitHubForgeContractAgainstMock: the same contract
// body pointed at the real api.github.com, to catch drift between the mock
// and the real API. Requires $GITHUB_TOKEN; run via `just test-contract`.
func TestGitHubForgeContractAgainstRealAPI(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set; contract test needs a real credential")
	}
	org := os.Getenv("FC_CONTRACT_GITHUB_ORG")
	if org == "" {
		org = "spf13" // arbitrary public org with >2 repos (docs/testing.md)
	}
	runGitHubForgeContract(t, DefaultBase, org, token)
}
