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

	log.Println("Connected to GitLab:", cfg.GetGitLabURL())
	log.Println("Output directory:", cfg.GetOutputDir())
	log.Println("Group IDs:", strings.Join(cfg.GetGroupIDs(), ","))
	log.Println("Skip Group IDs:", strings.Join(cfg.GetSkipGroupIDs(), ","))
	log.Println("Using SSH:", cfg.GetUseSSH())
	log.Println("Max workers:", cfg.GetMaxWorkers())
	log.Println("Max retries:", cfg.GetMaxRetries())
	log.Println()

	gitlabClient := NewGitlab(client)

	groupsChan, groupErrsChan := fetchGroups(ctx, gitlabClient, cfg)
	groupsChans := teeChan(ctx, groupsChan, 2)

	projectsChan, projectErrsChan := proceedGroups(ctx, gitlabClient, groupsChans[0])
	projectsChans := teeChan(ctx, projectsChan, 2)

	jobsChan := proceedProjects(ctx, NewGitCloner(), projectsChans[0])

	errGroup := mergeChans(ctx, groupErrsChan, projectErrsChan)

	counter := NewProgressCounter(0)
	errorsCounter := NewProgressCounter(0)

	go func() {
		for group := range groupsChans[1] {
			log.Printf("Fetching projects of group: %s\n", group.fullPath)
		}
	}()

	go func() {
		for project := range projectsChans[1] {
			log.Printf("Cloning project: %s\n", project.pathWithNamespace)
		}
	}()

	for jobsChan != nil || errGroup != nil {
		select {
		case result, ok := <-jobsChan:
			if !ok {
				jobsChan = nil
				continue
			}
			if result == nil {
				continue
			}

			if result.err != nil {
				counter.Update(false)
				projectPath := "unknown"
				if result.project != nil {
					projectPath = result.project.pathWithNamespace
				}
				log.Printf("Error cloning project: %s, %v\n", projectPath, result.err)
				continue
			}

			log.Printf("Successfully cloned project: %s\n", result.project.pathWithNamespace)

			counter.Update(true)
		case err, ok := <-errGroup:
			if !ok {
				errGroup = nil
				continue
			}
			if err != nil {
				errorsCounter.Update(false)
				log.Println(err)
			}
		case <-ctx.Done():
			break
		}
	}

	log.Println("Processing completed.")
	_, completed, success, errors := counter.GetStats()
	log.Println("Total projects processed:", completed)
	log.Println("Successful:", success)
	log.Println("Errors:", errors)

	fetchErrors := errorsCounter.GetErrors()
	if fetchErrors > 0 {
		log.Println("Errors occurred while fetching groups or projects:", fetchErrors)
	}

	return nil
}
