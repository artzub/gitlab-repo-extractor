package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/artzub/gitlab-repo-extractor/config"
)

type mockOSWrapper struct {
	isDirExists bool
	isDirErr    error
	cmdArgs     []string
	cmdOutput   []byte
	cmdErr      error
	removeErr   error
	removedDir  string
	mkdirErr    error
}

func (m *mockOSWrapper) IsDirExists(_ string) (bool, error) {
	return m.isDirExists, m.isDirErr
}

func (m *mockOSWrapper) ExecuteCommand(_ context.Context, name string, args ...string) ([]byte, error) {
	m.cmdArgs = append([]string{name}, args...)
	return m.cmdOutput, m.cmdErr
}

func (m *mockOSWrapper) RemoveAll(path string) error {
	m.removedDir = path
	return m.removeErr
}

func (m *mockOSWrapper) MakeDirAll(_ string) (bool, error) {
	return m.mkdirErr == nil, m.mkdirErr
}

func TestAddTokenToHTTPSURL(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		token    string
		expected string
	}{
		{
			name:     "Valid URL and token",
			url:      "https://gitlab.com/repo.git",
			token:    "mytoken",
			expected: "https://oauth2:mytoken@gitlab.com/repo.git",
		},
		{
			name:     "Valid URL and empty token",
			url:      "https://gitlab.com/repo.git",
			token:    "",
			expected: "https://gitlab.com/repo.git",
		},
		{
			name:     "Http URL",
			url:      "http://gitlab.com/repo.git",
			token:    "",
			expected: "http://gitlab.com/repo.git",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := addTokenToHTTPSURL(testCase.url, testCase.token)
			if result != testCase.expected {
				t.Errorf("expected %s, got %s", testCase.expected, result)
			}
		})
	}
}

func TestGitCloner_NewGitCloner(t *testing.T) {
	osWrapper := &mockOSWrapper{}
	cloner := NewGitCloner(osWrapper)

	if cloner == nil {
		t.Fatal("expected non-nil GitCloner")
	}

	if cloner.osWrapper != osWrapper {
		t.Errorf("expected osWrapper to be %v, got %v", osWrapper, cloner.osWrapper)
	}

	aOSWrapper := GetDefaultOSWrapper()
	if aOSWrapper == nil {
		t.Fatal("expected non-nil default OS wrapper")
	}

	clonerDefault := NewGitCloner()
	if clonerDefault == nil {
		t.Fatal("expected non-nil GitCloner with default OS wrapper")
	}

	if clonerDefault.GetOSWrapper() != aOSWrapper {
		t.Errorf("expected default OS wrapper to be %v, got %v", aOSWrapper, clonerDefault.GetOSWrapper())
	}

	if clonerDefault.GetOSWrapper() == cloner.GetOSWrapper() {
		t.Error("expected different OS wrappers for default and custom GitCloner")
	}
}

func TestGitCloner_cloneProject(t *testing.T) {
	project := &Project{
		httpURLToRepo:     "https://gitlab.com/repo.git",
		sshURLToRepo:      "git://gitlab.com:repo.git",
		pathWithNamespace: "repo",
	}
	emptyCfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
		config.CloneBareKey: "false",
	}))

	testCases := []struct {
		name          string
		project       *Project
		osWrapper     *mockOSWrapper
		cfg           *config.Config
		expectedError error
	}{
		{
			name:          "Passed nil project",
			project:       nil,
			osWrapper:     &mockOSWrapper{},
			cfg:           emptyCfg,
			expectedError: ErrorNoProjectsPassed,
		},
		{
			name:          "Passed nil config",
			project:       project,
			osWrapper:     &mockOSWrapper{},
			expectedError: ErrorNoConfigPassed,
		},
		{
			name:    "Directory already exists",
			project: project,
			osWrapper: &mockOSWrapper{
				isDirExists: true,
			},
			cfg:           emptyCfg,
			expectedError: ErrorDirExists(project.pathWithNamespace),
		},
		{
			name:    "Check directory existing error",
			project: project,
			osWrapper: &mockOSWrapper{
				isDirErr: errors.New("failed"),
			},
			cfg: emptyCfg,
			expectedError: &ErrorDirExistsCheck{
				project.pathWithNamespace,
				errors.New("failed"),
			},
		},
		{
			name:    "Failed to clone project",
			project: project,
			osWrapper: &mockOSWrapper{
				cmdErr:    errors.New("failed to execute command"),
				cmdOutput: []byte("command output"),
			},
			cfg: emptyCfg,
			expectedError: &ErrorFailedToCloneProject{
				project.pathWithNamespace,
				errors.New("failed to execute command"),
				[]byte("command output"),
			},
		},
		{
			name:      "Clone project with https URL",
			project:   project,
			osWrapper: &mockOSWrapper{},
			cfg: config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
				config.GitlabTokenKey: "test-token",
			})),
		},
		{
			name:      "Clone project with ssh URL",
			project:   project,
			osWrapper: &mockOSWrapper{},
			cfg: config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
				config.UseSSHKey: "true",
			})),
		},
		{
			name:      "Clone project with bare clone",
			project:   project,
			osWrapper: &mockOSWrapper{},
			cfg: config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
				config.CloneBareKey: "true",
			})),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cloner := NewGitCloner(testCase.osWrapper)
			err := cloner.cloneProject(context.Background(), testCase.cfg, testCase.project)
			if err != nil {
				if testCase.expectedError != nil {
					if errors.Is(err, testCase.expectedError) {
						return
					}

					var dirExistsErr *ErrorDirExistsCheck
					if errors.As(testCase.expectedError, &dirExistsErr) &&
						errors.As(err, &dirExistsErr) &&
						testCase.expectedError.Error() == err.Error() {
						return
					}

					var cloneErr *ErrorFailedToCloneProject
					if errors.As(testCase.expectedError, &cloneErr) &&
						errors.As(err, &cloneErr) &&
						testCase.expectedError.Error() == err.Error() {
						return
					}

					t.Errorf("expected error '%v', got: '%v'", testCase.expectedError, err)

					return
				}

				t.Fatalf("expected success, got error: %v", err)
			}

			if testCase.osWrapper.cmdArgs[0] != "git" || testCase.osWrapper.cmdArgs[1] != "clone" {
				t.Errorf("expected git clone command, got: %v", testCase.osWrapper.cmdArgs)
			}

			expectedURL := testCase.project.sshURLToRepo
			if !testCase.cfg.GetUseSSH() {
				expectedURL = addTokenToHTTPSURL(testCase.project.httpURLToRepo, testCase.cfg.GetAccessToken())
			}

			cloneBare := testCase.cfg.GetCloneBare()
			offset := 0
			if cloneBare {
				offset = 1

				if testCase.osWrapper.cmdArgs[2] != "--bare" {
					t.Errorf("expected '--bare' flag, got: %s", testCase.osWrapper.cmdArgs[2])
				}
			}

			if testCase.osWrapper.cmdArgs[offset+2] != expectedURL {
				t.Errorf("expected URL %s, got: %s", expectedURL, testCase.osWrapper.cmdArgs[offset+2])
			}

			if testCase.osWrapper.cmdArgs[offset+3] != testCase.project.pathWithNamespace {
				t.Errorf("expected project path %s, got: %s", testCase.project.pathWithNamespace, testCase.osWrapper.cmdArgs[offset+3])
			}
		})
	}
}

func TestGitCloner_CloneProjectWithRetry(t *testing.T) {
	project := &Project{
		httpURLToRepo:     "https://gitlab.com/repo.git",
		sshURLToRepo:      "git://gitlab.com:repo.git",
		pathWithNamespace: "repo",
	}
	emptyCfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{}))

	testCases := []struct {
		name          string
		project       *Project
		osWrapper     *mockOSWrapper
		cfg           *config.Config
		expectedError error
	}{
		{
			name:          "Passed nil config",
			project:       project,
			osWrapper:     &mockOSWrapper{},
			expectedError: ErrorNoConfigPassed,
		},
		{
			name:          "Passed nil project",
			project:       nil,
			osWrapper:     &mockOSWrapper{},
			cfg:           emptyCfg,
			expectedError: ErrorNoProjectsPassed,
		},
		{
			name:    "Failed after retries",
			project: project,
			osWrapper: &mockOSWrapper{
				cmdErr: errors.New("failed to execute command"),
			},
			cfg: config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
				config.MaxRetriesKey: "1",
				config.RetryDelayKey: "0",
			})),
			expectedError: &ErrorFailedAfterRetries{
				1,
				&ErrorFailedToCloneProject{
					project.pathWithNamespace,
					errors.New("failed to execute command"),
					nil,
				},
			},
		},
		{
			name:      "Clone project success",
			project:   project,
			osWrapper: &mockOSWrapper{},
			cfg:       emptyCfg,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cloner := NewGitCloner(testCase.osWrapper)
			err := cloner.CloneProjectWithRetry(context.Background(), testCase.cfg, testCase.project)
			if err == nil {
				return
			}

			if testCase.expectedError == nil {
				t.Fatalf("expected success, got error: %v", err)
			}

			if errors.Is(err, testCase.expectedError) {
				return
			}

			var afterRetiesErr *ErrorFailedAfterRetries
			if errors.As(testCase.expectedError, &afterRetiesErr) &&
				errors.As(err, &afterRetiesErr) &&
				testCase.expectedError.Error() == err.Error() {
				return
			}

			t.Errorf("expected error '%v', got: '%v'", testCase.expectedError, err)
		})
	}
}

func TestGitCloner_CloneProjectWithRetry_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	project := &Project{
		httpURLToRepo:     "https://gitlab.com/repo.git",
		sshURLToRepo:      "git://gitlab.com:repo.git",
		pathWithNamespace: "repo",
	}

	cfg := config.NewConfig(config.NewMemoryEnvLoader(map[string]string{
		config.MaxRetriesKey: "2",
		config.RetryDelayKey: "1",
	}))

	cloner := NewGitCloner(&mockOSWrapper{
		cmdErr: errors.New("failed"),
	})

	err := cloner.CloneProjectWithRetry(ctx, cfg, project)
	if err == nil {
		t.Fatalf("expected context canceled error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context canceled error, got: %v", err)
	}
}
