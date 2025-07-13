package main

import (
	"errors"
	"fmt"
)

type ErrorGroupNotFound string

func (e ErrorGroupNotFound) Error() string {
	return "group not found: " + string(e)
}

type ErrorDirExists string

func (e ErrorDirExists) Error() string {
	return "directory already exists: " + string(e)
}

type ErrorOutputDirNotCreated struct {
	dir           string
	originalError error
}

func (e *ErrorOutputDirNotCreated) Error() string {
	return fmt.Sprintf("output directory %s not created: %v", e.dir, e.originalError)
}

type ErrorFailedToCheckDirExists struct {
	dir           string
	originalError error
}

func (e *ErrorFailedToCheckDirExists) Error() string {
	return fmt.Sprintf("failed to check if directory exists (%s): %v", e.dir, e.originalError)
}

type ErrorFailedAfterRetries struct {
	maxRetries int
	lastError  error
}

func (e *ErrorFailedAfterRetries) Error() string {
	return fmt.Sprintf("failed after %d attempts: %v", e.maxRetries, e.lastError)
}

type ErrorFailedToCloneProject struct {
	projectDir    string
	originalError error
	output        []byte
}

func (e *ErrorFailedToCloneProject) Error() string {
	return fmt.Sprintf("failed to clone project (%s): %v\nOutput:\n%s", e.projectDir, e.originalError, e.output)
}

var (
	ErrorNoGroupIDs         = errors.New("no group IDs provided")
	ErrorAllGroupIDsSkipped = errors.New("all group IDs are skipped")
	ErrorNoGroupPassed      = errors.New("no group passed")
	ErrorNoConfigPassed     = errors.New("no configuration passed")
	ErrorNoProjectsPassed   = errors.New("no projects passed")
)
