package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gphotosuploader/google-photos-api-client-go/v3/media_items"
	"github.com/maxcleme/twitter-media-backup/twitter"
	"golang.org/x/oauth2"

	"github.com/gphotosuploader/google-photos-api-client-go/v3"
)

type gPhotosExporter struct {
	applicationKey    string
	applicationSecret string
	albumName         string
	tokenPath         string
	redirectURL       string
	callbackPort      int

	albumID string
	client  *gphotos.Client
}

type GPhotosOption func(f *gPhotosExporter)

func WithApplicationKey(v string) GPhotosOption {
	return func(e *gPhotosExporter) {
		e.applicationKey = v
	}
}

func WithApplicationSecret(v string) GPhotosOption {
	return func(e *gPhotosExporter) {
		e.applicationSecret = v
	}
}

func WithTokenPath(v string) GPhotosOption {
	return func(e *gPhotosExporter) {
		e.tokenPath = v
	}
}

func WithAlbumName(v string) GPhotosOption {
	return func(e *gPhotosExporter) {
		e.albumName = v
	}
}

func WithRedirectURL(v string) GPhotosOption {
	return func(e *gPhotosExporter) {
		e.redirectURL = v
	}
}

func WithCallbackPort(v int) GPhotosOption {
	return func(e *gPhotosExporter) {
		e.callbackPort = v
	}
}

func NewGPhotosExporter(ctx context.Context, opts ...GPhotosOption) (*gPhotosExporter, error) {
	e := &gPhotosExporter{}
	for _, opt := range opts {
		opt(e)
	}

	oauthClient, err := getClient(ctx, e, &oauth2.Config{
		ClientID:     e.applicationKey,
		ClientSecret: e.applicationSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/photoslibrary"},
		RedirectURL:  e.redirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	})
	if err != nil {
		return nil, err
	}

	// httpClient is an authenticated http.Client. See Authentication below.
	client, err := gphotos.NewClient(oauthClient)
	if err != nil {
		return nil, err
	}
	e.client = client

	// get or create a Photos Album with the specified name.
	album, err := client.Albums.GetByTitle(ctx, e.albumName)
	if err != nil {
		return nil, err
	}
	e.albumID = album.ID
	return e, nil
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Persist token to local file
func saveToken(path string, token *oauth2.Token) error {
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

func getClient(ctx context.Context, e *gPhotosExporter, config *oauth2.Config) (*http.Client, error) {
	tok, err := tokenFromFile(e.tokenPath)
	if err == nil {
		return config.Client(ctx, tok), nil
	}
	tok, err = getTokenFromWeb(ctx, e, config)
	if err != nil {
		return nil, err
	}
	if err := saveToken(e.tokenPath, tok); err != nil {
		return nil, err
	}
	return config.Client(ctx, tok), nil
}

func getTokenFromWeb(ctx context.Context, e *gPhotosExporter, conf *oauth2.Config) (*oauth2.Token, error) {
	authurl := conf.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for the auth dialog: %v\n", authurl)
	codeCh := make(chan string)
	errCh := make(chan error)
	rawurl, err := url.Parse(e.redirectURL)
	if err != nil {
		return nil, err
	}
	r := http.NewServeMux()
	r.HandleFunc(rawurl.Path, func(rw http.ResponseWriter, r *http.Request) {
		codeCh <- r.URL.Query().Get("code")
	})
	s := http.Server{
		Addr:    fmt.Sprintf(":%d", e.callbackPort),
		Handler: r,
	}
	// create goroutine responsible for handling oauth callback
	go func() {
		if err := s.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()
	select {
	case code := <-codeCh:
		// receive code from http server
		if err := s.Shutdown(ctx); err != nil {
			return nil, err
		}

		tok, err := conf.Exchange(ctx, code)
		if err != nil {
			return nil, err
		}
		return tok, nil
	case err := <-errCh:
		// receive error from http server
		return nil, err
	}
}

func (e *gPhotosExporter) Export(ctx context.Context, media *twitter.TwitterMedia) error {
	dir, err := os.MkdirTemp("", "twitter-media-backup-gphotos")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, media.Name)
	if err := os.WriteFile(path, media.Payload, 0o644); err != nil {
		return err
	}
	token, err := e.client.Uploader.UploadFile(ctx, path)
	if err != nil {
		return err
	}
	_, err = e.client.MediaItems.CreateToAlbum(ctx, e.albumID, media_items.SimpleMediaItem{
		Filename:    media.Name,
		UploadToken: token,
	})
	return err
}

func (e *gPhotosExporter) Type() string {
	return "gphotos"
}
