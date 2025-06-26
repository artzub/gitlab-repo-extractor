package main

import (
	"fmt"
	"os"

	"github.com/artzub/gitlab-repo-extractor/config"
)

func printError(messages ...any) {
	_, _ = fmt.Fprintln(os.Stderr, messages...)
}

func main() {
	cfg := config.GetConfig()

	if cfg == nil || cfg.GetAccessToken() == "" {
		printError("Error: GITLAB_TOKEN environment variable is required")
		printError("\nPlease set your GitLab access token:")
		printError("\n\texport RE_GITLAB_TOKEN=your_token_here")
		printError("\nor in .env.local file:")
		printError("\n\tRE_GITLAB_TOKEN=your_token_here")
		printError()

		os.Exit(1)
	}

	if err := run(); err != nil {
		printError(err)
		os.Exit(1)
	}
}
