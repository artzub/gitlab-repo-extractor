package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/artzub/gitlab-repo-extractor/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func run() error {
	ctx := context.Background()

	cfg := config.GetConfig()

	client, err := gitlab.NewClient(cfg.GetAccessToken(), gitlab.WithBaseURL(cfg.GetGitLabURL()))
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	fmt.Println("Connected to GitLab:", cfg.GetGitLabURL())
	fmt.Println("Output directory:", cfg.GetOutputDir())
	fmt.Println("Group IDs:", strings.Join(cfg.GetGroupIDs(), ","))
	fmt.Println("Skip Group IDs:", strings.Join(cfg.GetSkipGroupIDs(), ","))
	fmt.Println("Using SSH:", cfg.GetUseSSH())
	fmt.Println("Max workers:", cfg.GetMaxWorkers())
	fmt.Println("Max retries:", cfg.GetMaxRetries())
	fmt.Println()

	dataChan, errsChan := fetchGroups(ctx, NewGitlab(client))

	for dataChan != nil || errsChan != nil {
		select {
		case group, ok := <-dataChan:
			if !ok {
				dataChan = nil
				continue
			}
			if group == nil {
				continue
			}

			fmt.Printf("Fetched group: %s (ID: %d)\n", group.fullPath, group.id)

		case err, ok := <-errsChan:
			if !ok {
				errsChan = nil
				continue
			}
			if err != nil {
				printError(err)
			}
		case <-ctx.Done():
			break
		}
	}

	return nil
}
