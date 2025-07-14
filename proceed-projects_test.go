package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/artzub/gitlab-repo-extractor/config"
)

type mockCloner struct {
	osWrapper       OSWrapper
	projectCloneErr error
}

func (m *mockCloner) GetOSWrapper() OSWrapper {
	return m.osWrapper
}

func (m *mockCloner) cloneProject(_ context.Context, _ *config.Config, _ *Project) error {
	return m.projectCloneErr
}

func (m *mockCloner) CloneProjectWithRetry(ctx context.Context, cfg *config.Config, project *Project) error {
	return m.cloneProject(ctx, cfg, project)
}

func TestProceedProjects(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T) <-chan *Result
	}{
		{
			name: "error result if output directory is not created",
			fn: func(t *testing.T) <-chan *Result {
				outputDir := config.DefaultOutputDir
				config.GetConfig(config.NewMemoryEnvLoader(map[string]string{}))
				expectedErr := &ErrorOutputDirNotCreated{
					outputDir,
					ErrorPathExistsButNotDir,
				}

				cloner := &mockCloner{
					osWrapper: &mockOSWrapper{
						mkdirErr: ErrorPathExistsButNotDir,
					},
				}

				projectsChan := make(chan *Project)
				go func() {
					defer close(projectsChan)
					projectsChan <- &Project{pathWithNamespace: "project1"}
				}()

				resultChan := proceedProjects(context.Background(), cloner, projectsChan)
				resultDone := false

				for !resultDone {
					select {
					case result, ok := <-resultChan:
						if !ok {
							resultDone = true
							continue
						}
						if result.err == nil || result.err.Error() != expectedErr.Error() {
							t.Fatalf("expected error %v, got %v", expectedErr, result.err)
						}
					case <-time.After(50 * time.Millisecond):
						t.Error("timeout waiting for result")
					}
				}

				return resultChan
			},
		},
		{
			name: "error proxying from clone project process",
			fn: func(t *testing.T) <-chan *Result {
				config.GetConfig(config.NewMemoryEnvLoader(map[string]string{}))
				expectedErr := errors.New("clone error")

				cloner := &mockCloner{
					projectCloneErr: expectedErr,
					osWrapper:       &mockOSWrapper{},
				}

				projectsChan := make(chan *Project)
				go func() {
					defer close(projectsChan)
					projectsChan <- &Project{pathWithNamespace: "project1"}
				}()

				resultChan := proceedProjects(context.Background(), cloner, projectsChan)
				resultDone := false

				for !resultDone {
					select {
					case result, ok := <-resultChan:
						if !ok {
							resultDone = true
							continue
						}
						if result.err == nil || result.err.Error() != expectedErr.Error() {
							t.Fatalf("expected error %v, got %v", expectedErr, result.err)
						}
					case <-time.After(50 * time.Millisecond):
						t.Error("timeout waiting for result")
					}
				}

				return resultChan
			},
		},
		{
			name: "clone projects successfully",
			fn: func(t *testing.T) <-chan *Result {
				config.GetConfig(config.NewMemoryEnvLoader(map[string]string{}))

				cloner := &mockCloner{
					osWrapper: &mockOSWrapper{},
				}

				projects := []*Project{
					{pathWithNamespace: "project1"},
					{pathWithNamespace: "project2"},
					{pathWithNamespace: "project3"},
				}

				projectsChan := make(chan *Project)
				go func() {
					defer close(projectsChan)
					for _, project := range projects {
						projectsChan <- project
					}
				}()

				resultChan := proceedProjects(context.Background(), cloner, projectsChan)
				resultDone := false

				received := map[string]struct{}{}

				for !resultDone {
					select {
					case result, ok := <-resultChan:
						if !ok {
							resultDone = true
							continue
						}
						if result.err != nil {
							t.Fatalf("unexpected error %v", result.err)
						}

						received[result.project.pathWithNamespace] = struct{}{}
					case <-time.After(50 * time.Millisecond):
						t.Error("timeout waiting for result")
					}
				}

				if len(received) != len(projects) {
					t.Fatalf("expected %d projects, got %d", len(projects), len(received))
				}

				for _, project := range projects {
					if _, found := received[project.pathWithNamespace]; !found {
						t.Errorf("project %s was not processed", project.pathWithNamespace)
					}
				}

				return resultChan
			},
		},
		{
			name: "should skip nil projects",
			fn: func(t *testing.T) <-chan *Result {
				config.GetConfig(config.NewMemoryEnvLoader(map[string]string{}))

				cloner := &mockCloner{
					osWrapper: &mockOSWrapper{},
				}

				projects := []*Project{
					{pathWithNamespace: "project1"},
					nil,
					{pathWithNamespace: "project3"},
				}

				projectsChan := make(chan *Project)
				go func() {
					defer close(projectsChan)
					for _, project := range projects {
						projectsChan <- project
					}
				}()

				resultChan := proceedProjects(context.Background(), cloner, projectsChan)
				resultDone := false

				received := map[string]struct{}{}

				for !resultDone {
					select {
					case result, ok := <-resultChan:
						if !ok {
							resultDone = true
							continue
						}
						if result.err != nil {
							t.Fatalf("unexpected error %v", result.err)
						}

						received[result.project.pathWithNamespace] = struct{}{}
					case <-time.After(50 * time.Millisecond):
						t.Error("timeout waiting for result")
					}
				}

				if len(received) != (len(projects) - 1) {
					t.Fatalf("expected %d projects, got %d", len(projects), len(received))
				}

				for _, project := range projects {
					if project == nil {
						continue
					}
					if _, found := received[project.pathWithNamespace]; !found {
						t.Errorf("project %s was not processed", project.pathWithNamespace)
					}
				}

				return resultChan
			},
		},
		{
			name: "should handle context cancellation",
			fn: func(t *testing.T) <-chan *Result {
				config.GetConfig(config.NewMemoryEnvLoader(map[string]string{}))

				cloner := &mockCloner{
					osWrapper: &mockOSWrapper{},
				}

				projectsChan := make(chan *Project)
				go func() {
					defer close(projectsChan)
					projectsChan <- &Project{pathWithNamespace: "project1"}
				}()

				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel the context immediately

				resultChan := proceedProjects(ctx, cloner, projectsChan)

				time.Sleep(50 * time.Millisecond)

				select {
				case _, ok := <-resultChan:
					if ok {
						t.Fatal("expected no data to be sent")
					}
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for error")
				}

				return resultChan
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultChan := test.fn(t)

			select {
			case _, ok := <-resultChan:
				if ok {
					t.Fatal("channel should be closed")
				}
			case <-time.After(50 * time.Millisecond):
				t.Fatal("timeout waiting for channel to close")
			}
		})
	}
}
