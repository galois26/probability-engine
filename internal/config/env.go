package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// envOrDefault returns the value of the environment variable name, or def if unset.
func envOrDefault(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// envDuration parses an environment variable as a time.Duration.
// Returns def if the variable is unset, or an error if the value is malformed.
func envDuration(name string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(name)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("env %s: invalid duration %q", name, v)
	}
	return d, nil
}

// envInt parses an environment variable as an int.
// Returns def if the variable is unset, or an error if the value is malformed.
func envInt(name string, def int) (int, error) {
	v := os.Getenv(name)
	if v == "" {
		return def, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("env %s: invalid int %q", name, v)
	}
	return i, nil
}

// envBool parses an environment variable as a bool.
// Returns def if the variable is unset, or an error if the value is malformed.
func envBool(name string, def bool) (bool, error) {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if v == "" {
		return def, nil
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("env %s: invalid bool %q", name, v)
	}
}
