// Package envx gives adapters typed, fakeable access to environment
// variables. It is an adapter-layer helper, not a port: the core has no
// concept of the environment.
package envx

import "os"

// Env reads environment variables.
type Env interface {
	Get(key string) (string, bool)
}

// OsEnv is the real environment.
type OsEnv struct{}

func (OsEnv) Get(key string) (string, bool) {
	return os.LookupEnv(key)
}

// Fake is a scriptable environment for tests.
type Fake map[string]string

func (f Fake) Get(key string) (string, bool) {
	v, ok := f[key]
	return v, ok
}
