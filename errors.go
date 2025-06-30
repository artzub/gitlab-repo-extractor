package main

import (
	"errors"
)

type ErrorGroupNotFound string

func (e ErrorGroupNotFound) Error() string {
	return "group not found: " + string(e)
}

var (
	ErrorNoGroupIDs         = errors.New("no group IDs provided")
	ErrorAllGroupIDsSkipped = errors.New("all group IDs are skipped")
)
