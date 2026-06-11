// Package fakes provides in-memory fakes for every port, used by the unit
// tests in internal/core/. Fakes record calls and can be scripted to fail.
package fakes

import (
	"context"
	"io"
	"os"

	"github.com/spf13/afero"

	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// CloneCall records one GitRepo.Clone invocation.
type CloneCall struct {
	URL  string
	Dest string
}

// FakeGitRepo implements ports.GitRepo.
type FakeGitRepo struct {
	// Clones and Refreshes record successful-or-not invocations in order.
	Clones    []CloneCall
	Refreshes []string
	// ManagedDirs answers IsManagedClone per dest.
	ManagedDirs map[string]bool
	// CloneErrs scripts Clone failures by clone URL.
	CloneErrs map[string]error
	// RefreshErrs scripts Refresh failures by dest.
	RefreshErrs map[string]error
	// FS, when set, gets the dest directory created on successful Clone so
	// filesystem-observing assertions see the clone appear.
	FS *FakeFileStore
}

func NewGitRepo() *FakeGitRepo {
	return &FakeGitRepo{
		ManagedDirs: make(map[string]bool),
		CloneErrs:   make(map[string]error),
		RefreshErrs: make(map[string]error),
	}
}

func (g *FakeGitRepo) Clone(_ context.Context, cloneURL, dest string) error {
	g.Clones = append(g.Clones, CloneCall{URL: cloneURL, Dest: dest})
	if err := g.CloneErrs[cloneURL]; err != nil {
		return err
	}
	if g.FS != nil {
		if err := g.FS.MkdirAll(dest); err != nil {
			return err
		}
	}
	// A successful clone is, by definition, a managed clone afterwards.
	g.ManagedDirs[dest] = true
	return nil
}

func (g *FakeGitRepo) Refresh(_ context.Context, dest string) error {
	g.Refreshes = append(g.Refreshes, dest)
	return g.RefreshErrs[dest]
}

func (g *FakeGitRepo) IsManagedClone(_ context.Context, dest string) (bool, error) {
	return g.ManagedDirs[dest], nil
}

// FakeHostRepoLocator implements ports.HostRepoLocator with a constant.
type FakeHostRepoLocator struct {
	Root string
	Err  error
}

func (l *FakeHostRepoLocator) RepoRoot(context.Context) (string, error) {
	return l.Root, l.Err
}

// FakeFileStore implements ports.FileStore over an in-memory afero fs.
type FakeFileStore struct {
	Fs afero.Fs
}

func NewFileStore() *FakeFileStore {
	return &FakeFileStore{Fs: afero.NewMemMapFs()}
}

func (f *FakeFileStore) MkdirAll(path string) error {
	return f.Fs.MkdirAll(path, 0o755)
}

func (f *FakeFileStore) WriteFile(path string, data []byte) error {
	return afero.WriteFile(f.Fs, path, data, 0o644)
}

func (f *FakeFileStore) OpenForRead(path string) (io.ReadCloser, error) {
	return f.Fs.Open(path)
}

func (f *FakeFileStore) Remove(path string) error {
	return f.Fs.RemoveAll(path)
}

func (f *FakeFileStore) Exists(path string) (bool, error) {
	return afero.Exists(f.Fs, path)
}

func (f *FakeFileStore) Walk(root string, fn ports.WalkFunc) error {
	return afero.Walk(f.Fs, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return fn(path, info.IsDir())
	})
}

// ReadString is a test convenience: the file's content, or "" if absent.
func (f *FakeFileStore) ReadString(path string) string {
	b, err := afero.ReadFile(f.Fs, path)
	if err != nil {
		return ""
	}
	return string(b)
}

// Interface conformance.
var (
	_ ports.GitRepo         = (*FakeGitRepo)(nil)
	_ ports.HostRepoLocator = (*FakeHostRepoLocator)(nil)
	_ ports.FileStore       = (*FakeFileStore)(nil)
)
