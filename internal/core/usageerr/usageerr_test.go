package usageerr

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewAndMessage(t *testing.T) {
	err := New("a subcommand is required")
	if got := err.Error(); got != "a subcommand is required" {
		t.Errorf("Error() = %q", got)
	}
	err = Newf("unknown command %q", "frobnicate")
	if got := err.Error(); got != `unknown command "frobnicate"` {
		t.Errorf("Error() = %q", got)
	}
}

func TestIsUsage(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{"usage error", New("bad args"), true},
		{"wrapped usage error", fmt.Errorf("context: %w", New("bad args")), true},
		{"wrap of plain error", Wrap(errors.New("unknown flag")), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsUsage(c.err); got != c.want {
				t.Errorf("IsUsage(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

func TestWrapPreservesMessageAndUnwraps(t *testing.T) {
	inner := errors.New("unknown flag: --bogus")
	err := Wrap(inner)
	if err.Error() != inner.Error() {
		t.Errorf("Wrap changed the message: %q", err.Error())
	}
	if !errors.Is(err, inner) {
		t.Error("Wrap does not unwrap to the inner error")
	}
}
