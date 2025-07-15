package main

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/artzub/gitlab-repo-extractor/config"
)

type Result struct {
	project *Project
	err     error
}

func proceedProjects(ctx context.Context, cloner Cloner, projectsChan <-chan *Project) <-chan *Result {
	resultsChan := make(chan *Result)

	go func() {
		defer close(resultsChan)

		cfg := config.GetConfig()
		maxWorkers := cfg.GetMaxWorkers()
		outputDir := cfg.GetOutputDir()

		outputDirOnce := &sync.Once{}
		outputDirExists := atomic.Bool{}
		outputDirNotifyOnce := &sync.Once{}
		var outputDirErr error

		wg := &sync.WaitGroup{}
		wg.Add(maxWorkers)

		for range maxWorkers {
			go func() {
				defer wg.Done()

				for {
					select {
					case <-ctx.Done():
						return
					case project, ok := <-projectsChan:
						if !ok {
							return
						}

						if project == nil {
							continue
						}

						outputDirOnce.Do(func() {
							mkErr := cloner.GetOSWrapper().MakeDirAll(outputDir)
							outputDirErr = mkErr
							outputDirExists.Store(mkErr == nil)
						})

						if !outputDirExists.Load() {
							outputDirNotifyOnce.Do(func() {
								select {
								case <-ctx.Done():
								case resultsChan <- &Result{nil, &ErrorOutputDirNotCreated{
									outputDir,
									outputDirErr,
								}}:
								}
							})
							return
						}

						err := cloner.CloneProjectWithRetry(ctx, cfg, project)
						select {
						case <-ctx.Done():
							return
						case resultsChan <- &Result{project, err}:
						}
					}
				}
			}()
		}

		wg.Wait()
	}()

	return resultsChan
}
