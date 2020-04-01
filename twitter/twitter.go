package twitter

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/sirupsen/logrus"
)

var (
	LastTweetID int64 = -1
)

type TwitterMedia struct {
	Name    string
	content []byte
}

func (m *TwitterMedia) Content() io.Reader {
	return bytes.NewReader(m.content)
}

type fetcher struct {
	applicationKey    string
	applicationSecret string
	accessToken       string
	accessTokenSecret string
	since             int64
	pollInterval      time.Duration

	httpClient *http.Client
	client     *twitter.Client
	user       *twitter.User
}

type Option func(f *fetcher)

func WithApplicationKey(v string) Option {
	return func(f *fetcher) {
		f.applicationKey = v
	}
}

func WithApplicationSecret(v string) Option {
	return func(f *fetcher) {
		f.applicationSecret = v
	}
}

func WithAccessToken(v string) Option {
	return func(f *fetcher) {
		f.accessToken = v
	}
}

func WithAccessTokenSecret(v string) Option {
	return func(f *fetcher) {
		f.accessTokenSecret = v
	}
}

func WithPollInterval(v time.Duration) Option {
	return func(f *fetcher) {
		f.pollInterval = v
	}
}

func WithSince(v int64) Option {
	return func(f *fetcher) {
		f.since = v
	}
}

func NewFetcher(opts ...Option) (*fetcher, error) {
	f := &fetcher{}
	for _, opt := range opts {
		opt(f)
	}

	// Twitter client
	config := oauth1.NewConfig(f.applicationKey, f.applicationSecret)
	token := oauth1.NewToken(f.accessToken, f.accessTokenSecret)
	f.httpClient = config.Client(oauth1.NoContext, token)
	f.client = twitter.NewClient(f.httpClient)

	// Get authenticated user
	user, _, err := f.client.Accounts.VerifyCredentials(&twitter.AccountVerifyParams{
	})
	if err != nil {
		return nil, err
	}
	f.user = user

	// Setting lower bound
	if f.since == LastTweetID {
		tweets, _, err := f.client.Timelines.UserTimeline(&twitter.UserTimelineParams{
			ScreenName: user.ScreenName,
			Count:      1,
		})
		if err != nil {
			return nil, err
		}

		if len(tweets) != 1 {
			return nil, fmt.Errorf("cannot fetch user last tweet")
		}
		f.since = tweets[0].ID
	}
	return f, nil
}

func (f fetcher) Content() (<-chan *TwitterMedia, <-chan error) {

	c := make(chan *TwitterMedia)
	errCh := make(chan error)

	go func() {
		lastID := f.since

		for {
			tweets, _, err := f.client.Timelines.UserTimeline(&twitter.UserTimelineParams{
				ScreenName:      f.user.ScreenName,
				IncludeRetweets: pbool(false),
				ExcludeReplies:  pbool(true),
				TweetMode:       "extended",
				SinceID:         lastID,
			})

			if err != nil {
				errCh <- err
				return
			}

			logrus.
				WithField("since", lastID).
				WithField("tweets", len(tweets)).
				Debug("poll")
			for _, t := range tweets {
				if t.ID > lastID {
					lastID = t.ID
				}
				if t.ExtendedEntities != nil {
					for _, media := range t.ExtendedEntities.Media {
						share, err := download(f.httpClient, media)
						if err != nil {
							errCh <- err
							return
						}
						c <- share
					}
				}
			}
			time.Sleep(f.pollInterval)
		}
	}()
	return c, errCh
}

func download(client *http.Client, media twitter.MediaEntity) (*TwitterMedia, error) {
	switch media.Type {
	case "photo":
		return downloadPhoto(client, media)
	case "video":
		return downloadVideo(client, media)
	}
	return nil, fmt.Errorf("unknown media type : %s", media.Type)
}
func downloadVideo(client *http.Client, media twitter.MediaEntity) (*TwitterMedia, error) {
	// find highest mp4 / bitrate
	var target twitter.VideoVariant
	for _, variant := range media.VideoInfo.Variants {
		if variant.ContentType == "video/mp4" && variant.Bitrate > target.Bitrate {
			target = variant
		}
	}

	// get content
	resp, err := client.Get(target.URL)
	if err != nil {
		return nil, err
	}

	// find file name
	rawurl, err := url.Parse(target.URL)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &TwitterMedia{
		Name:    filepath.Base(rawurl.Path),
		content: content,
	}, nil
}
func downloadPhoto(client *http.Client, media twitter.MediaEntity) (*TwitterMedia, error) {
	// get content
	resp, err := client.Get(media.MediaURL)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &TwitterMedia{
		Name:    filepath.Base(media.MediaURL),
		content: content,
	}, nil
}

func pbool(b bool) *bool {
	return &b
}
