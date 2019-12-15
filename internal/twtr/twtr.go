package twtr

import (
	"github.com/ChimeraCoder/anaconda"
	"github.com/wcsiu/tg-demuxer/internal/config"
)

var (
	client *anaconda.TwitterApi
)

// Load initialize the twitter api client.
func Load() {
	client = anaconda.NewTwitterApiWithCredentials(config.C.TWTR.AccessToken, config.C.TWTR.AccessTokenSecret, config.C.TWTR.ConsumerKey, config.C.TWTR.ConsumerSecret)
}
