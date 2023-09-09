// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build windows

// Package main implements main
package main

import (
	"context"
	"os"
	"path"

	"go.uber.org/fx"

	commonpath "github.com/DataDog/datadog-agent/cmd/agent/common/path"
	"github.com/DataDog/datadog-agent/cmd/agent/common/signals"
	"github.com/DataDog/datadog-agent/cmd/internal/runcmd"
	"github.com/DataDog/datadog-agent/cmd/security-agent/command"
	saconfig "github.com/DataDog/datadog-agent/cmd/security-agent/config"
	"github.com/DataDog/datadog-agent/cmd/security-agent/subcommands"
	"github.com/DataDog/datadog-agent/cmd/security-agent/subcommands/start"
	"github.com/DataDog/datadog-agent/comp/core"
	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/core/log"
	"github.com/DataDog/datadog-agent/comp/core/telemetry"
	"github.com/DataDog/datadog-agent/comp/forwarder"
	"github.com/DataDog/datadog-agent/comp/forwarder/defaultforwarder"

	pkgconfig "github.com/DataDog/datadog-agent/pkg/config"

	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
	"github.com/DataDog/datadog-agent/pkg/util/winutil/servicemain"
)

type service struct {
	errChan chan error
	ctxChan chan context.Context
}

var defaultSecurityAgentConfigFilePaths = []string{
	path.Join(commonpath.DefaultConfPath, "datadog.yaml"),
	path.Join(commonpath.DefaultConfPath, "security-agent.yaml"),
}

func (s *service) Name() string {
	return saconfig.ServiceName
}

func (s *service) Init() error {
	s.ctxChan = make(chan context.Context)

	s.errChan = make(chan error)

	return nil
}

func (s *service) Run(svcctx context.Context) error {

	// run startSystemProbe in an app, so that the log and config components get initialized
	err := fxutil.OneShot(
		func(log log.Component, config config.Component, telemetry telemetry.Component, forwarder defaultforwarder.Component) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer start.StopAgent(cancel, log)

			err := start.RunAgent(ctx, log, config, telemetry, forwarder, "")
			// notify outer that startAgent finished
			if err != nil {
				return err
			}

			// Wait for stop signal
			select {
			case <-signals.Stopper:
				log.Info("Received stop command, shutting down...")
			case <-signals.ErrorStopper:
				_ = log.Critical("The Agent has encountered an error, shutting down...")
			case <-svcctx.Done():
				log.Info("Received stop from service manager, shutting down...")
			}

			return nil
		},
		fx.Supply(core.BundleParams{
			ConfigParams: config.NewSecurityAgentParams(defaultSecurityAgentConfigFilePaths),
			LogParams:    log.LogForDaemon(command.LoggerName, "security_agent.log_file", pkgconfig.DefaultSecurityAgentLogFile),
		}),
		core.Bundle,
		forwarder.Bundle,
		fx.Provide(defaultforwarder.NewParamsWithResolvers),
	)

	// startSystemProbe succeeded. provide errChan to caller so they can wait for fxutil.OneShot to stop
	return err
}

func main() {
	// if command line arguments are supplied, even in a non-interactive session,
	// then just execute that.  Used when the service is executing the executable,
	// for instance to trigger a restart.
	if len(os.Args) == 1 {
		if servicemain.RunningAsWindowsService() {
			servicemain.Run(&service{})
			return
		}
	}

	rootCmd := command.MakeCommand(subcommands.SecurityAgentSubcommands())
	os.Exit(runcmd.Run(rootCmd))
}
