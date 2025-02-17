package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Telegram Telegram `yaml:"telegram"` // telegram chat id
	Kube     Kube     `yaml:"kube"`     // kube
}

type Telegram struct {
	Token    string         `yaml:"token"` // telegram bot token
	Chat     int            `yaml:"chat"`  // cat ids
	Debug    bool           `yaml:"debug"` // debug
	DelayMsg time.Duration  `yaml:"delay_msg"`
	Topics   map[string]int `yaml:"topics"`
}

type Kube struct {
	Target     []Target `yaml:"target"`    // pod and containers name
	KubeConfig string   `yaml:"config"`    // path to kube config
	Namespace  string   `yaml:"namespace"` // kube namespace
	Filter     string   `yaml:"filter"`
}

type Target struct {
	Pod       string   `yaml:"pod"`       // pod name
	Container []string `yaml:"container"` // container names
}

func NewConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	cfg := Config{}

	if decodeErr := yaml.NewDecoder(file).Decode(&cfg); decodeErr != nil {
		return Config{}, decodeErr
	}

	return cfg, nil

}
