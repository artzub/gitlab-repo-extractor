package main

import (
	"strconv"
	"testing"
	"time"
)

type TestEnvLoader struct {
	envs map[string]string
}

func (t *TestEnvLoader) Load() {
	// No-op for test loader, as we are providing the environment variables directly.
}

func (t *TestEnvLoader) Get(key string, defaultValue ...string) string {
	if value, ok := t.envs[key]; ok && value != "" {
		return value
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func (t *TestEnvLoader) GetInt(key string, defaultValue int) int {
	value := t.Get(key)
	return getInt(value, defaultValue)
}

func NewTestEnvLoader(envs map[string]string) *TestEnvLoader {
	return &TestEnvLoader{
		envs: envs,
	}
}

func TestNewConfig(t *testing.T) {
	expectConfig := &Config{
		GitLabURL:   "https://gitlab.example.com",
		AccessToken: "example_token",
		OutputDir:   "/tmp/gitlab-repos",
		GroupID:     "example_group",
		RetryDelay:  3 * time.Second,
		MaxWorkers:  10,
		MaxRetries:  5,
		UseSSH:      true,
	}
	expectations := map[string]string{
		"RE_GITLAB_URL":          expectConfig.GitLabURL,
		"RE_GITLAB_TOKEN":        expectConfig.AccessToken,
		"RE_OUTPUT_DIR":          expectConfig.OutputDir,
		"RE_GROUP_ID":            expectConfig.GroupID,
		"RE_RETRY_DELAY_SECONDS": strconv.Itoa(int(expectConfig.RetryDelay.Seconds())),
		"RE_MAX_WORKERS":         strconv.Itoa(expectConfig.MaxWorkers),
		"RE_MAX_RETRIES":         strconv.Itoa(expectConfig.MaxRetries),
		"RE_USE_SSH":             strconv.FormatBool(expectConfig.UseSSH),
	}

	loader := NewTestEnvLoader(expectations)

	config := NewConfig(loader)

	if config.GitLabURL != expectConfig.GitLabURL {
		t.Errorf("Expected GitLabURL %s, got %s", expectConfig.GitLabURL, config.GitLabURL)
	}
	if config.AccessToken != expectConfig.AccessToken {
		t.Errorf("Expected AccessToken %s, got %s", expectConfig.AccessToken, config.AccessToken)
	}
	if config.OutputDir != expectConfig.OutputDir {
		t.Errorf("Expected OutputDir %s, got %s", expectConfig.OutputDir, config.OutputDir)
	}
	if config.GroupID != expectConfig.GroupID {
		t.Errorf("Expected GroupID %s, got %s", expectConfig.GroupID, config.GroupID)
	}
	if config.RetryDelay != expectConfig.RetryDelay {
		t.Errorf("Expected RetryDelay %s, got %s", expectConfig.RetryDelay, config.RetryDelay)
	}
	if config.MaxWorkers != expectConfig.MaxWorkers {
		t.Errorf("Expected MaxWorkers %d, got %d", expectConfig.MaxWorkers, config.MaxWorkers)
	}
	if config.MaxRetries != expectConfig.MaxRetries {
		t.Errorf("Expected MaxRetries %d, got %d", expectConfig.MaxRetries, config.MaxRetries)
	}
	if config.UseSSH != expectConfig.UseSSH {
		t.Errorf("Expected UseSSH %t, got %t", expectConfig.UseSSH, config.UseSSH)
	}
}

func TestGetConfigSingleton(t *testing.T) {
	expectations := map[string]string{
		"RE_GITLAB_URL": "https://gitlab.example.com",
	}
	loader := NewTestEnvLoader(expectations)

	config1 := GetConfig(loader)
	config2 := GetConfig(loader)

	if config1 != config2 {
		t.Error("GetConfig should return the same instance on subsequent calls")
	}

	if config1.GitLabURL != expectations["RE_GITLAB_URL"] {
		t.Errorf("Expected GitLabURL %s, got %s", expectations["RE_GITLAB_URL"], config1.GitLabURL)
	}
}
