//go:build contract

package gitlab

import (
	"os"
	"testing"
)

// The opt-in twin of TestGitLabForgeContractAgainstMock: the same contract
// body pointed at the real gitlab.com API, to catch drift between the mock
// and the real API. Requires $GITLAB_TOKEN; run via `just test-contract`.
func TestGitLabForgeContractAgainstRealAPI(t *testing.T) {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		t.Skip("GITLAB_TOKEN not set; contract test needs a real credential")
	}
	group := os.Getenv("FC_CONTRACT_GITLAB_GROUP")
	if group == "" {
		group = "gitlab-org" // arbitrary public group with >2 projects
	}
	runGitLabForgeContract(t, DefaultBase, group, token)
}
