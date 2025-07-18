package main

import (
	"context"
	"errors"
	"strconv"
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

		skipGroupIDs := cfg.GetSkipGroupIDs()
		skipGroups, err := fetchSkippedGroupIDs(ctx, client, skipGroupIDs)
		if err != nil {
			select {
			case <-ctx.Done():
			case errsChan <- err:
			}
			return
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
		skipGroupIDs := cfg.GetSkipGroupIDs()

		if len(groupIDs) == 0 {
			errsChan <- ErrorNoGroupIDs
			return
		}

		filteredGroupIDs := filterGroupIDs(groupIDs, skipGroupIDs)

		if len(filteredGroupIDs) == 0 {
			errsChan <- ErrorAllGroupIDsSkipped
			return
		}

		skipGroups, err := fetchSkippedGroupIDs(ctx, client, skipGroupIDs)
		if err != nil {
			select {
			case <-ctx.Done():
			case errsChan <- err:
			}
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

				order := []string{strconv.Itoa(group.id)}
				for len(order) > 0 {
					anGroupID := order[0]
					order = order[1:]

					groups, err := fetchSubGroups(ctx, client, anGroupID, skipGroups)
					if err != nil {
						select {
						case <-ctx.Done():
						case errsChan <- err:
						}
						return
					}

					for _, subGroup := range groups {
						order = append(order, strconv.Itoa(subGroup.id))

						select {
						case <-ctx.Done():
							return
						case dataChan <- subGroup:
						}
					}
				}
			}(groupID)
		}

		wg.Wait()
	}()

	return dataChan, errsChan
}

func fetchSkippedGroupIDs(ctx context.Context, client GroupsService, groupIDs []string) ([]int, error) {
	if len(groupIDs) == 0 {
		return []int{}, nil
	}

	var fetchErr *ErrorGroupFetching

	skipGroups := make([]int, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		group, err := fetchGroupByID(ctx, client, groupID)

		isNotFound := errors.As(err, &fetchErr) && fetchErr.IsGroupNotFound()
		if err != nil && !isNotFound {
			return nil, err
		}

		if group != nil {
			skipGroups = append(skipGroups, group.id)
		}
	}

	return skipGroups, nil
}

func fetchSubGroups(ctx context.Context, client GroupsService, groupId string, skipGroups []int) ([]*Group, error) {
	allAvailable := true
	opt := &gitlab.ListSubGroupsOptions{
		AllAvailable: &allAvailable,
		SkipGroups:   &skipGroups,
	}
	opt.PerPage = 100

	var result []*Group

	for {
		groups, resp, err := client.ListSubGroups(groupId, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, &ErrorSubGroupsFetching{groupId, err}
		}

		for _, subGroup := range groups {
			if subGroup == nil {
				continue
			}

			result = append(result, &Group{
				id:       subGroup.ID,
				fullPath: subGroup.FullPath,
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return result, nil
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
