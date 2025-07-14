package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/artzub/gitlab-repo-extractor/config"
)

func TestProceedGroups(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T) (<-chan *Project, <-chan error)
	}{
		{
			name: "Ignore nil group",
			fn: func(t *testing.T) (<-chan *Project, <-chan error) {
				config.GetConfig(config.NewMemoryEnvLoader(map[string]string{}))

				groupsChan := make(chan *Group)
				go func() {
					defer close(groupsChan)

					groupsChan <- nil
				}()

				dataChan, errsChan := proceedGroups(context.Background(), &FakeGitlabProjects{}, groupsChan)

				select {
				case project, ok := <-dataChan:
					if ok {
						t.Errorf("Expected no projects, got %v", project)
					}
				case err := <-errsChan:
					if err != nil {
						t.Errorf("Expected no error, got %v", err)
					}
				}

				return dataChan, errsChan
			},
		},
		{
			name: "Correct proxying errors",
			fn: func(t *testing.T) (<-chan *Project, <-chan error) {
				config.GetConfig(config.NewMemoryEnvLoader(map[string]string{}))

				groupsChan := make(chan *Group)
				go func() {
					defer close(groupsChan)

					groupsChan <- &Group{id: 1}
				}()

				fetchErr := errors.New("fetch error")
				expectedErr := &ErrorProjectsFetching{1, fetchErr}
				fakeProjects := &FakeGitlabProjects{
					fetchErr: fetchErr,
				}

				dataChan, errsChan := proceedGroups(context.Background(), fakeProjects, groupsChan)

				select {
				case project, ok := <-dataChan:
					if ok {
						t.Errorf("Expected no projects, got %v", project)
					}
				case err := <-errsChan:
					if err == nil || err.Error() != expectedErr.Error() {
						t.Errorf("Expected error: %v, got %v", expectedErr, err)
					}
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

				groupsChan := make(chan *Group)
				go func() {
					defer close(groupsChan)

					groupsChan <- &Group{id: 1}
				}()

				wg := &sync.WaitGroup{}
				wg.Add(1)

				go func() {
					defer wg.Done()

					dataChan, errsChan = proceedGroups(ctx, &FakeGitlabProjects{
						projects: projects,
					}, groupsChan)
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
				groupsChan := make(chan *Group)
				go func() {
					defer close(groupsChan)

					groupsChan <- &Group{id: 1}
				}()

				dataChan, errsChan := proceedGroups(context.Background(), &FakeGitlabProjects{
					projects: projects,
				}, groupsChan)

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
