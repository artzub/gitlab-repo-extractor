package main

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorGroupFetching is an error type that indicates a failure to fetch a group.
type ErrorGroupFetching struct {
	groupID       string
	originalError error
}

func (e *ErrorGroupFetching) Error() string {
	return fmt.Sprintf("failed to fetch group %s: %v", e.groupID, e.originalError)
}

func (e *ErrorGroupFetching) IsGroupNotFound() bool {
	if errors.Is(e.originalError, ErrorNoGroupPassed) {
		return true
	}

	return e.originalError != nil && strings.Contains(e.originalError.Error(), "not found")
}

// ErrorGroupsFetching is an error type that indicates a failure to fetch all groups.
type ErrorGroupsFetching struct {
	originalError error
}

func (e *ErrorGroupsFetching) Error() string {
	return fmt.Sprintf("failed to fetch groups: %v", e.originalError)
}

// ErrorDirExists is an error type that indicates a directory already exists.
type ErrorDirExists string

func (e ErrorDirExists) Error() string {
	return "directory already exists: " + string(e)
}

// ErrorDirExistsCheck is an error type that indicates a failure to check if a directory exists.
type ErrorDirExistsCheck struct {
	dir           string
	originalError error
}

func (e *ErrorDirExistsCheck) Error() string {
	return fmt.Sprintf("failed to check if directory exists (%s): %v", e.dir, e.originalError)
}

// ErrorOutputDirNotCreated is an error type that indicates a failure to create an output directory.
type ErrorOutputDirNotCreated struct {
	dir           string
	originalError error
}

func (e *ErrorOutputDirNotCreated) Error() string {
	return fmt.Sprintf("output directory %s not created: %v", e.dir, e.originalError)
}

// ErrorFailedAfterRetries is an error type that indicates a failure after multiple retries.
type ErrorFailedAfterRetries struct {
	maxRetries int
	lastError  error
}

func (e *ErrorFailedAfterRetries) Error() string {
	return fmt.Sprintf("failed after %d attempts: %v", e.maxRetries, e.lastError)
}

// ErrorFailedToCloneProject is an error type that indicates a failure to clone a project.
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
