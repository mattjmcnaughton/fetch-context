// Package repoid normalizes repo references (R6 in docs/acceptance.md):
// shorthand `foo/bar`, host-qualified `github.com/foo/bar`, and full clone
// URLs all collapse to one (host, owner, repo) identity.
package repoid

import (
	"fmt"
	"net/url"
	"strings"
)

// DefaultHost is assumed for bare `owner/repo` shorthand.
const DefaultHost = "github.com"

// Spec is a normalized repo identity. Owner may contain slashes (GitLab
// subgroup paths). Scheme is set when the input was a full URL or an
// scp-like SSH ref; User holds the SSH user (e.g. "git").
type Spec struct {
	// Ref is the input as given, for error reporting.
	Ref    string
	Scheme string
	User   string
	Host   string
	Owner  string
	Repo   string
	// scpLike records that the SSH input used scp syntax
	// (git@host:owner/repo) so CloneURL round-trips that form rather than
	// the ssh:// URL form.
	scpLike bool
}

// Key identifies the destination; equivalent input forms share a Key.
func (s Spec) Key() string {
	return s.Host + "/" + s.Owner + "/" + s.Repo
}

// CloneURL is the canonical URL to clone from: the original scheme (https
// when the input was not a URL) plus the normalized path with `.git`. SSH
// refs preserve their user and original (scp-like vs ssh://) form.
func (s Spec) CloneURL() string {
	switch s.Scheme {
	case "ssh":
		user := ""
		if s.User != "" {
			user = s.User + "@"
		}
		if s.scpLike {
			return user + s.Host + ":" + s.Owner + "/" + s.Repo + ".git"
		}
		return "ssh://" + user + s.Key() + ".git"
	case "http":
		return "http://" + s.Key() + ".git"
	default:
		return "https://" + s.Key() + ".git"
	}
}

// Parse normalizes one repo reference.
func Parse(ref string) (Spec, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return Spec{}, fmt.Errorf("empty repo reference")
	}

	spec := Spec{Ref: ref}
	if scheme, _, ok := strings.Cut(trimmed, "://"); ok {
		switch scheme {
		case "http", "https", "ssh":
		default:
			return Spec{}, fmt.Errorf("unsupported scheme %q in %q (only http, https, and ssh clone URLs are accepted)", scheme, ref)
		}
		u, err := url.Parse(trimmed)
		if err != nil {
			return Spec{}, fmt.Errorf("invalid URL %q: %w", ref, err)
		}
		if u.Host == "" {
			return Spec{}, fmt.Errorf("invalid URL %q: missing host", ref)
		}
		spec.Scheme = scheme
		spec.Host = u.Host
		if u.User != nil {
			spec.User = u.User.Username()
		}
		if err := spec.fillPath(strings.TrimPrefix(u.Path, "/"), 2); err != nil {
			return Spec{}, err
		}
		return spec, nil
	}

	// scp-like SSH syntax: [user@]host:owner/repo(.git). It is distinguished
	// from a host-qualified shorthand by the user@ prefix and a colon that
	// precedes the first slash (so host:port/owner/repo shorthand is unaffected).
	if user, rest, ok := strings.Cut(trimmed, "@"); ok && user != "" && !strings.Contains(user, "/") {
		if host, path, ok := strings.Cut(rest, ":"); ok && host != "" && !strings.Contains(host, "/") {
			spec.Scheme = "ssh"
			spec.scpLike = true
			spec.User = user
			spec.Host = host
			if err := spec.fillPath(path, 2); err != nil {
				return Spec{}, err
			}
			return spec, nil
		}
	}

	segments := pathSegments(trimmed)
	if len(segments) >= 1 && looksLikeHost(segments[0]) {
		spec.Host = segments[0]
		if err := spec.fillSegments(segments[1:], 2); err != nil {
			return Spec{}, err
		}
		return spec, nil
	}
	spec.Host = DefaultHost
	if err := spec.fillSegments(segments, 2); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

// fillPath splits a URL path and assigns owner/repo.
func (s *Spec) fillPath(path string, minSegments int) error {
	return s.fillSegments(pathSegments(path), minSegments)
}

// fillSegments assigns owner (all but last, possibly multi-segment) and repo
// (last, with .git stripped).
func (s *Spec) fillSegments(segments []string, minSegments int) error {
	if len(segments) < minSegments {
		return fmt.Errorf("invalid repo reference %q: want <owner>/<repo> (optionally host-qualified or a full clone URL)", s.Ref)
	}
	last := strings.TrimSuffix(segments[len(segments)-1], ".git")
	if last == "" {
		return fmt.Errorf("invalid repo reference %q: empty repo name", s.Ref)
	}
	s.Owner = strings.Join(segments[:len(segments)-1], "/")
	s.Repo = last
	return nil
}

// pathSegments splits on "/", dropping empty segments (handles trailing
// slashes and accidental doubles).
func pathSegments(p string) []string {
	var out []string
	for _, seg := range strings.Split(p, "/") {
		if seg != "" {
			out = append(out, seg)
		}
	}
	return out
}

// looksLikeHost reports whether the first path segment is a hostname rather
// than an owner: it contains a dot (domain) or a colon (host:port).
func looksLikeHost(seg string) bool {
	return strings.ContainsAny(seg, ".:")
}
