package main

import (
	"time"
)

// Config holds the configuration for the GitLab repository downloader.
type Config struct {
	GitLabURL   string
	AccessToken string
	OutputDir   string
	GroupID     string
	RetryDelay  time.Duration
	MaxWorkers  int
	MaxRetries  int
	UseSSH      bool
}

func NewConfig(loaders ...EnvLoader) *Config {
	var loader EnvLoader

	if len(loaders) > 0 {
		loader = loaders[0]
	}

	if loader == nil {
		loader = DefaultEnvLoader
	}

	loader.Load()

	return &Config{
		GitLabURL:   loader.Get("RE_GITLAB_URL", "https://gitlab.com"),
		AccessToken: loader.Get("RE_GITLAB_TOKEN"),
		OutputDir:   loader.Get("RE_OUTPUT_DIR", "./gitlab-repos"),
		UseSSH:      loader.Get("RE_USE_SSH", "false") == "true",
		GroupID:     loader.Get("RE_GROUP_ID"),
		MaxWorkers:  loader.GetInt("RE_MAX_WORKERS", 5),
		MaxRetries:  loader.GetInt("RE_MAX_RETRIES", 3),
		RetryDelay:  time.Duration(loader.GetInt("RE_RETRY_DELAY_SECONDS", 2)) * time.Second,
	}
}

// singleton instance of Config
var configInstance *Config

// GetConfig returns the singleton instance of Config.
func GetConfig(loaders ...EnvLoader) *Config {
	if configInstance == nil {
		configInstance = NewConfig(loaders...)
	}
	return configInstance
}
