package main

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type FakeGitlabProjects struct {
	nextPage int
	projects map[int]map[int]*gitlab.Project
	fetchErr error
}

func (f *FakeGitlabProjects) ListGroupProjects(gid int, opt *gitlab.ListGroupProjectsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
	if f.fetchErr != nil {
		return nil, nil, f.fetchErr
	}

	projects, exists := f.projects[gid]
	if exists {
		nextPage := f.nextPage
		if opt.Page == nextPage {
			nextPage = 0
		}
		return slices.Collect(maps.Values(projects)), &gitlab.Response{
			NextPage: nextPage,
		}, nil
	}

	return nil, nil, errors.New("group not found")
}

func getFakeProjects() map[int]map[int]*gitlab.Project {
	groups := map[int]map[int]*gitlab.Project{}

	projects := map[int]*gitlab.Project{}

	for id := range 10 {
		projects[id] = &gitlab.Project{ID: id}
	}

	groups[1] = projects

	return groups
}

func TestFetchProjects(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T) (<-chan *Project, <-chan error)
	}{
		{
			name: "Passed nil group",
			fn: func(t *testing.T) (<-chan *Project, <-chan error) {
				dataChan, errsChan := fetchProjectByGroup(context.Background(), &FakeGitlabProjects{}, nil)

				select {
				case err := <-errsChan:
					if err == nil || !errors.Is(err, ErrorNoGroupPassed) {
						t.Fatalf("expected error 'no group passed', got %v", err)
					}
				case <-time.After(50 * time.Millisecond):
					t.Fatal("timeout waiting for error channel")
				}

				return dataChan, errsChan
			},
		},
		{
			name: "Throw error when fetching projects",
			fn: func(t *testing.T) (<-chan *Project, <-chan error) {
				group := &Group{id: 1}
				fetchErr := errors.New("fetch error")
				dataChan, errsChan := fetchProjectByGroup(context.Background(), &FakeGitlabProjects{
					fetchErr: fetchErr,
				}, group)

				expectedErr := &ErrorProjectsFetching{group.id, fetchErr}

				select {
				case err := <-errsChan:
					if err == nil || err.Error() != expectedErr.Error() {
						t.Fatalf("expected error %v, got %v", expectedErr, err)
					}
				case <-time.After(50 * time.Millisecond):
					t.Fatal("timeout waiting for error channel")
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should handle context cancellation",
			fn: func(t *testing.T) (<-chan *Project, <-chan error) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				projects := getFakeProjects()

				var dataChan <-chan *Project
				var errsChan <-chan error

				wg := &sync.WaitGroup{}
				wg.Add(1)

				go func() {
					defer wg.Done()

					dataChan, errsChan = fetchProjectByGroup(ctx, &FakeGitlabProjects{
						projects: projects,
					}, &Group{id: 1})
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
			name: "Fetch projects successfully",
			fn: func(t *testing.T) (<-chan *Project, <-chan error) {
				projects := getFakeProjects()
				dataChan, errsChan := fetchProjectByGroup(context.Background(), &FakeGitlabProjects{
					projects: projects,
				}, &Group{id: 1})

				received := map[int]struct{}{}

				dataDone := false
				errsDone := false

				for !dataDone || !errsDone {
					select {
					case project, ok := <-dataChan:
						if !ok {
							dataDone = true
							continue
						}
						received[project.id] = struct{}{}
					case err, ok := <-errsChan:
						if !ok {
							errsDone = true
							continue
						}
						t.Fatalf("unexpected error: %v", err)
					case <-time.After(50 * time.Millisecond):
						t.Fatal("timeout waiting for channels to close")
					}
				}

				if len(received) != len(projects[1]) {
					t.Fatalf("expected %d projects, got %d", len(projects[1]), len(received))
				}

				for id := range projects[1] {
					if _, exists := received[id]; !exists {
						t.Fatalf("expected project with id %d, but it was not received", id)
					}
				}

				return dataChan, errsChan
			},
		},
		{
			name: "should work with pagination",
			fn: func(t *testing.T) (<-chan *Project, <-chan error) {
				projects := getFakeProjects()
				dataChan, errsChan := fetchProjectByGroup(context.Background(), &FakeGitlabProjects{
					projects: projects,
					nextPage: 1,
				}, &Group{id: 1})

				received := map[int]int{}

				dataDone := false
				errsDone := false

				for !dataDone || !errsDone {
					select {
					case project, ok := <-dataChan:
						if !ok {
							dataDone = true
							continue
						}
						received[project.id]++
					case err, ok := <-errsChan:
						if !ok {
							errsDone = true
							continue
						}
						t.Fatalf("unexpected error: %v", err)
					case <-time.After(50 * time.Millisecond):
						t.Fatal("timeout waiting for channels to close")
					}
				}

				if len(received) != len(projects[1]) {
					t.Fatalf("expected %d projects, got %d", len(projects[1]), len(received))
				}

				for id := range projects[1] {
					if _, exists := received[id]; !exists {
						t.Fatalf("expected project with id %d, but it was not received", id)
					}
					if received[id] != 2 {
						t.Fatalf("expected project with id %d to be received twice, got %d", id, received[id])
					}
				}

				return dataChan, errsChan
			},
		},
		{
			name: "skip nil project",
			fn: func(t *testing.T) (<-chan *Project, <-chan error) {
				projects := getFakeProjects()
				nilProjectId := 5
				projects[1][nilProjectId] = nil // Simulate a nil project
				dataChan, errsChan := fetchProjectByGroup(context.Background(), &FakeGitlabProjects{
					projects: projects,
				}, &Group{id: 1})

				received := map[int]struct{}{}

				dataDone := false
				errsDone := false

				for !dataDone || !errsDone {
					select {
					case project, ok := <-dataChan:
						if !ok {
							dataDone = true
							continue
						}
						received[project.id] = struct{}{}
					case err, ok := <-errsChan:
						if !ok {
							errsDone = true
							continue
						}
						t.Fatalf("unexpected error: %v", err)
					case <-time.After(50 * time.Millisecond):
						t.Fatal("timeout waiting for channels to close")
					}
				}

				if len(received) != (len(projects[1]) - 1) {
					t.Fatalf("expected %d projects, got %d", len(projects[1]), len(received))
				}

				if _, exists := received[nilProjectId]; exists {
					t.Fatalf("expected project with id %d to be skipped, but it was received", nilProjectId)
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
