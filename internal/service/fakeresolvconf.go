package service

import (
	"dnsmanager/internal/resolvconf"
	"io"
	"log/slog"
)

type fakeRC struct {
	file     *resolvconf.File
	writes   int
	writeErr error
}

func newFake(content string) *fakeRC {
	return &fakeRC{file: resolvconf.Parse([]byte(content))}
}

func (f *fakeRC) Read() (*resolvconf.File, error) {
	return resolvconf.Parse(f.file.Render()), nil
}

func (f *fakeRC) Write(file *resolvconf.File) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.file = file
	f.writes++
	return nil
}

func newSvc(rc ResolvConf) *Service {
	return New(rc, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
