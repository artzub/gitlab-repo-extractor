package main

import (
	"context"
	"sync"

	"github.com/artzub/gitlab-repo-extractor/config"
)

func proceedGroups(ctx context.Context, client ProjectsService, groupsChan <-chan *Group) (<-chan *Project, <-chan error) {
	dataChan := make(chan *Project)
	errsChan := make(chan error)

	go func() {
		defer func() {
			close(dataChan)
			close(errsChan)
		}()

		cfg := config.GetConfig()

		semaphore := make(chan struct{}, cfg.GetMaxWorkers())
		wg := &sync.WaitGroup{}

		for group := range groupsChan {
			if group == nil {
				continue
			}

			wg.Add(1)
			go func(group *Group) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				case semaphore <- struct{}{}:
				}

				defer func() { <-semaphore }()

				projectsChan, projErrsChan := fetchProjectByGroup(ctx, client, group)

				for projectsChan != nil || projErrsChan != nil {
					select {
					case <-ctx.Done():
						return
					case project, ok := <-projectsChan:
						if !ok {
							projectsChan = nil
							continue
						}
						if project == nil {
							continue
						}

						select {
						case dataChan <- project:
						case <-ctx.Done():
							return
						}
					case err, ok := <-projErrsChan:
						if !ok {
							projErrsChan = nil
							continue
						}

						if err != nil {
							select {
							case errsChan <- err:
							case <-ctx.Done():
								return
							}
						}
					}
				}
			}(group)
		}

		wg.Wait()
	}()

	return dataChan, errsChan
}
