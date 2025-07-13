package config

import "runtime"

var (
	GitlabTokenKey = "RE_GITLAB_TOKEN"

	GitlabURLKey     = "RE_GITLAB_URL"
	DefaultGitlabURL = "https://gitlab.com"

	OutputDirKey     = "RE_OUTPUT_DIR"
	DefaultOutputDir = "./gitlab-repos"

	UseSSHKey     = "RE_USE_SSH"
	DefaultUseSSH = "false"

	GroupIDsKey     = "RE_GROUP_IDS"
	SkipGroupIDsKey = "RE_SKIP_GROUP_IDS"

	RetryDelayKey     = "RE_RETRY_DELAY_SECONDS"
	DefaultRetryDelay = 2

	MaxRetriesKey     = "RE_MAX_RETRIES"
	DefaultMaxRetries = 3

	MaxWorkersKey     = "RE_MAX_WORKERS"
	DefaultMaxWorkers = runtime.NumCPU()
)
