package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// C config values
var C Config

// Config configuration struct.
type Config struct {
	TG       TG      `yaml:"tg"`
	DB       DB      `yaml:"db"`
	TWTR     TWTR    `yaml:"twtr"`
	ChatList []int64 `yaml:"chat_list"`
}

// TG telegram config struct.
type TG struct {
	APIID   string `yaml:"api_id"`
	APIHash string `yaml:"api_hash"`
	Backup  string `yaml:"backup"`
}

// DB database config struct.
type DB struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// TWTR twitter api config struct.
type TWTR struct {
	ConsumerKey       string `yaml:"consumer_key"`
	ConsumerSecret    string `yaml:"consumer_secret"`
	AccessToken       string `yaml:"access_token"`
	AccessTokenSecret string `yaml:"access_token_secret"`
}

// Load load config from path.
func Load(path string) error {
	var content, err = ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(content, &C); err != nil {
		return err
	}
	return nil
}
