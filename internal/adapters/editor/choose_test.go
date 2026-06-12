package editor

import (
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/envx"
)

func TestChooseEditorPrecedence(t *testing.T) {
	cases := []struct {
		name string
		env  envx.Fake
		want string
	}{
		{"VISUAL wins over EDITOR", envx.Fake{"VISUAL": "vis", "EDITOR": "ed"}, "vis"},
		{"EDITOR when no VISUAL", envx.Fake{"EDITOR": "ed"}, "ed"},
		{"vi as final fallback", envx.Fake{}, "vi"},
		{"empty VISUAL falls through", envx.Fake{"VISUAL": "", "EDITOR": "ed"}, "ed"},
		{"empty both falls to vi", envx.Fake{"VISUAL": "", "EDITOR": ""}, "vi"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := chooseEditor(c.env); got != c.want {
				t.Errorf("chooseEditor(%v) = %q, want %q", c.env, got, c.want)
			}
		})
	}
}
