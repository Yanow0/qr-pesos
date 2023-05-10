package main

import (
	"os"
	"strings"
)

type Config struct {
	Port           string
	StaticFilesDir string
	Languages      []string
}

func LoadConfig() *Config {
	// Load the configuration options from a configuration file or environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticFilesDir := os.Getenv("STATIC_FILES_DIR")
	if staticFilesDir == "" {
		staticFilesDir = "static"
	}

	languagesStr := os.Getenv("LANGUAGES")
	languages := strings.Split(languagesStr, ",")
	if len(languages) == 0 {
		languages = []string{"en"}
	}

	// Return a Config object with the options
	return &Config{
		Port:           port,
		StaticFilesDir: staticFilesDir,
		Languages:      languages,
	}
}
