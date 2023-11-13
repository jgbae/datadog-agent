// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020-present Datadog, Inc.

// Package serverimpl implements the traps server.
package serverimpl

import (
	"context"

	"github.com/DataDog/datadog-agent/comp/aggregator/demultiplexer"
	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/core/hostname"
	"github.com/DataDog/datadog-agent/comp/core/log"

	trapsconfig "github.com/DataDog/datadog-agent/comp/snmptraps/config"
	"github.com/DataDog/datadog-agent/comp/snmptraps/config/configimpl"
	"github.com/DataDog/datadog-agent/comp/snmptraps/formatter/formatterimpl"
	"github.com/DataDog/datadog-agent/comp/snmptraps/forwarder"
	"github.com/DataDog/datadog-agent/comp/snmptraps/forwarder/forwarderimpl"
	"github.com/DataDog/datadog-agent/comp/snmptraps/listener"
	"github.com/DataDog/datadog-agent/comp/snmptraps/listener/listenerimpl"
	"github.com/DataDog/datadog-agent/comp/snmptraps/oidresolver/oidresolverimpl"
	"github.com/DataDog/datadog-agent/comp/snmptraps/server"
	"github.com/DataDog/datadog-agent/comp/snmptraps/status"
	"github.com/DataDog/datadog-agent/comp/snmptraps/status/statusimpl"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
	"go.uber.org/fx"
)

// Module defines the fx options for this component.
var Module = fxutil.Component(
	fx.Provide(newServer),
)

type dependencies struct {
	fx.In
	Conf      config.Component
	HNService hostname.Component
	Demux     demultiplexer.Component
	Logger    log.Component
}

type injections struct {
	fx.Out
	Conf      config.Component
	HNService hostname.Component
	Demux     demultiplexer.Component
	Logger    log.Component
}

// TrapsServer implements the SNMP traps service.
type TrapsServer struct {
	app     *fx.App
	running bool
	stat    status.Component
}

// Running indicates whether the traps server is currently running.
func (w *TrapsServer) Running() bool {
	return w.running
}

// Error reports any error from server initialization/startup.
func (w *TrapsServer) Error() error {
	if w.stat == nil {
		return nil
	}
	return w.stat.GetStartError()
}

// newServer creates a new traps server, registering it with the fx lifecycle
// system if traps are enabled.
func newServer(lc fx.Lifecycle, deps dependencies) server.Component {
	if !trapsconfig.IsEnabled(deps.Conf) {
		return &TrapsServer{running: false}
	}
	stat := statusimpl.New()
	app := fx.New(
		fx.Supply(injections{
			Conf:      deps.Conf,
			HNService: deps.HNService,
			Demux:     deps.Demux,
			Logger:    deps.Logger,
		}),
		configimpl.Module,
		formatterimpl.Module,
		forwarderimpl.Module,
		listenerimpl.Module,
		oidresolverimpl.Module,
		fx.Provide(stat),
		fx.Invoke(func(_ forwarder.Component, _ listener.Component) {}),
	)
	server := &TrapsServer{app: app, stat: stat}
	if err := app.Err(); err != nil {
		deps.Logger.Errorf("Failed to initialize snmp-traps server: %s", err)
		server.stat.SetStartError(err)
		return server
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err := app.Start(ctx)
			if err != nil {
				server.stat.SetStartError(err)
			} else {
				server.running = true
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			server.running = false
			return app.Stop(ctx)
		},
	})
	return server
}
