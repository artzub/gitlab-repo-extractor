package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/artzub/gitlab-repo-extractor/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func run() error {
	ctx := context.Background()

	cfg := config.GetConfig()

	client, errClient := gitlab.NewClient(cfg.GetAccessToken(), gitlab.WithBaseURL(cfg.GetGitLabURL()))
	if errClient != nil {
		return fmt.Errorf("failed to create GitLab client: %w", errClient)
	}

	fmt.Println("Connected to GitLab:", cfg.GetGitLabURL())
	fmt.Println("Output directory:", cfg.GetOutputDir())
	fmt.Println("Group IDs:", strings.Join(cfg.GetGroupIDs(), ","))
	fmt.Println("Skip Group IDs:", strings.Join(cfg.GetSkipGroupIDs(), ","))
	fmt.Println("Using SSH:", cfg.GetUseSSH())
	fmt.Println("Max workers:", cfg.GetMaxWorkers())
	fmt.Println("Max retries:", cfg.GetMaxRetries())
	fmt.Println()

	gitlabClient := NewGitlab(client)

	groupsChan, groupErrsChan := fetchGroups(ctx, gitlabClient, cfg)
	projectsChan, projectErrsChan := proceedGroups(ctx, gitlabClient, groupsChan)

	errGroup := mergeChans(ctx, groupErrsChan, projectErrsChan)

	for projectsChan != nil || errGroup != nil {
		select {
		case project, ok := <-projectsChan:
			if !ok {
				projectsChan = nil
				continue
			}
			if project == nil {
				continue
			}

			fmt.Printf("Fetched project: %v\n", project)
		case err, ok := <-errGroup:
			if !ok {
				errGroup = nil
				continue
			}
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			break
		}
	}

	return nil
}
