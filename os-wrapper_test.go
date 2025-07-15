package main

import (
	"context"
	"os"
	"path"
	"testing"
)

var dirName = path.Join(os.TempDir(), "test_dir_name")

func setup() {
	_ = os.RemoveAll(dirName)
}

func shutdown() {
	_ = os.RemoveAll(dirName)
}

func TestMain(m *testing.M) {
	defer shutdown()
	setup()
	code := m.Run()
	os.Exit(code)
}

func TestGetDefaultOSWrapper(t *testing.T) {
	w := GetDefaultOSWrapper()
	if w == nil {
		t.Fatal("expected non-nil OSWrapper")
	}

	// Check if the returned type is DefaultOSWrapper
	if _, ok := w.(*DefaultOSWrapper); !ok {
		t.Fatal("expected OSWrapper to be of type DefaultOSWrapper")
	}
}

func TestDefaultOSWrapper_MakeDirAll(t *testing.T) {
	w := GetDefaultOSWrapper()

	dir := path.Join(dirName, "test_make_dir_all")

	err := w.MakeDirAll(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = w.MakeDirAll(dir)
	if err != nil {
		t.Fatalf("unexpected error on attempt to make existing dir: %v", err)
	}
}

func TestDefaultOSWrapper_IsDirExists(t *testing.T) {
	w := GetDefaultOSWrapper()

	dir := path.Join(dirName, "test_exists")

	exists, err := w.IsDirExists(dir)
	if exists {
		t.Fatal("expected directory to not exist")
	}
	if err != nil {
		t.Fatalf("unexpected error, %v", err)
	}

	_ = os.MkdirAll(dir, 0o755)
	exists, err = w.IsDirExists(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected directory to exist")
	}

	_ = os.RemoveAll(dir)

	file, err := os.Create(dir)
	if err != nil {
		t.Fatalf("unexpected error creating file: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	exists, err = w.IsDirExists(dir)
	if err == nil {
		t.Errorf("expected error: %v, got nil", ErrorPathExistsButNotDir)
	}
	if exists {
		t.Error("expected directory to not exist as a directory")
	}
}

func TestDefaultOSWrapper_RemoveAll(t *testing.T) {
	w := GetDefaultOSWrapper()

	dir := path.Join(dirName, "test_remove_all")
	_ = os.MkdirAll(dir, 0o755)

	err := w.RemoveAll(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err = os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected directory to be removed")
	}
}

func TestDefaultOSWrapper_ExecuteCommand(t *testing.T) {
	w := GetDefaultOSWrapper()

	ctx := context.Background()

	// Use a cross-platform command
	cmd := "echo"
	args := []string{"hello"}
	if os.PathSeparator == '\\' {
		cmd = "cmd"
		args = []string{"/C", "echo hello"}
	}

	out, err := w.ExecuteCommand(ctx, cmd, args...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) == "" {
		t.Errorf("expected output, got empty string")
	}
}
