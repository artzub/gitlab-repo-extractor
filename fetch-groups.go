package main

import (
	"context"
	"errors"
	"sync"

	"github.com/artzub/gitlab-repo-extractor/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type Group struct {
	id       int
	fullPath string
}

func fetchGroups(ctx context.Context, client GroupsService, cfg *config.Config) (<-chan *Group, <-chan error) {
	if len(cfg.GetGroupIDs()) > 0 {
		return fetchGroupsByIDs(ctx, client, cfg)
	}

	return fetchAllGroups(ctx, client, cfg)
}

func fetchAllGroups(ctx context.Context, client GroupsService, cfg *config.Config) (<-chan *Group, <-chan error) {
	dataChan := make(chan *Group)
	errsChan := make(chan error)

	go func() {
		defer func() {
			close(dataChan)
			close(errsChan)
		}()

		var skipGroups []int

		if len(cfg.GetSkipGroupIDs()) > 0 {
			for _, skipGroupID := range cfg.GetSkipGroupIDs() {
				var fetchErr *ErrorGroupFetching
				group, err := fetchGroupByID(ctx, client, skipGroupID)
				isNotFound := errors.As(err, &fetchErr) && fetchErr.IsGroupNotFound()
				if err != nil && !isNotFound {
					select {
					case <-ctx.Done():
					case errsChan <- err:
					}
					return
				}

				if !isNotFound {
					skipGroups = append(skipGroups, group.id)
				}
			}
		}

		topLevelOnly := false
		opt := &gitlab.ListGroupsOptions{
			SkipGroups:   &skipGroups,
			TopLevelOnly: &topLevelOnly,
		}
		opt.PerPage = 100

		for {
			groups, resp, err := client.ListGroups(opt, gitlab.WithContext(ctx))
			if err != nil {
				errsChan <- &ErrorGroupsFetching{err}
				return
			}

			for _, group := range groups {
				if group == nil {
					continue
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
			}

			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}()

	return dataChan, errsChan
}

func filterGroupIDs(groupIDs []string, skipGroupIDs []string) []string {
	skipGroups := map[string]struct{}{}
	for _, groupID := range skipGroupIDs {
		skipGroups[groupID] = struct{}{}
	}

	var filteredGroupIDs []string
	for _, groupID := range groupIDs {
		if _, skip := skipGroups[groupID]; !skip {
			filteredGroupIDs = append(filteredGroupIDs, groupID)
		}
	}

	return filteredGroupIDs
}

func fetchGroupsByIDs(ctx context.Context, client GroupsService, cfg *config.Config) (<-chan *Group, <-chan error) {
	dataChan := make(chan *Group)
	errsChan := make(chan error)

	go func() {
		defer func() {
			close(dataChan)
			close(errsChan)
		}()

		groupIDs := cfg.GetGroupIDs()

		if len(groupIDs) == 0 {
			errsChan <- ErrorNoGroupIDs
			return
		}

		filteredGroupIDs := filterGroupIDs(groupIDs, cfg.GetSkipGroupIDs())

		if len(filteredGroupIDs) == 0 {
			errsChan <- ErrorAllGroupIDsSkipped
			return
		}

		semaphore := make(chan struct{}, cfg.GetMaxWorkers())
		wg := &sync.WaitGroup{}

		for _, groupID := range filteredGroupIDs {
			wg.Add(1)
			go func(groupID string) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return // Exit if context is done
				case semaphore <- struct{}{}: // Acquire a worker slot
				}

				defer func() { <-semaphore }() // Release the worker slot

				group, err := fetchGroupByID(ctx, client, groupID)
				if err != nil {
					select {
					case <-ctx.Done():
					case errsChan <- err:
					}
					return
				}

				select {
				case <-ctx.Done():
				case dataChan <- group:
				}
			}(groupID)
		}

		wg.Wait()
	}()

	return dataChan, errsChan
}

func fetchGroupByID(ctx context.Context, client GroupsService, groupID string) (*Group, error) {
	group, _, err := client.GetGroup(groupID, &gitlab.GetGroupOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, &ErrorGroupFetching{groupID, err}
	}

	if group == nil {
		return nil, &ErrorGroupFetching{groupID, ErrorNoGroupPassed}
	}

	return &Group{
		id:       group.ID,
		fullPath: group.FullPath,
	}, nil
}
