package fakes

import "context"

// FakeMaterializer is the generic shape of the three materializer fakes
// LoadProfile's unit tests bind (FakeMaterializer[materialize.RepoRequest]
// etc.). Being generic keeps this package free of a core/materialize import,
// which the materialize tests' own use of these fakes would turn into a
// cycle.
type FakeMaterializer[Req any] struct {
	Requests []Req
	Err      error
}

func (f *FakeMaterializer[Req]) Materialize(_ context.Context, req Req) error {
	f.Requests = append(f.Requests, req)
	return f.Err
}
