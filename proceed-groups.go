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
		maxWorkers := cfg.GetMaxWorkers()

		wg := &sync.WaitGroup{}
		wg.Add(maxWorkers)

		for range maxWorkers {
			go func() {
				defer wg.Done()

				for {
					select {
					case <-ctx.Done():
						return
					case group, ok := <-groupsChan:
						if !ok {
							return
						}

						if group == nil {
							continue
						}

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
					}
				}
			}()
		}

		wg.Wait()
	}()

	return dataChan, errsChan
}
