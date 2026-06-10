package version

// Version is the current version of fetch-context.
// Override at build time with:
//
//	go build -ldflags "-X github.com/mattjmcnaughton/fetch-context/internal/version.Version=x.y.z" ./cmd/fetch-context
var Version = "dev"
