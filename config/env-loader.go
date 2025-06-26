package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type EnvLoader interface {
	// Load loads environment variables from .env files.
	Load()
	// Get retrieves the value of the environment variable with the given key.
	// If the variable is not set, it returns the provided default value or an empty string.
	Get(key string, defaultValue ...string) string
	// GetInt retrieves the value of the environment variable with the given key as an integer.
	// If the variable is not set or cannot be converted to an integer, it returns the provided default value.
	GetInt(key string, defaultValue int) int
}

func getInt(value string, defaultValue int) int {
	if value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}

	return defaultValue
}

type dotEnvLoader struct{}

// Load loads environment variables from .env files.
func (d *dotEnvLoader) Load() {
	_ = godotenv.Load(".env.local", ".env")
}

// Get retrieves the value of the environment variable with the given key.
// If multiple default values are provided, it returns the first one.
func (d *dotEnvLoader) Get(key string, defaultValue ...string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return ""
}

func (d *dotEnvLoader) GetInt(key string, defaultValue int) int {
	value := d.Get(key)
	return getInt(value, defaultValue)
}

var DefaultEnvLoader EnvLoader

func init() {
	DefaultEnvLoader = &dotEnvLoader{}
}
