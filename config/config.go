package config

import (
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Config holds the configuration for the GitLab repository downloader.
type Config struct {
	groupIDs     []string
	skipGroupIDs []string
	gitLabURL    string
	accessToken  string
	outputDir    string
	retryDelay   time.Duration
	maxWorkers   int
	maxRetries   int
	useSSH       bool
}

func extractGroupIDs(groupIDs string) []string {
	cleaned := strings.FieldsFunc(groupIDs, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	cleaned = slices.DeleteFunc(cleaned, func(s string) bool {
		return s == ""
	})
	return slices.Compact(cleaned)
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
		gitLabURL:    loader.Get(GitlabURLKey, DefaultGitlabURL),
		accessToken:  loader.Get(GitlabTokenKey),
		outputDir:    loader.Get(OutputDirKey, DefaultOutputDir),
		useSSH:       loader.Get(UseSSHKey, DefaultUseSSH) == "true",
		groupIDs:     extractGroupIDs(loader.Get(GroupIDsKey)),
		skipGroupIDs: extractGroupIDs(loader.Get(SkipGroupIDsKey)),
		maxWorkers:   loader.GetInt(MaxWorkersKey, DefaultMaxWorkers),
		maxRetries:   loader.GetInt(MaxRetriesKey, DefaultMaxRetries),
		retryDelay:   time.Duration(loader.GetInt(RetryDelayKey, DefaultRetryDelay)) * time.Second,
	}
}

func (c *Config) GetGitLabURL() string {
	return c.gitLabURL
}

func (c *Config) GetAccessToken() string {
	return c.accessToken
}

func (c *Config) GetOutputDir() string {
	return c.outputDir
}

func (c *Config) GetGroupIDs() []string {
	return c.groupIDs
}

func (c *Config) GetSkipGroupIDs() []string {
	return c.skipGroupIDs
}

func (c *Config) GetRetryDelay() time.Duration {
	return c.retryDelay
}

func (c *Config) GetMaxWorkers() int {
	return c.maxWorkers
}

func (c *Config) GetMaxRetries() int {
	return c.maxRetries
}

func (c *Config) GetUseSSH() bool {
	return c.useSSH
}

// singleton instance of Config
var (
	configInstance *Config
	configOnce     sync.Once
)

// GetConfig returns the singleton instance of Config.
func GetConfig(loaders ...EnvLoader) *Config {
	configOnce.Do(func() {
		configInstance = NewConfig(loaders...)
	})
	return configInstance
}
