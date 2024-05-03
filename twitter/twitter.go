package twitter

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/n0madic/twitter-scraper"
)

type TwitterMedia struct {
	Name    string
	Payload []byte
}

type fetcher struct {
	username     string
	password     string
	pollInterval time.Duration

	scrapper *twitterscraper.Scraper
}

type Option func(f *fetcher)

func WithUsername(v string) Option {
	return func(f *fetcher) {
		f.username = v
	}
}

func WithPassword(v string) Option {
	return func(f *fetcher) {
		f.password = v
	}
}

func WithPollInterval(v time.Duration) Option {
	return func(f *fetcher) {
		f.pollInterval = v
	}
}

func NewFetcher(opts ...Option) (*fetcher, error) {
	f := &fetcher{}
	for _, opt := range opts {
		opt(f)
	}

	scraper := twitterscraper.New()
	if err := scraper.Login(f.username, f.password); err != nil {
		return nil, err
	}
	f.scrapper = scraper
	return f, nil
}

func (f fetcher) Fetch() (<-chan *TwitterMedia, <-chan error) {
	c := make(chan *TwitterMedia)
	errCh := make(chan error)
	go func() {
		var last int64
		ticker := time.NewTicker(f.pollInterval)
		defer ticker.Stop()
		for range ticker.C {
			tweets, _, err := f.scrapper.FetchTweets(f.username, 5, "")
			if err != nil {
				errCh <- err
				return
			}
			for _, tweet := range tweets {
				if tweet.Timestamp <= last {
					break
				}
				last = tweet.Timestamp
				if len(tweet.Videos) > 0 {
					for _, video := range tweet.Videos {
						payload, err := download(video.URL)
						if err != nil {
							errCh <- err
							return
						}
						c <- &TwitterMedia{
							Name:    video.ID + ".mp4",
							Payload: payload,
						}
					}
				}
				if len(tweet.Photos) > 0 {
					for _, photo := range tweet.Photos {
						u, err := url.Parse(photo.URL)
						if err != nil {
							errCh <- err
							return
						}
						params := u.Query()
						params.Set("format", "jpg")
						params.Set("name", "large")
						u.RawQuery = params.Encode()
						payload, err := download(u.String())
						if err != nil {
							errCh <- err
							return
						}
						c <- &TwitterMedia{
							Name:    photo.ID + ".jpg",
							Payload: payload,
						}
					}
				}
			}
		}
	}()
	return c, errCh
}

func download(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "TwitterAndroid/99")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
