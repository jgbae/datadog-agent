// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package metrics

import (
	"os"
	"strings"
	"time"

	dogstatsdServer "github.com/DataDog/datadog-agent/comp/dogstatsd/server"
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/config/resolver"
	"github.com/DataDog/datadog-agent/pkg/config/utils"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// ServerlessMetricAgent represents the DogStatsD server and the aggregator
type ServerlessMetricAgent struct {
	dogStatsDServer dogstatsdServer.Component
	tags            []string
	Demux           aggregator.Demultiplexer

	SketchesBucketOffset time.Duration
}

// MetricConfig abstacts the config package
type MetricConfig struct {
}

// MetricDogStatsD abstracts the DogStatsD package
type MetricDogStatsD struct {
}

// MultipleEndpointConfig abstracts the config package
type MultipleEndpointConfig interface {
	GetMultipleEndpoints() (map[string][]string, error)
}

// DogStatsDFactory allows create a new DogStatsD server
type DogStatsDFactory interface {
	NewServer(aggregator.Demultiplexer) (dogstatsdServer.Component, error)
}

const (
	statsDMetricBlocklistKey = "statsd_metric_blocklist"
	proxyEnabledEnvVar       = "DD_EXPERIMENTAL_ENABLE_PROXY"
)

// GetMultipleEndpoints returns the api keys per domain specified in the main agent config
func (m *MetricConfig) GetMultipleEndpoints() (map[string][]string, error) {
	return utils.GetMultipleEndpoints(config.Datadog)
}

// NewServer returns a running DogStatsD server
func (m *MetricDogStatsD) NewServer(demux aggregator.Demultiplexer) (dogstatsdServer.Component, error) {
	s := dogstatsdServer.NewServerlessServer()
	return s, s.Start(demux)
}

// Start starts the DogStatsD agent
func (c *ServerlessMetricAgent) Start(forwarderTimeout time.Duration, multipleEndpointConfig MultipleEndpointConfig, dogstatFactory DogStatsDFactory) {
	// prevents any UDP packets from being stuck in the buffer and not parsed during the current invocation
	// by setting this option to 1ms, all packets received will directly be sent to the parser
	config.Datadog.Set("dogstatsd_packet_buffer_flush_timeout", 1*time.Millisecond)

	// the invocation metric is also generated by Lambda Layers
	// we want to avoid duplicate metric
	customerList := config.Datadog.GetStringSlice(statsDMetricBlocklistKey)

	// if the proxy is enabled we need to also block the errorMetric
	if strings.ToLower(os.Getenv(proxyEnabledEnvVar)) == "true" {
		config.Datadog.Set(statsDMetricBlocklistKey, buildMetricBlocklistForProxy(customerList))
	} else {
		config.Datadog.Set(statsDMetricBlocklistKey, buildMetricBlocklist(customerList))
	}
	demux := buildDemultiplexer(multipleEndpointConfig, forwarderTimeout)

	if demux != nil {
		statsd, err := dogstatFactory.NewServer(demux)
		if err != nil {
			log.Errorf("Unable to start the DogStatsD server: %s", err)
		} else {
			c.dogStatsDServer = statsd
			c.Demux = demux
		}
	}
}

// IsReady indicates whether or not the DogStatsD server is ready
func (c *ServerlessMetricAgent) IsReady() bool {
	return c.dogStatsDServer != nil
}

// Flush triggers a DogStatsD flush
func (c *ServerlessMetricAgent) Flush() {
	if c.IsReady() {
		c.dogStatsDServer.ServerlessFlush(c.SketchesBucketOffset)
	}
}

// Stop stops the DogStatsD server
func (c *ServerlessMetricAgent) Stop() {
	if c.IsReady() {
		c.dogStatsDServer.Stop()
	}
}

// SetExtraTags sets extra tags on the DogStatsD server
func (c *ServerlessMetricAgent) SetExtraTags(tagArray []string) {
	if c.IsReady() {
		c.tags = tagArray
		c.dogStatsDServer.SetExtraTags(tagArray)
	}
}

// GetExtraTags gets extra tags
func (c *ServerlessMetricAgent) GetExtraTags() []string {
	return c.tags
}

func buildDemultiplexer(multipleEndpointConfig MultipleEndpointConfig, forwarderTimeout time.Duration) aggregator.Demultiplexer {
	log.Debugf("Using a SyncForwarder with a %v timeout", forwarderTimeout)
	keysPerDomain, err := multipleEndpointConfig.GetMultipleEndpoints()
	if err != nil {
		log.Errorf("Misconfiguration of agent endpoints: %s", err)
		return nil
	}
	return aggregator.InitAndStartServerlessDemultiplexer(resolver.NewSingleDomainResolvers(keysPerDomain), forwarderTimeout)
}

func buildMetricBlocklist(userProvidedList []string) []string {
	return append(userProvidedList, invocationsMetric)
}

// Need to account for duplicate metrics when using the proxy.
func buildMetricBlocklistForProxy(userProvidedList []string) []string {
	return append(buildMetricBlocklist(userProvidedList), ErrorsMetric)
}
