package config

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	expectConfig := &Config{
		gitLabURL:    "https://gitlab.example.com",
		accessToken:  "example_token",
		outputDir:    "/tmp/gitlab-repos",
		groupIDs:     []string{"example_group", "example_group5"},
		skipGroupIDs: []string{"example_group1", "example_group2"},
		retryDelay:   3 * time.Second,
		maxWorkers:   10,
		maxRetries:   5,
		useSSH:       true,
	}
	expectations := map[string]string{
		GitlabURLKey:    expectConfig.gitLabURL,
		GitlabTokenKey:  expectConfig.accessToken,
		OutputDirKey:    expectConfig.outputDir,
		GroupIDsKey:     strings.Join(expectConfig.groupIDs, " "),
		SkipGroupIDsKey: strings.Join(expectConfig.skipGroupIDs, ","),
		RetryDelayKey:   strconv.Itoa(int(expectConfig.retryDelay.Seconds())),
		MaxWorkersKey:   strconv.Itoa(expectConfig.maxWorkers),
		MaxRetriesKey:   strconv.Itoa(expectConfig.maxRetries),
		UseSSHKey:       strconv.FormatBool(expectConfig.useSSH),
	}

	loader := NewMemoryEnvLoader(expectations)

	config := NewConfig(loader)

	// Verify the configuration values
	if config.gitLabURL != expectConfig.gitLabURL {
		t.Errorf("Expected gitLabURL %s, got %s", expectConfig.gitLabURL, config.gitLabURL)
	}
	if config.accessToken != expectConfig.accessToken {
		t.Errorf("Expected accessToken %s, got %s", expectConfig.accessToken, config.accessToken)
	}
	if config.outputDir != expectConfig.outputDir {
		t.Errorf("Expected outputDir %s, got %s", expectConfig.outputDir, config.outputDir)
	}
	if !slices.Equal(config.groupIDs, expectConfig.groupIDs) {
		t.Errorf("Expected GroupIDs %s, got %s", expectConfig.groupIDs, config.groupIDs)
	}
	if !slices.Equal(config.skipGroupIDs, expectConfig.skipGroupIDs) {
		t.Errorf("Expected skipGroupIDs %s, got %s", expectConfig.skipGroupIDs, config.skipGroupIDs)
	}
	if config.retryDelay != expectConfig.retryDelay {
		t.Errorf("Expected retryDelay %s, got %s", expectConfig.retryDelay, config.retryDelay)
	}
	if config.maxWorkers != expectConfig.maxWorkers {
		t.Errorf("Expected maxWorkers %d, got %d", expectConfig.maxWorkers, config.maxWorkers)
	}
	if config.maxRetries != expectConfig.maxRetries {
		t.Errorf("Expected maxRetries %d, got %d", expectConfig.maxRetries, config.maxRetries)
	}
	if config.useSSH != expectConfig.useSSH {
		t.Errorf("Expected useSSH %t, got %t", expectConfig.useSSH, config.useSSH)
	}

	// Verify getters
	if config.GetGitLabURL() != config.gitLabURL {
		t.Errorf("Expected gitLabURL %s, got %s", config.gitLabURL, config.GetGitLabURL())
	}
	if config.GetAccessToken() != config.accessToken {
		t.Errorf("Expected accessToken %s, got %s", config.accessToken, config.GetAccessToken())
	}
	if config.GetOutputDir() != config.outputDir {
		t.Errorf("Expected outputDir %s, got %s", config.outputDir, config.GetOutputDir())
	}
	if !slices.Equal(config.GetGroupIDs(), config.groupIDs) {
		t.Errorf("Expected GroupIDs %s, got %s", config.groupIDs, config.GetGroupIDs())
	}
	if !slices.Equal(config.GetSkipGroupIDs(), config.skipGroupIDs) {
		t.Errorf("Expected skipGroupIDs %s, got %s", config.skipGroupIDs, config.GetSkipGroupIDs())
	}
	if config.GetRetryDelay() != config.retryDelay {
		t.Errorf("Expected retryDelay %s, got %s", config.retryDelay, config.GetRetryDelay())
	}
	if config.GetMaxWorkers() != config.maxWorkers {
		t.Errorf("Expected maxWorkers %d, got %d", config.maxWorkers, config.GetMaxWorkers())
	}
	if config.GetMaxRetries() != config.maxRetries {
		t.Errorf("Expected maxRetries %d, got %d", config.maxRetries, config.GetMaxRetries())
	}
	if config.GetUseSSH() != config.useSSH {
		t.Errorf("Expected useSSH %t, got %t", config.useSSH, config.GetUseSSH())
	}

	beforeDefaultLoader := DefaultEnvLoader
	defer func() {
		DefaultEnvLoader = beforeDefaultLoader
	}()
	DefaultEnvLoader = loader
	config = NewConfig()
	if config.gitLabURL != expectConfig.gitLabURL {
		t.Errorf("Expected gitLabURL %s, got %s", expectConfig.gitLabURL, config.gitLabURL)
	}
	if config.accessToken != expectConfig.accessToken {
		t.Errorf("Expected accessToken %s, got %s", expectConfig.accessToken, config.accessToken)
	}
	if config.outputDir != expectConfig.outputDir {
		t.Errorf("Expected outputDir %s, got %s", expectConfig.outputDir, config.outputDir)
	}
	if !slices.Equal(config.groupIDs, expectConfig.groupIDs) {
		t.Errorf("Expected GroupIDs %s, got %s", expectConfig.groupIDs, config.groupIDs)
	}
	if !slices.Equal(config.skipGroupIDs, expectConfig.skipGroupIDs) {
		t.Errorf("Expected skipGroupIDs %s, got %s", expectConfig.skipGroupIDs, config.skipGroupIDs)
	}
	if config.retryDelay != expectConfig.retryDelay {
		t.Errorf("Expected retryDelay %s, got %s", expectConfig.retryDelay, config.retryDelay)
	}
	if config.maxWorkers != expectConfig.maxWorkers {
		t.Errorf("Expected maxWorkers %d, got %d", expectConfig.maxWorkers, config.maxWorkers)
	}
	if config.maxRetries != expectConfig.maxRetries {
		t.Errorf("Expected maxRetries %d, got %d", expectConfig.maxRetries, config.maxRetries)
	}
	if config.useSSH != expectConfig.useSSH {
		t.Errorf("Expected useSSH %t, got %t", expectConfig.useSSH, config.useSSH)
	}
}

func TestGetConfigSingleton(t *testing.T) {
	expectations := map[string]string{
		"RE_GITLAB_URL": "https://gitlab.example.com",
	}
	loader := NewMemoryEnvLoader(expectations)

	config1 := GetConfig(loader)
	config2 := GetConfig()

	if config1 != config2 {
		t.Error("GetConfig should return the same instance on subsequent calls")
	}

	if config1.gitLabURL != expectations["RE_GITLAB_URL"] {
		t.Errorf("Expected gitLabURL %s, got %s", expectations["RE_GITLAB_URL"], config1.gitLabURL)
	}
}

func TestExtractGroupIDs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"group1, group2, group3", []string{"group1", "group2", "group3"}},
		{"group1,   group2,", []string{"group1", "group2"}},
		{"group1,  group2,  group2,", []string{"group1", "group2"}}, //nolint:dupword
		{"", []string{}},
		{", ,, ,", []string{}},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := extractGroupIDs(test.input)
			if !slices.Equal(result, test.expected) {
				t.Errorf("Expected %v, got %v for input %s", test.expected, result, test.input)
			}
		})
	}
}
