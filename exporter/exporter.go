package exporter

import (
	"context"

	"github.com/maxcleme/twitter-media-backup/twitter"
)

type Exporter interface {
	Type() string
	Export(ctx context.Context, media *twitter.TwitterMedia) error
}
