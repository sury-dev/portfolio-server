package config

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

const loggingSection = "LOGGING"

const (
	defaultLogLevel  = "debug"
	defaultLogFormat = "text"
	defaultLogOutput = "stdout"
)

func loadLoggingConfig(section *ini.Section) (LoggingConfig, error) {
	level, err := ResolveString(loggingSection, "LEVEL", configKey(section, "LEVEL"), defaultLogLevel)
	if err != nil {
		return LoggingConfig{}, err
	}
	format, err := ResolveString(loggingSection, "FORMAT", configKey(section, "FORMAT"), defaultLogFormat)
	if err != nil {
		return LoggingConfig{}, err
	}
	output, err := ResolveString(loggingSection, "OUTPUT", configKey(section, "OUTPUT"), defaultLogOutput)
	if err != nil {
		return LoggingConfig{}, err
	}

	loggingConfig := LoggingConfig{
		Level:  canonicalizeLogLevel(level),
		Format: strings.ToLower(format),
		Output: canonicalizeLogOutput(output),
	}
	if err := loggingConfig.validate(); err != nil {
		return LoggingConfig{}, err
	}
	return loggingConfig, nil
}

func canonicalizeLogLevel(level string) string {
	level = strings.ToLower(level)
	if level == "warn" {
		return "warning"
	}
	return level
}

func canonicalizeLogOutput(output string) string {
	switch strings.ToLower(output) {
	case "stdout", "stderr":
		return strings.ToLower(output)
	default:
		return output
	}
}

func (c LoggingConfig) validate() error {
	switch c.Level {
	case "debug", "info", "warning", "error":
	default:
		return fmt.Errorf("LOGGING.LEVEL must be one of debug, info, warning, error, got %q", c.Level)
	}

	switch c.Format {
	case "json", "text":
	default:
		return fmt.Errorf("LOGGING.FORMAT must be one of json, text, got %q", c.Format)
	}

	switch c.Output {
	case "stdout", "stderr":
		return nil
	default:
		if strings.ContainsRune(c.Output, 0) {
			return fmt.Errorf("LOGGING.OUTPUT must be stdout, stderr, or a file path, got %q", c.Output)
		}
		return nil
	}
}
