package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/artzub/gitlab-repo-extractor/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type Group struct {
	id       int
	fullPath string
}

func fetchGroups(ctx context.Context, client GroupsService) (<-chan *Group, <-chan error) {
	cfg := config.GetConfig()

	if len(cfg.GetGroupIDs()) > 0 {
		return fetchGroupsByIDs(ctx, client, cfg.GetGroupIDs())
	}

	return fetchAllGroups(ctx, client)
}

func convertToInt(items []string) *[]int {
	ints := make([]int, len(items))
	for _, item := range items {
		value, err := strconv.Atoi(item)
		if err != nil {
			ints = append(ints, value)
		}
	}
	return &ints
}

func fetchAllGroups(ctx context.Context, client GroupsService) (<-chan *Group, <-chan error) {
	dataChan := make(chan *Group)
	errsChan := make(chan error)

	go func() {
		defer func() {
			close(dataChan)
			close(errsChan)
		}()

		cfg := config.GetConfig()

		wg := &sync.WaitGroup{}

		proceedGroups := func(groups []*gitlab.Group) {
			defer wg.Done()
			for _, group := range groups {
				prepare := &Group{
					id:       group.ID,
					fullPath: group.FullPath,
				}

				select {
				case <-ctx.Done():
					return
				case dataChan <- prepare:
				}
			}
		}

		topLevelOnly := false
		opt := &gitlab.ListGroupsOptions{
			SkipGroups:   convertToInt(cfg.GetSkipGroupIDs()),
			TopLevelOnly: &topLevelOnly,
		}
		opt.PerPage = 100

		for {
			groups, resp, err := client.ListGroups(opt, gitlab.WithContext(ctx))
			if err != nil {
				errsChan <- fmt.Errorf("failed to fetch groups: %w", err)
				return
			}

			wg.Add(1)
			go proceedGroups(groups)

			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}

		wg.Wait()
	}()

	return dataChan, errsChan
}

func fetchGroupsByIDs(ctx context.Context, client GroupsService, groupIDs []string) (<-chan *Group, <-chan error) {
	dataChan := make(chan *Group)
	errsChan := make(chan error)

	go func() {
		defer func() {
			close(dataChan)
			close(errsChan)
		}()

		if len(groupIDs) == 0 {
			errsChan <- fmt.Errorf("no group IDs provided")
			return
		}

		cfg := config.GetConfig()

		skipGroups := map[string]struct{}{}
		for _, groupID := range cfg.GetSkipGroupIDs() {
			skipGroups[groupID] = struct{}{}
		}

		var filteredGroupIDs []string
		for _, groupID := range groupIDs {
			if _, skip := skipGroups[groupID]; !skip {
				filteredGroupIDs = append(filteredGroupIDs, groupID)
			}
		}

		if len(filteredGroupIDs) == 0 {
			errsChan <- fmt.Errorf("all group IDs are skipped")
			return
		}

		allowed := make(chan struct{}, cfg.GetMaxWorkers())
		wg := &sync.WaitGroup{}

		for _, groupID := range filteredGroupIDs {
			wg.Add(1)
			go func(groupID string) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return // Exit if context is done
				case allowed <- struct{}{}: // Acquire a worker slot
					defer func() {
						println("Releasing worker slot for group:", groupID)
						<-allowed
					}() // Release the worker slot
				}

				println("process group:", groupID)

				group, _, err := client.GetGroup(groupID, &gitlab.GetGroupOptions{}, gitlab.WithContext(ctx))
				if err != nil {
					select {
					case <-ctx.Done():
					case errsChan <- err:
					}
					return
				}

				if group == nil {
					select {
					case <-ctx.Done():
					case errsChan <- fmt.Errorf("group not found: %v", groupID):
					}
					return
				}

				prepare := &Group{
					id:       group.ID,
					fullPath: group.FullPath,
				}

				select {
				case <-ctx.Done():
					return
				case dataChan <- prepare:
				}
			}(groupID)
		}

		wg.Wait()
	}()

	return dataChan, errsChan
}
