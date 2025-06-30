package config

import (
	"os"
	"testing"
)

func TestGetInt(t *testing.T) {
	tests := []struct {
		value        string
		defaultValue int
		expected     int
	}{
		{"123", 42, 123},
		{"", 42, 42},
		{"notanint", 99, 99},
	}

	for _, test := range tests {
		result := getInt(test.value, test.defaultValue)
		if result != test.expected {
			t.Errorf("getInt(%q, %d) = %d; want %d", test.value, test.defaultValue, result, test.expected)
		}
	}
}

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

func TestMemoryEnvLoader(t *testing.T) {
	envs := map[string]string{
		"MEM__STR_KEY": "str_value",
		"MEM__INT_KEY": "123",
		"MEM__EMPTY":   "",
	}
	loader := NewMemoryEnvLoader(envs)

	// Test Get with existing key
	if v := loader.Get("MEM__STR_KEY", "default"); v != "str_value" {
		t.Errorf("Expected str_value, got %s", v)
	}

	// Test Get with missing key
	if v := loader.Get("MEM__MISSING", "default"); v != "default" {
		t.Errorf("Expected default, got %s", v)
	}

	// Test Get with empty value
	if v := loader.Get("MEM__EMPTY", "default"); v != "default" {
		t.Errorf("Expected default, got %s", v)
	}

	// Test GetInt with valid int
	if v := loader.GetInt("MEM__INT_KEY", 42); v != 123 {
		t.Errorf("Expected 123, got %d", v)
	}

	// Test GetInt with missing key
	if v := loader.GetInt("MEM__MISSING_INT", 42); v != 42 {
		t.Errorf("Expected 42, got %d", v)
	}

	// Test GetInt with invalid int
	envs["MEM__BAD_INT"] = "notanint"
	if v := loader.GetInt("MEM__BAD_INT", 99); v != 99 {
		t.Errorf("Expected 99, got %d", v)
	}
}
