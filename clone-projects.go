package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/artzub/gitlab-repo-extractor/config"
)

type Cloner interface {
	GetOSWrapper() OSWrapper
	CloneProjectWithRetry(ctx context.Context, cfg *config.Config, project *Project) error
	cloneProject(ctx context.Context, cfg *config.Config, project *Project) error
}

type GitCloner struct {
	osWrapper OSWrapper
}

func NewGitCloner(osWrappers ...OSWrapper) *GitCloner {
	var osWrapper OSWrapper

	if len(osWrappers) > 0 {
		osWrapper = osWrappers[0]
	}

	if osWrapper == nil {
		osWrapper = GetDefaultOSWrapper()
	}

	return &GitCloner{
		osWrapper: osWrapper,
	}
}

func (c *GitCloner) GetOSWrapper() OSWrapper {
	return c.osWrapper
}

func (c *GitCloner) CloneProjectWithRetry(ctx context.Context, cfg *config.Config, project *Project) error {
	if cfg == nil {
		return ErrorNoConfigPassed
	}

	if project == nil {
		return ErrorNoProjectsPassed
	}

	var lastErr error

	maxRetries := cfg.GetMaxRetries()
	retryDelay := cfg.GetRetryDelay()

	for attempt := range maxRetries {
		if attempt > 0 {
			// log.Printf("Retrying to clone project '%s' (%d/%d)\n", project.pathWithNamespace, attempt+1, maxRetries)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		err := c.cloneProject(ctx, cfg, project)
		if err == nil {
			return nil
		}

		lastErr = err

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return &ErrorFailedAfterRetries{maxRetries, lastErr}
}

func (c *GitCloner) cloneProject(ctx context.Context, cfg *config.Config, project *Project) error {
	if cfg == nil {
		return ErrorNoConfigPassed
	}

	if project == nil {
		return ErrorNoProjectsPassed
	}

	useSSH := cfg.GetUseSSH()
	token := cfg.GetAccessToken()

	projectDir := project.pathWithNamespace

	ok, err := c.osWrapper.IsDirExists(projectDir)
	if ok || err != nil {
		if err != nil {
			return &ErrorDirExistsCheck{projectDir, err}
		}

		return ErrorDirExists(projectDir)
	}

	cloneURL := project.httpURLToRepo
	if useSSH {
		cloneURL = project.sshURLToRepo
	}

	url := cloneURL
	if !useSSH {
		url = addTokenToHTTPSURL(cloneURL, token)
	}

	output, err := c.osWrapper.ExecuteCommand(ctx, "git", "clone", url, projectDir)
	if err != nil {
		_ = c.osWrapper.RemoveAll(projectDir)
		return &ErrorFailedToCloneProject{projectDir, err, output}
	}

	return nil
}

func addTokenToHTTPSURL(gitURL, token string) string {
	if len(token) > 0 && strings.HasPrefix(gitURL, "https://") {
		return strings.Replace(gitURL, "https://", fmt.Sprintf("https://oauth2:%s@", token), 1)
	}
	return gitURL
}
