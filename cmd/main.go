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

// cmd/main.go - main entry point for agent.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	gosightagent "github.com/aaronlmathis/gosight-agent/internal/agent"
	"github.com/aaronlmathis/gosight-agent/internal/bootstrap"
	"github.com/aaronlmathis/gosight-shared/utils"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "none"
)

func run(configFlag *string) {

	// Bootstrap config loading (flags -> env -> file)
	cfg := bootstrap.LoadAgentConfig(configFlag)
	fmt.Printf("About to init logger with level = %s\n", cfg.Logs.LogLevel)

	bootstrap.SetupLogging(cfg)
	utils.Debug("debug logging is active from main.go")

	// Graceful shutdown context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		utils.Warn("signal received, shutting down agent...")
		cancel()
	}()

	// Create Agent
	agent, err := gosightagent.NewAgent(ctx, cfg, Version)
	if err != nil {
		utils.Error("failed to initialize agent: %v", err)
		os.Exit(1)
	}

	// Start Agent
	agent.Start(ctx)

	<-ctx.Done()

	utils.Info("Context canceled, beginning agent shutdown...")

	agent.Close()
}

// main is the entry point for the GoSight server.
func main() {
	versionFlag := flag.Bool("version", false, "print version information and exit")
	configFlag := flag.String("config", "", "Path to server config file")
	flag.Parse()
	if *versionFlag {
		fmt.Printf(
			"GoSight %s (built %s, commit %s)\n",
			Version, BuildTime, GitCommit,
		)
		os.Exit(0)
	}
	run(configFlag)
}
