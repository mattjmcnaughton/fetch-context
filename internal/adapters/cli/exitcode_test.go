package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
)

func TestExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil is success", nil, 0},
		{"runtime error", errors.New("clone failed"), 1},
		{"usage error", usageerr.New("a subcommand is required"), 2},
		{"wrapped usage error", fmt.Errorf("load: %w", usageerr.Newf("unknown profile %q", "x")), 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ExitCode(c.err); got != c.want {
				t.Errorf("ExitCode(%v) = %d, want %d", c.err, got, c.want)
			}
		})
	}
}
