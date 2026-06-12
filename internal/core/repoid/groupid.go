package repoid

import (
	"fmt"
	"strings"
)

// GroupSpec identifies an org/group on a forge host. Slug may span several
// segments (GitLab nested groups).
type GroupSpec struct {
	// Ref is the input as given, for error reporting.
	Ref  string
	Host string
	Slug string
}

// ParseGroup normalizes a group reference: host-qualified
// (`gitlab.com/acme/platform`) or a bare slug (defaults to github.com).
func ParseGroup(ref string) (GroupSpec, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return GroupSpec{}, fmt.Errorf("empty group reference")
	}
	segments := pathSegments(trimmed)
	spec := GroupSpec{Ref: ref, Host: DefaultHost}
	if len(segments) > 0 && looksLikeHost(segments[0]) {
		spec.Host = segments[0]
		segments = segments[1:]
	}
	if len(segments) == 0 {
		return GroupSpec{}, fmt.Errorf("invalid group reference %q: want <host>/<org-or-group>", ref)
	}
	spec.Slug = strings.Join(segments, "/")
	return spec, nil
}
