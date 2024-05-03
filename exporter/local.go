package exporter

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/maxcleme/twitter-media-backup/twitter"
)

type localExporter struct {
	rootPath string
}

type LocalOption func(f *localExporter)

func WithRootPath(v string) LocalOption {
	return func(e *localExporter) {
		e.rootPath = v
	}
}

func NewLocalExporter(opts ...LocalOption) (*localExporter, error) {
	e := &localExporter{}
	for _, opt := range opts {
		opt(e)
	}
	return e, nil
}

func (e *localExporter) Export(media *twitter.TwitterMedia) error {
	// Create the file
	path := filepath.Join(e.rootPath, media.Name)
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, bytes.NewReader(media.Payload))
	return err
}

func (e *localExporter) Type() string {
	return "local"
}
