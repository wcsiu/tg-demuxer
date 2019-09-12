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
	ChatList []int64 `yaml:"chatlist"`
}

// TG telegram config struct.
type TG struct {
	APIID   string `yaml:"apiid"`
	APIHash string `yaml:"apihash"`
}

// DB database config stuct.
type DB struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
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
