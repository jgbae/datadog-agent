// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package settings

import (
	"testing"

	global "github.com/DataDog/datadog-agent/cmd/agent/dogstatsd"
	"github.com/DataDog/datadog-agent/comp/core"
	"github.com/DataDog/datadog-agent/comp/dogstatsd"
	"github.com/DataDog/datadog-agent/comp/dogstatsd/server"
	"github.com/DataDog/datadog-agent/comp/dogstatsd/serverDebug"
	"github.com/DataDog/datadog-agent/comp/forwarder/defaultforwarder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

type testDeps struct {
	fx.In
	Server         server.Component
	Debug          serverDebug.Component
	AggregatorDeps aggregator.TestDeps
}

func TestDogstatsdMetricsStats(t *testing.T) {
	assert := assert.New(t)
	var err error

	opts := aggregator.DefaultAgentDemultiplexerOptions()
	opts.DontStartForwarders = true
	deps := fxutil.Test[testDeps](t, fx.Options(
		core.MockBundle,
		fx.Supply(core.BundleParams{}),
		fx.Supply(server.Params{
			Serverless: false,
		}),
		dogstatsd.Bundle,
		defaultforwarder.MockModule,
	))
	demux := aggregator.InitAndStartAgentDemultiplexerForTest(deps.AggregatorDeps, opts, "hostname")

	global.DSD = deps.Server
	deps.Server.Start(demux)

	require.Nil(t, err)

	s := DsdStatsRuntimeSetting{
		ServerDebug: deps.Debug,
	}

	// runtime settings set/get underlying implementation

	// true string

	err = s.Set("true")
	assert.Nil(err)
	assert.Equal(deps.Debug.IsDebugEnabled(), true)
	v, err := s.Get()
	assert.Nil(err)
	assert.Equal(v, true)

	// false string

	err = s.Set("false")
	assert.Nil(err)
	assert.Equal(deps.Debug.IsDebugEnabled(), false)
	v, err = s.Get()
	assert.Nil(err)
	assert.Equal(v, false)

	// true boolean

	err = s.Set(true)
	assert.Nil(err)
	assert.Equal(deps.Debug.IsDebugEnabled(), true)
	v, err = s.Get()
	assert.Nil(err)
	assert.Equal(v, true)

	// false boolean

	err = s.Set(false)
	assert.Nil(err)
	assert.Equal(deps.Debug.IsDebugEnabled(), false)
	v, err = s.Get()
	assert.Nil(err)
	assert.Equal(v, false)

	// ensure the getter uses the value from the actual server

	deps.Debug.SetMetricStatsEnabled(true)
	v, err = s.Get()
	assert.Nil(err)
	assert.Equal(v, true)
}
