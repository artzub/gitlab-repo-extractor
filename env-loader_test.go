package main

import (
	"os"
	"testing"
)

func TestDefaultEnvLoaderGet(t *testing.T) {
	tests := []struct {
		key          string
		defaultValue string
		toSet        string
	}{
		{
			"FAKE__TEST_KEY",
			"default_value",
			"new_value",
		},
		{
			"FAKE__GITLAB_URL",
			"https://gitlab.com",
			"https://gitlab.example.com",
		},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			defer func() {
				_ = os.Unsetenv(test.key)
			}()

			// Ensure the environment variable is unset before the test
			err := os.Unsetenv(test.key)
			if err != nil {
				t.Fatalf("Failed to unset environment variable %s: %v", test.key, err)
			}

			result := DefaultEnvLoader.Get(test.key, test.defaultValue)
			if result != test.defaultValue {
				t.Errorf("Expected %s, got %s for key %s", test.defaultValue, result, test.key)
			}

			// Set the empty value to the environment variable
			err = os.Setenv(test.key, "")
			if err != nil {
				t.Fatalf("Failed to set environment variable %s: %v", test.key, err)
			}
			result = DefaultEnvLoader.Get(test.key, test.defaultValue)
			if result != test.defaultValue {
				t.Errorf("Expected %s, got %s for key %s after setting empty value", test.defaultValue, result, test.key)
			}

			// Set the new value to the environment variable
			err = os.Setenv(test.key, test.toSet)
			if err != nil {
				t.Fatalf("Failed to set environment variable %s: %v", test.key, err)
			}
			result = DefaultEnvLoader.Get(test.key, test.defaultValue)
			if result != test.toSet {
				t.Errorf("Expected %s, got %s for key %s after setting", test.toSet, result, test.key)
			}
		})
	}
}

func TestDefaultEnvLoaderGetInt(t *testing.T) {
	tests := []struct {
		key          string
		defaultValue int
		toSet        string
		expected     int
	}{
		{
			"FAKE__TEST_INT_KEY",
			42,
			"100",
			100,
		},
		{
			"FAKE__GITLAB_MAX_WORKERS",
			5,
			"10",
			10,
		},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			defer func() {
				_ = os.Unsetenv(test.key)
			}()

			// Ensure the environment variable is unset before the test
			err := os.Unsetenv(test.key)
			if err != nil {
				t.Fatalf("Failed to unset environment variable %s: %v", test.key, err)
			}

			result := DefaultEnvLoader.GetInt(test.key, test.defaultValue)
			if result != test.defaultValue {
				t.Errorf("Expected %d, got %d for key %s", test.defaultValue, result, test.key)
			}

			// Set the empty value to the environment variable
			err = os.Setenv(test.key, "")
			if err != nil {
				t.Fatalf("Failed to set environment variable %s: %v", test.key, err)
			}
			result = DefaultEnvLoader.GetInt(test.key, test.defaultValue)
			if result != test.defaultValue {
				t.Errorf("Expected %d, got %d for key %s after setting empty value", test.defaultValue, result, test.key)
			}

			// Set the new value to the environment variable
			err = os.Setenv(test.key, test.toSet)
			if err != nil {
				t.Fatalf("Failed to set environment variable %s: %v", test.key, err)
			}
			result = DefaultEnvLoader.GetInt(test.key, test.defaultValue)
			if result != test.expected {
				t.Errorf("Expected %d, got %d for key %s after setting", test.expected, result, test.key)
			}
		})
	}
}
