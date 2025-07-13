package main

import (
	"log"
	"os"

	"github.com/artzub/gitlab-repo-extractor/config"
)

func main() {
	cfg := config.GetConfig()

	if cfg == nil || cfg.GetAccessToken() == "" {
		log.SetOutput(os.Stderr)
		log.Println("Error: GITLAB_TOKEN environment variable is required")
		log.Println("Please set your GitLab access token:")
		log.Println()
		log.Println("\texport RE_GITLAB_TOKEN=your_token_here")
		log.Println()
		log.Println("or in .env.local file:")
		log.Println()
		log.Println("\tRE_GITLAB_TOKEN=your_token_here")
		log.Fatalln()
	}

	log.SetOutput(os.Stdout)
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
