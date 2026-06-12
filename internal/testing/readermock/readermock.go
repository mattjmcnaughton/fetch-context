// Package readermock mocks the jina reader proxy as an httptest.Server: it
// returns canned markdown containing ACCEPTANCE-MARKER for any wrapped URL
// and records request URIs so tests can assert the literal wrapping
// (AC-URL-04).
package readermock

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
)

// Marker appears in every page the mock returns.
const Marker = "ACCEPTANCE-MARKER"

type Server struct {
	httpSrv *httptest.Server

	mu       sync.Mutex
	requests []string
}

func New() *Server {
	s := &Server{}
	s.httpSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r.RequestURI)
		s.mu.Unlock()
		fmt.Fprintf(w, "%s\n\n# canned page for %s\n", Marker, r.RequestURI)
	}))
	return s
}

func (s *Server) Close() { s.httpSrv.Close() }

// URL is the mock's base URL (the JINA_BASE_URL value for tests).
func (s *Server) URL() string { return s.httpSrv.URL }

// Requests returns every request URI received, in order.
func (s *Server) Requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

// LastRequest returns the most recent request URI, or "".
func (s *Server) LastRequest() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requests) == 0 {
		return ""
	}
	return s.requests[len(s.requests)-1]
}
