package main

import (
	"github.com/artzub/gitlab-repo-extractor/config"
	"log"
)

func main() {
	cfg := config.GetConfig()

	if cfg == nil || cfg.GetAccessToken() == "" {
		log.Println("Error: GITLAB_TOKEN environment variable is required")
		log.Println("\nPlease set your GitLab access token:")
		log.Println("\n\texport RE_GITLAB_TOKEN=your_token_here")
		log.Println("\nor in .env.local file:")
		log.Println("\n\tRE_GITLAB_TOKEN=your_token_here")
		log.Fatalln()
	}

	if err := run(); err != nil {
		log.Panicln(err)
	}
}
