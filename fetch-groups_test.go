package main

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/artzub/gitlab-repo-extractor/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type FakeGitlabGroups struct {
	groups      map[string]*gitlab.Group
	sleep       time.Duration
	fetchErr    error
	fetchAllErr error
}

func NewFakeGitlab(groups map[string]*gitlab.Group, sleeps ...time.Duration) *FakeGitlabGroups {
	sleep := 0 * time.Second
	if len(sleeps) > 0 {
		sleep = sleeps[0]
	}

	return &FakeGitlabGroups{
		groups: groups,
		sleep:  sleep,
	}
}

func (f *FakeGitlabGroups) GetGroup(gid string, _ *gitlab.GetGroupOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
	if f.sleep > 0 {
		time.Sleep(f.sleep)
	}

	if f.fetchErr != nil {
		return nil, nil, f.fetchErr
	}

	if group, exists := f.groups[gid]; exists {
		return group, nil, nil
	}
	return nil, nil, fmt.Errorf("group %s: not found %d", gid, 404)
}

func (f *FakeGitlabGroups) ListGroups(opt *gitlab.ListGroupsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
	if f.sleep > 0 {
		time.Sleep(f.sleep)
	}

	if f.fetchAllErr != nil {
		return nil, nil, f.fetchAllErr
	}

	skipGroupIDs := opt.SkipGroups

	var filteredGroups []*gitlab.Group

	for _, group := range f.groups {
		if skipGroupIDs != nil && slices.Contains(*skipGroupIDs, group.ID) {
			continue
		}
		filteredGroups = append(filteredGroups, group)
	}

	return filteredGroups, &gitlab.Response{
		NextPage: 0,
	}, nil
}

func getGitlabGroups(shouldBeNil ...int) map[string]*gitlab.Group {
	gitlabGroups := map[string]*gitlab.Group{}
	hasShouldBeNil := len(shouldBeNil) > 0
	for index := range 10 {
		key := fmt.Sprintf("example_group%d", index)
		if hasShouldBeNil && slices.Contains(shouldBeNil, index) {
			gitlabGroups[key] = nil
			continue
		}

		gitlabGroups[key] = &gitlab.Group{
			ID:       index,
			FullPath: key,
		}
	}
	return gitlabGroups
}

func TestFilterGroups(t *testing.T) {
	tests := []struct {
		name             string
		groupIDs         []string
		skipGroupIDs     []string
		expectedGroupIDs []string
	}{
		{
			name:             "no groups",
			groupIDs:         []string{},
			skipGroupIDs:     []string{},
			expectedGroupIDs: []string{},
		},
		{
			name:             "some groups",
			groupIDs:         []string{"example_group1", "example_group2", "example_group3", "example_group2"},
			skipGroupIDs:     []string{"example_group2"},
			expectedGroupIDs: []string{"example_group1", "example_group3"},
		},
		{
			name:             "all groups skipped",
			groupIDs:         []string{"example_group1", "example_group2"},
			skipGroupIDs:     []string{"example_group1", "example_group2"},
			expectedGroupIDs: []string{},
		},
		{
			name:             "no groups skipped",
			groupIDs:         []string{"example_group1", "example_group2"},
			skipGroupIDs:     []string{"example_group3"},
			expectedGroupIDs: []string{"example_group1", "example_group2"},
		},
		{
			name:             "no groups skipped, skipGroupIDs empty",
			groupIDs:         []string{"example_group1", "example_group2"},
			skipGroupIDs:     []string{},
			expectedGroupIDs: []string{"example_group1", "example_group2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filtered := filterGroupIDs(test.groupIDs, test.skipGroupIDs)
			if !slices.Equal(filtered, test.expectedGroupIDs) {
				t.Errorf("expected %v, got %v", test.expectedGroupIDs, filtered)
			}
		})
	}
}

func TestFetchGroupByID(t *testing.T) {
	gitlabGroups := getGitlabGroups()

	tests := []struct {
		name        string
		groups      map[string]*gitlab.Group
		groupID     string
		expected    *Group
		shouldBeNil bool
		throwErr    error
		expectedErr error
	}{
		{
			name:    "should fetch existing group",
			groups:  gitlabGroups,
			groupID: "example_group1",
			expected: &Group{
				id:       gitlabGroups["example_group1"].ID,
				fullPath: gitlabGroups["example_group1"].FullPath,
			},
		},
		{
			name:        "should return error for non-existing group",
			groups:      gitlabGroups,
			groupID:     "non_existent_group",
			throwErr:    errors.New("group not found"),
			expectedErr: &ErrorGroupFetching{"non_existent_group", errors.New("group not found")},
		},
		{
			name:        "should return error for nil group",
			groups:      getGitlabGroups(1), // only example_group1 will be nil
			groupID:     "example_group1",
			expectedErr: &ErrorGroupFetching{"example_group1", ErrorNoGroupPassed},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := NewFakeGitlab(test.groups)
			if test.throwErr != nil {
				client.fetchErr = test.throwErr
			}

			ctx := context.Background()
			group, err := fetchGroupByID(ctx, client, test.groupID)
			if err != nil {
				var fetchErr *ErrorGroupFetching
				if errors.As(err, &fetchErr) && err.Error() == test.expectedErr.Error() {
					return
				}

				t.Fatalf("unexpected error: %v", err)
			}

			if group.fullPath != test.expected.fullPath {
				t.Errorf("expected group %v, got %v", test.expected, group)
			}
		})
	}
}

func TestFetchGroupsByIDs(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T) (<-chan *Group, <-chan error)
	}{
		{
			name: "should fetch groups by IDs",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
					config.GroupIDsKey:     "example_group1, example_group5, example_group2",
					config.SkipGroupIDsKey: "example_group2",
				}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)

				groupIDs := cfg.GetGroupIDs()
				skipGroupIDs := cfg.GetSkipGroupIDs()
				filteredGroupIDs := filterGroupIDs(groupIDs, skipGroupIDs)

				ctx := context.Background()
				dataChan, errsChan := fetchGroupsByIDs(ctx, client, cfg)

				groups := map[string]struct{}{}

				// expect to receive only the groups that are not skipped
				for range filteredGroupIDs {
					select {
					case group := <-dataChan:
						if group != nil {
							groups[group.fullPath] = struct{}{}
						}
					case err := <-errsChan:
						t.Fatalf("unexpected error: %v", err)
					case <-time.After(time.Second):
						t.Fatal("timeout waiting for group data")
					}
				}

				for _, groupID := range filteredGroupIDs {
					if _, exists := groups[groupID]; !exists {
						t.Fatalf("expected group %s to be fetched, but it was not", groupID)
					}
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should return error if no group IDs provided",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
					config.GroupIDsKey: "",
				}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)

				ctx := context.Background()
				dataChan, errsChan := fetchGroupsByIDs(ctx, client, cfg)

				select {
				case err := <-errsChan:
					if err == nil || !errors.Is(err, ErrorNoGroupIDs) {
						t.Fatalf("expected error '%v', got %v", ErrorNoGroupIDs, err)
					}
				case <-dataChan:
					t.Fatal("expected no data to be sent")
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for error")
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should return error if all group IDs are skipped",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
					config.GroupIDsKey:     "example_group1, example_group2",
					config.SkipGroupIDsKey: "example_group1, example_group2",
				}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)

				ctx := context.Background()
				dataChan, errsChan := fetchGroupsByIDs(ctx, client, cfg)

				select {
				case err := <-errsChan:
					if err == nil || !errors.Is(err, ErrorAllGroupIDsSkipped) {
						t.Fatalf("expected error '%v', got %v", ErrorAllGroupIDsSkipped, err)
					}
				case <-dataChan:
					t.Fatal("expected no data to be sent")
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for error")
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should handle context cancellation",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
					config.GroupIDsKey: "example_group1, example_group2",
				}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				var dataChan <-chan *Group
				var errsChan <-chan error

				wg := &sync.WaitGroup{}
				wg.Add(1)

				go func() {
					defer wg.Done()

					dataChan, errsChan = fetchGroupsByIDs(ctx, client, cfg)
				}()
				wg.Wait()

				time.Sleep(50 * time.Millisecond)

				select {
				case _, ok := <-errsChan:
					if ok {
						t.Fatal("expected no error")
					}
				case _, ok := <-dataChan:
					if ok {
						t.Fatal("expected no data to be sent")
					}
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for error")
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should error if group not found",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
					config.GroupIDsKey: "non_existent_group",
				}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)

				ctx := context.Background()
				dataChan, errsChan := fetchGroupsByIDs(ctx, client, cfg)

				select {
				case err := <-errsChan:
					if err == nil {
						t.Fatalf("expected an not found error, got %v", err)
					}
				case <-dataChan:
					t.Fatal("expected no data to be sent")
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for error")
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should error if a group is nil",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
					config.GroupIDsKey: "example_group1",
				}))
				expectedErr := ErrorGroupFetching{"example_group1", ErrorNoGroupPassed}

				gitlabGroups := getGitlabGroups(1)
				client := NewFakeGitlab(gitlabGroups)

				ctx := context.Background()
				dataChan, errsChan := fetchGroupsByIDs(ctx, client, cfg)

				select {
				case err := <-errsChan:
					if err == nil || expectedErr.Error() != err.Error() {
						t.Fatalf("expected error '%v', got %v", expectedErr, err)
					}
				case <-dataChan:
					t.Fatal("expected no data to be sent")
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for error")
				}

				return dataChan, errsChan
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dataChan, errsChan := test.fn(t)

			// cannot to use a common function since golang does not let pass channel in different types,
			// so that means we cannot do so: `func checkChannelClosed(ch <-chan any) {}`
			// or `type wrapper struct { ch <-chan any }`
			select {
			case _, ok := <-dataChan:
				if ok {
					t.Fatal("channel should be closed")
				}
			case <-time.After(50 * time.Millisecond):
				t.Fatal("timeout waiting for channel to close")
			}

			select {
			case _, ok := <-errsChan:
				if ok {
					t.Fatal("channel should be closed")
				}
			case <-time.After(50 * time.Millisecond):
				t.Fatal("timeout waiting for channel to close")
			}
		})
	}
}

func TestFetchAllGroups(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T) (<-chan *Group, <-chan error)
	}{
		{
			name: "should fetch all groups",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)

				ctx := context.Background()
				dataChan, errsChan := fetchAllGroups(ctx, client, cfg)

				groups := map[string]struct{}{}

				for range gitlabGroups {
					select {
					case group := <-dataChan:
						if group != nil {
							groups[group.fullPath] = struct{}{}
						}
					case err := <-errsChan:
						t.Fatalf("unexpected error: %v", err)
					case <-time.After(time.Second):
						t.Fatal("timeout waiting for group data")
					}
				}

				for _, aGroup := range gitlabGroups {
					if _, exists := groups[aGroup.FullPath]; !exists {
						t.Fatalf("expected group %s to be fetched, but it was not", aGroup.FullPath)
					}
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should return error if fetching error",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)
				client.fetchAllErr = errors.New("fetching error")

				expectedErr := &ErrorGroupsFetching{errors.New("fetching error")}

				ctx := context.Background()
				dataChan, errsChan := fetchAllGroups(ctx, client, cfg)

				select {
				case err := <-errsChan:
					if err == nil || err.Error() != expectedErr.Error() {
						t.Fatalf("expected error '%v', got %v", expectedErr, err)
					}
				case <-dataChan:
					t.Fatal("expected no data to be sent")
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for error")
				}

				return dataChan, errsChan
			},
		},
		{
			name: "Respect skip group IDs",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
					config.SkipGroupIDsKey: "example_group2",
				}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)
				skipGroupIDs := cfg.GetSkipGroupIDs()

				ctx := context.Background()
				dataChan, errsChan := fetchAllGroups(ctx, client, cfg)

				groups := map[string]struct{}{}

				dataDone := false
				errsDone := false

				for !dataDone || !errsDone {
					select {
					case group, ok := <-dataChan:
						if !ok {
							dataDone = true
							continue
						}
						if group != nil {
							groups[group.fullPath] = struct{}{}
						}
					case err, ok := <-errsChan:
						if !ok {
							errsDone = true
							continue
						}
						t.Fatalf("unexpected error: %v", err)
					case <-time.After(time.Second):
						t.Fatal("timeout waiting for group data")
					}
				}

				for key := range groups {
					if slices.Contains(skipGroupIDs, key) {
						t.Fatalf("group %s should be skipped, but it was fetched", key)
					}
					if _, exists := gitlabGroups[key]; !exists {
						t.Fatalf("expected group %s to be fetched, but it was not", key)
					}
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should not throw error if an skip group is not found",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
					config.SkipGroupIDsKey: "example_group20",
				}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)
				skipGroupIDs := cfg.GetSkipGroupIDs()

				ctx := context.Background()
				dataChan, errsChan := fetchAllGroups(ctx, client, cfg)

				groups := map[string]struct{}{}

				dataDone := false
				errsDone := false

				for !dataDone || !errsDone {
					select {
					case group, ok := <-dataChan:
						if !ok {
							dataDone = true
							continue
						}
						if group != nil {
							groups[group.fullPath] = struct{}{}
						}
					case err, ok := <-errsChan:
						if !ok {
							errsDone = true
							continue
						}
						t.Fatalf("unexpected error: %v", err)
					case <-time.After(time.Second):
						t.Fatal("timeout waiting for group data")
					}
				}

				for key := range groups {
					if slices.Contains(skipGroupIDs, key) {
						t.Fatalf("group %s should be skipped, but it was fetched", key)
					}
					if _, exists := gitlabGroups[key]; !exists {
						t.Fatalf("expected group %s to be fetched, but it was not", key)
					}
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should throw error if fetching a skip group is failed",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
					config.SkipGroupIDsKey: "example_group20",
				}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)
				client.fetchErr = errors.New("fetching error for skip group")
				expectedErr := &ErrorGroupFetching{"example_group20", client.fetchErr}

				ctx := context.Background()
				dataChan, errsChan := fetchAllGroups(ctx, client, cfg)

				groups := map[string]struct{}{}

				dataDone := false
				errsDone := false

				for !dataDone || !errsDone {
					select {
					case group, ok := <-dataChan:
						if !ok {
							dataDone = true
							continue
						}
						if group != nil {
							groups[group.fullPath] = struct{}{}
						}
					case err, ok := <-errsChan:
						if !ok {
							errsDone = true
							continue
						}

						if err == nil || err.Error() != expectedErr.Error() {
							t.Fatalf("unexpected error: %v", err)
						}
					case <-time.After(time.Second):
						t.Fatal("timeout waiting for group data")
					}
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should handle context cancellation",
			fn: func(t *testing.T) (<-chan *Group, <-chan error) {
				cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{}))

				gitlabGroups := getGitlabGroups()
				client := NewFakeGitlab(gitlabGroups)

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				var dataChan <-chan *Group
				var errsChan <-chan error

				wg := &sync.WaitGroup{}
				wg.Add(1)

				go func() {
					defer wg.Done()

					dataChan, errsChan = fetchAllGroups(ctx, client, cfg)
				}()
				wg.Wait()

				time.Sleep(50 * time.Millisecond)

				select {
				case _, ok := <-errsChan:
					if ok {
						t.Fatal("expected no error")
					}
				case _, ok := <-dataChan:
					if ok {
						t.Fatal("expected no data to be sent")
					}
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for error")
				}

				return dataChan, errsChan
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dataChan, errsChan := test.fn(t)

			select {
			case _, ok := <-dataChan:
				if ok {
					t.Fatal("channel should be closed")
				}
			case <-time.After(50 * time.Millisecond):
				t.Fatal("timeout waiting for channel to close")
			}

			select {
			case _, ok := <-errsChan:
				if ok {
					t.Fatal("channel should be closed")
				}
			case <-time.After(50 * time.Millisecond):
				t.Fatal("timeout waiting for channel to close")
			}
		})
	}
}

func toEntries[K comparable](seq iter.Seq[K]) iter.Seq2[K, struct{}] {
	return func(yield func(K, struct{}) bool) {
		for k := range seq {
			if !yield(k, struct{}{}) {
				return
			}
		}
	}
}

func TestFetchGroups(t *testing.T) {
	gitlabGroups := getGitlabGroups()
	tests := []struct {
		name           string
		cfg            *config.Config
		expectedGroups map[string]struct{}
	}{
		{
			name:           "should fetch all groups",
			cfg:            config.NewConfig(config.NewMemoryEnvLoader(map[string]string{})),
			expectedGroups: maps.Collect(toEntries(maps.Keys(gitlabGroups))),
		},
		{
			name: "should fetch groups by IDs",
			cfg: config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
				config.GroupIDsKey: "example_group1, example_group2",
			})),
			expectedGroups: map[string]struct{}{
				"example_group1": {},
				"example_group2": {},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := NewFakeGitlab(gitlabGroups)
			ctx := context.Background()

			dataChan, errsChan := fetchGroups(ctx, client, test.cfg)

			groups := map[string]struct{}{}
			dataDone := false
			errsDone := false
			for !dataDone || !errsDone {
				select {
				case group, ok := <-dataChan:
					if !ok {
						dataDone = true
						continue
					}
					if group != nil {
						groups[group.fullPath] = struct{}{}
					}
				case err, ok := <-errsChan:
					if !ok {
						errsDone = true
						continue
					}
					t.Fatalf("unexpected error: %v", err)
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for group data")
				}
			}

			if len(groups) != len(test.expectedGroups) {
				t.Fatalf("expected %d groups, got %d", len(test.expectedGroups), len(groups))
			}

			for group := range groups {
				if _, exists := test.expectedGroups[group]; !exists {
					t.Fatalf("expected group %s to be fetched, but it was not", group)
				}
			}
		})
	}
}
