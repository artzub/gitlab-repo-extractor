package main

import (
	"context"
	"os"
	"os/exec"
)

type OSWrapper interface {
	MakeDirAll(path string) error
	IsDirExists(path string) (bool, error)
	RemoveAll(path string) error
	ExecuteCommand(ctx context.Context, cmd string, args ...string) ([]byte, error)
}

type DefaultOSWrapper struct{}

func (w *DefaultOSWrapper) MakeDirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (w *DefaultOSWrapper) IsDirExists(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		if stat.IsDir() {
			return true, nil
		}

		return false, ErrorPathExistsButNotDir
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func (w *DefaultOSWrapper) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (w *DefaultOSWrapper) ExecuteCommand(ctx context.Context, cmd string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, cmd, args...)
	return command.CombinedOutput()
}

var defaultOSWrapper OSWrapper = &DefaultOSWrapper{}

func GetDefaultOSWrapper() OSWrapper {
	return defaultOSWrapper
}
