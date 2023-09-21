package config

import (
	"encoding/json"
	"net/url"
	"os"
	"strings"
)

type Release struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	Owner           string
	Repo            string
	Platform        string `json:"platform"`
	TasReleaseName  string `json:"tas_release_name"`
	TaswReleaseName string `json:"tasw_release_name"`
	IstReleaseName  string `json:"ist_release_name"`
}

type Config struct {
	Releases []Release `json:"releases"`
}

func LoadConfig(filePath string) (Config, error) {
	var cfg Config
	configFile, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, err
	}
	err = json.Unmarshal([]byte(configFile), &cfg)
	if err != nil {
		return Config{}, err
	}
	for i, release := range cfg.Releases {
		url, err := url.Parse(release.URL)
		if err != nil {
			return Config{}, err
		}
		parts := strings.Split(strings.TrimLeft(url.Path, "/"), "/")
		cfg.Releases[i].Owner = parts[0]
		cfg.Releases[i].Repo = parts[1]
	}
	return cfg, nil
}
