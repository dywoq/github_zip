package config

import (
	"os"
)

type Config struct {
	Token   string
	User    string
	BaseDir string
}

func Load() (*Config, error) {
	token := os.Getenv("GITHUB_TOKEN")
	user := os.Getenv("GITHUB_USER")
	if token == "" || user == "" {
		return nil, nil
	}
	return &Config{
		Token:   token,
		User:    user,
		BaseDir: "./github_zips",
	}, nil
}
