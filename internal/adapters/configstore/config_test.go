package configstore

import "testing"

func TestDefaultTargetWhenNoConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Target != ".agentic/sources" {
		t.Errorf("Default().Target = %q, want .agentic/sources (AC-CONFIG-04)", cfg.Target)
	}
}
