package main

import (
	"context"
	"os"
	"os/exec"
)

type OSWrapper interface {
	MakeDirAll(path string) (bool, error)
	IsDirExists(path string) (bool, error)
	RemoveAll(path string) error
	ExecuteCommand(ctx context.Context, cmd string, args ...string) ([]byte, error)
}

type DefaultOSWrapper struct{}

func (w *DefaultOSWrapper) MakeDirAll(path string) (bool, error) {
	err := os.MkdirAll(path, 0o755)

	if err != nil && os.IsExist(err) {
		return true, nil
	}

	return err == nil, err
}

func (w *DefaultOSWrapper) IsDirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
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
