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

package bootstrap

import (
	"reflect"
	"testing"

	"github.com/aaronlmathis/gosight-agent/internal/config"
	"github.com/aaronlmathis/gosight-shared/utils"
)

func TestSetupLoggingUsesLogsPaths(t *testing.T) {
	var got [5]string
	initLogger = func(app, err, access, debug, level string) error {
		got = [5]string{app, err, access, debug, level}
		return nil
	}
	defer func() { initLogger = utils.InitLogger }()

	cfg := &config.Config{}
	cfg.Logs.AppLogFile = "app.log"
	cfg.Logs.ErrorLogFile = "err.log"
	cfg.Logs.AccessLogFile = "access.log"
	cfg.Logs.DebugLogFile = "debug.log"
	cfg.Logs.LogLevel = "info"

	SetupLogging(cfg)

	want := [5]string{"app.log", "err.log", "access.log", "debug.log", "info"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("initLogger called with %v, want %v", got, want)
	}
}
