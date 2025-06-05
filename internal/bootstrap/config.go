/*
SPDX-License-Identifier: GPL-3.0-or-later

Copyright (C) 2025 Aaron Mathis aaron.mathis@gmail.com

This file is part of GoSight.

GoSight is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

GoSight is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with GoSight. If not, see https://www.gnu.org/licenses/.
*/

// File: agent/internal/bootstrap/config.go
// Loads ENV, FLAG, Configs

package bootstrap

import (
	"log"
	"os"
	"path/filepath"

	"github.com/aaronlmathis/gosight-agent/internal/config"
	"github.com/aaronlmathis/gosight-shared/utils"
)

// LoadAgentConfig loads the agent configuration from a file, environment variables, and command-line flags.
// It applies the overrides in the following order: command-line flags > environment variables > config file.
// The function returns a pointer to the loaded configuration.
func LoadAgentConfig(configFlag *string) *config.Config {

	// Resolve config path
	configPath := resolvePath(*configFlag, "GOSIGHT_AGENT_CONFIG", "./config/config.yaml")
	log.Printf("Loaded config file from: %s", configPath)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	config.ApplyEnvOverrides(cfg)

	return cfg
}

// resolvePath resolves the path for a given flag value, environment variable, and fallback value.
// It checks if the flag value is set, then checks the environment variable,
// and finally falls back to the provided default value.
func resolvePath(flagVal, envVar, fallback string) string {
	if flagVal != "" {
		return absPath(flagVal)
	}
	if val := os.Getenv(envVar); val != "" {
		utils.Debug("Using %s from environment variable: %s", envVar, val)
		return absPath(val)
	}
	return absPath(fallback)
}

// absPath resolves the absolute path of a given path.
// It uses filepath.Abs to get the absolute path and handles any errors that may occur.
func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Failed to resolve path: %v", err)
	}
	return abs
}
