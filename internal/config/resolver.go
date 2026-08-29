package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// configKey returns the key only when the config file actually defines it.
// Section.Key creates an empty key on demand, which would make a missing key
// indistinguishable from one that was explicitly set to an empty value.
func configKey(section *ini.Section, key string) *ini.Key {
	if section == nil || !section.HasKey(key) {
		return nil
	}
	return section.Key(key)
}

func envName(section, key string) string {
	return fmt.Sprintf("%s_%s", strings.ToUpper(section), strings.ToUpper(key))
}

// lookup returns the raw value for a key plus a label describing where it came
// from. The environment variable wins over the config file. found is false only
// when neither supplies the key, which is the one case where a default applies:
// a value that was supplied but cannot be used is an error, never a fallback.
func lookup(section, key string, fileKey *ini.Key) (raw, source string, found bool) {
	envVar := envName(section, key)
	if value, ok := os.LookupEnv(envVar); ok {
		return strings.TrimSpace(value), "environment variable " + envVar, true
	}
	if fileKey != nil {
		return strings.TrimSpace(fileKey.String()), fmt.Sprintf("config key %s.%s", strings.ToUpper(section), strings.ToUpper(key)), true
	}
	return "", "", false
}

// ResolveString resolves a string value.
// Precedence is environment variable, then config file, then defaultValue.
// A key that is present but empty is rejected instead of defaulted.
func ResolveString(section, key string, fileKey *ini.Key, defaultValue string) (string, error) {
	raw, source, found := lookup(section, key, fileKey)
	if !found {
		return defaultValue, nil
	}
	if raw == "" {
		return "", fmt.Errorf("%s is set but empty", source)
	}
	return raw, nil
}

// ResolveInt resolves an int value.
// Precedence is environment variable, then config file, then defaultValue.
// A key that is present but empty or not an integer is rejected.
func ResolveInt(section, key string, fileKey *ini.Key, defaultValue int) (int, error) {
	raw, source, found := lookup(section, key, fileKey)
	if !found {
		return defaultValue, nil
	}
	if raw == "" {
		return 0, fmt.Errorf("%s is set but empty", source)
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", source, raw)
	}
	return value, nil
}

// ResolveSeconds resolves a whole number of seconds as a time.Duration, keeping
// the unit in the configuration key name and out of the application code.
// Precedence is environment variable, then config file, then defaultValue.
func ResolveSeconds(section, key string, fileKey *ini.Key, defaultValue time.Duration) (time.Duration, error) {
	seconds, err := ResolveInt(section, key, fileKey, int(defaultValue/time.Second))
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds) * time.Second, nil
}

// ResolveDuration resolves a Go duration string (e.g. 15m, 168h) or a plain
// integer number of seconds. Precedence is environment, then file, then default.
func ResolveDuration(section, key string, fileKey *ini.Key, defaultValue time.Duration) (time.Duration, error) {
	raw, source, found := lookup(section, key, fileKey)
	if !found {
		return defaultValue, nil
	}
	if raw == "" {
		return 0, fmt.Errorf("%s is set but empty", source)
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("%s must be non-negative, got %q", source, raw)
		}
		return time.Duration(seconds) * time.Second, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration (e.g. 15m, 168h) or seconds, got %q", source, raw)
	}
	if value < 0 {
		return 0, fmt.Errorf("%s must be non-negative, got %q", source, raw)
	}
	return value, nil
}

// ResolveBool resolves a bool value.
// Precedence is environment variable, then config file, then defaultValue.
// A key that is present but empty or not a boolean is rejected.
func ResolveBool(section, key string, fileKey *ini.Key, defaultValue bool) (bool, error) {
	raw, source, found := lookup(section, key, fileKey)
	if !found {
		return defaultValue, nil
	}
	if raw == "" {
		return false, fmt.Errorf("%s is set but empty", source)
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean, got %q", source, raw)
	}
}
