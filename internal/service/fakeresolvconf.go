package service

import (
	"bytes"
	"io"
	"log/slog"
	"slices"
	"sync"

	"dnsmanager/internal/model"
	"dnsmanager/internal/resolvconf"
	"dnsmanager/internal/storage"
)

type fakeRC struct {
	mu       sync.Mutex
	file     *resolvconf.File
	writes   int
	readErr  error
	writeErr error
}

func newFakeRC(content string) *fakeRC {
	return &fakeRC{file: resolvconf.Parse([]byte(content))}
}

func (f *fakeRC) Read() (*resolvconf.File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readErr != nil {
		return nil, f.readErr
	}
	return resolvconf.Parse(f.file.Render()), nil
}

func (f *fakeRC) Write(file *resolvconf.File) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeErr != nil {
		return f.writeErr
	}
	f.file = resolvconf.Parse(file.Render())
	f.writes++
	return nil
}

func (f *fakeRC) setContent(s string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.file = resolvconf.Parse([]byte(s))
}

func (f *fakeRC) content() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return string(f.file.Render())
}

func (f *fakeRC) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writes
}

type fakeBackup struct {
	mu      sync.Mutex
	saved   []string
	rev     uint64
	origin  string
	saves   int
	saveErr error
	loadErr error
}

func (b *fakeBackup) Save(st model.State) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.saveErr != nil {
		return b.saveErr
	}
	b.saved = slices.Clone(st.Servers)
	if b.saved == nil {
		b.saved = []string{}
	}
	b.rev, b.origin = st.Revision, st.Origin
	b.saves++
	return nil
}

func (b *fakeBackup) Load() (model.State, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.loadErr != nil {
		return model.State{}, b.loadErr
	}
	if b.saved == nil {
		return model.State{}, storage.ErrNotFound
	}
	return model.State{Revision: b.rev, Origin: b.origin, Servers: slices.Clone(b.saved)}, nil
}

func (b *fakeBackup) saveCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.saves
}

func (b *fakeBackup) last() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.saved)
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
