package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/artzub/gitlab-repo-extractor/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type FakeGitlab struct {
	groups map[string]*gitlab.Group
	sleep  time.Duration
}

func NewFakeGitlab(groups map[string]*gitlab.Group, sleeps ...time.Duration) *FakeGitlab {
	sleep := 0 * time.Second
	if len(sleeps) > 0 {
		sleep = sleeps[0]
	}

	return &FakeGitlab{
		groups: groups,
		sleep:  sleep,
	}
}

func (f *FakeGitlab) GetGroup(gid string, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
	if f.sleep > 0 {
		time.Sleep(f.sleep)
	}

	if group, exists := f.groups[gid]; exists {
		return group, nil, nil
	}
	return nil, nil, fmt.Errorf("group %s: not found %d", gid, 404)
}

func (f *FakeGitlab) ListGroups(opt *gitlab.ListGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
	if f.sleep > 0 {
		time.Sleep(f.sleep)
	}

	return slices.Collect(maps.Values(f.groups)), &gitlab.Response{
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
						t.Errorf("expected group %s to be fetched, but it was not", groupID)
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
				expectedErr := ErrorGroupNotFound("example_group1")

				gitlabGroups := getGitlabGroups(1)
				client := NewFakeGitlab(gitlabGroups)

				ctx := context.Background()
				dataChan, errsChan := fetchGroupsByIDs(ctx, client, cfg)

				select {
				case err := <-errsChan:
					if err == nil || !errors.Is(err, expectedErr) {
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
	// TODO add tests for fetchAllGroups function.
}

func TestFetchGroups(t *testing.T) {
	// TODO add tests for fetchGroups function.
}
