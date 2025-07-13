package main

import (
	"errors"
	"testing"
)

func TestErrorGroupFetching_Error(t *testing.T) {
	err := &ErrorGroupFetching{"123", errors.New("not found")}
	want := "failed to fetch group 123: not found"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorGroupFetching_IsGroupNotFound(t *testing.T) {
	err := &ErrorGroupFetching{"123", errors.New("not found")}
	if !err.IsGroupNotFound() {
		t.Error("expected IsGroupNotFound to be true")
	}

	noGroupErr := &ErrorGroupFetching{"123", ErrorNoGroupPassed}
	if !noGroupErr.IsGroupNotFound() {
		t.Error("expected IsGroupNotFound to be true for ErrorNoGroupPassed")
	}
}

func TestErrorGroupsFetching_Error(t *testing.T) {
	err := &ErrorGroupsFetching{errors.New("fail")}
	want := "failed to fetch groups: fail"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorDirExists_Error(t *testing.T) {
	err := ErrorDirExists("repo")
	want := "directory already exists: repo"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorDirExistsCheck_Error(t *testing.T) {
	err := &ErrorDirExistsCheck{"repo", errors.New("fail")}
	want := "failed to check if directory exists (repo): fail"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorOutputDirNotCreated_Error(t *testing.T) {
	err := &ErrorOutputDirNotCreated{"repo", errors.New("fail")}
	want := "output directory repo not created: fail"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorFailedAfterRetries_Error(t *testing.T) {
	err := &ErrorFailedAfterRetries{3, errors.New("fail")}
	want := "failed after 3 attempts: fail"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorFailedToCloneProject_Error(t *testing.T) {
	err := &ErrorFailedToCloneProject{
		"repo",
		errors.New("fail"),
		[]byte("output"),
	}
	want := "failed to clone project (repo): fail\nOutput:\noutput"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorVars(t *testing.T) {
	if ErrorNoGroupIDs.Error() != "no group IDs provided" {
		t.Error("ErrorNoGroupIDs string mismatch")
	}
	if ErrorAllGroupIDsSkipped.Error() != "all group IDs are skipped" {
		t.Error("ErrorAllGroupIDsSkipped string mismatch")
	}
	if ErrorNoGroupPassed.Error() != "no group passed" {
		t.Error("ErrorNoGroupPassed string mismatch")
	}
	if ErrorNoConfigPassed.Error() != "no configuration passed" {
		t.Error("ErrorNoConfigPassed string mismatch")
	}
	if ErrorNoProjectsPassed.Error() != "no projects passed" {
		t.Error("ErrorNoProjectsPassed string mismatch")
	}
}
