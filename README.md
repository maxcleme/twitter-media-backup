# twitter-media-backup

⚠️ Since X/Twitter [doesn't allow your own tweets retrieval with a Free plan anymore](https://developer.twitter.com/en/docs/twitter-api/getting-started/about-twitter-api), this project is relying on https://github.com/n0madic/twitter-scraper to get content, use at your own risk. ⚠️

## Overview

Idea behind this project is to backup all pictures/videos tweets of one Twitter account somewhere else by using **exporters**. It runs infinitly, waiting for new tweets to backup.

#### Exporters

| Name | Description |
|--|--|
| `local` | Backup media to local dedicated folder |
| `gphotos` | Backup media to Google Photos' dedicated album |

## Configuration

Put the following file into your home folder : 

**.twitter-media-backup.yaml**
```yaml
log_level: info

# Account credentials
twitter_username: XXXXXXXXXXXXXXXXXX
twitter_password: XXXXXXXXXXXXXXXXXX
# Poll new tweets interval
twitter_poll_interval: 5s

# Local exporter configuration, straightforward.
local: true # Enable exporter
local_root_path: /tmp/twitter-media-backup/local

# Google Photos exporter configuration
# Google APIs use oauth2 under the hood
# As for Twitter, you will have to create a Google application
# Google Photos API also needs to be enabled for your application
gphotos: true # Enable exporter
# This define parameters for oauth2 Authorization Code flow
# Program is going to ask you to visit url at startup, 
# once done your token will be saved as json 
# in order to prevent this behaviors the next time.
gphotos_oauth2_token_path: /tmp/twitter-media-backup/gphotos/token.json
gphotos_oauth2_redirect_url: http://localhost:8080/callback
gphotos_oauth2_port: 8080
gphotos_oauth2_application_key: XXXXXXXXXXXXXXXXXX
gphotos_oauth2_application_secret: XXXXXXXXXXXXXXXXXX
# Google Photos album name destination, will be created if not exists.
gphotos_album: MyAwesomeAlbum

```

## Why ?

I first started this project because I wanted to backup my Nintendo Switch pictures/videos without having to remove the SD card. Nintendo Switch supports sharing only to Twitter and/or Facebook, I choose Twitter as a proxy.

## Q&A : 

> It never stops ?!

*This is the requested behaviors, let this program running somewhere and forget it, all your tweets' media will be backup as soon as new tweet are posted.*

> Why not use Twitter Streaming API ?

*Since it needs to supports Protected Account, we can't use Streaming API because they exlude all protected tweets in stream for obvious reason.*

