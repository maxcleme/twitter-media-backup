package exporter

import "github.com/maxcleme/twitter-media-backup/twitter"

type Exporter interface {
	Type() string
	Export(media *twitter.TwitterMedia) error
}
