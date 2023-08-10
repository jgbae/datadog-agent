// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//revive:disable:var-naming

//go:build kubeapiserver

// Package telemetry TODO comment
package telemetry

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/config/utils"
	httputils "github.com/DataDog/datadog-agent/pkg/util/http"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	mainEndpointPrefix = "https://instrumentation-telemetry-intake."
// const mainEndpointUrlKey should be mainEndpointURLKey
	mainEndpointUrlKey = "apm_config.telemetry.dd_url"

	httpClientResetInterval = 5 * time.Minute
	httpClientTimeout       = 10 * time.Second
  // Success exported const should have comment (or a comment on this block) or be unexported
	Success                 = 0
  // ConfigParseFailure exported const should have comment (or a comment on this block) or be unexported
	ConfigParseFailure      = 1
  // InvalidPatchRequest exported const should have comment (or a comment on this block) or be unexported
	InvalidPatchRequest     = 2
  // FailedToMutateConfig exported const should have comment (or a comment on this block) or be unexported
	FailedToMutateConfig    = 3
)

// ApmRemoteConfigEvent is used to report remote config updates to the Datadog backend
type ApmRemoteConfigEvent struct {
	RequestType string                      `json:"request_type"`
// struct field ApiVersion should be APIVersion
	ApiVersion  string                      `json:"api_version"`
	Payload     ApmRemoteConfigEventPayload `json:"payload,omitempty"`
}

// ApmRemoteConfigEventPayload contains the information on an individual remote config event
type ApmRemoteConfigEventPayload struct {
	EventName string                    `json:"event_name"`
	Tags      ApmRemoteConfigEventTags  `json:"tags"`
	Error     ApmRemoteConfigEventError `json:"error,omitempty"`
}

// ApmRemoteConfigEventTags store the information on an individual remote config event
type ApmRemoteConfigEventTags struct {
	Env                 string `json:"env"`
// struct field RcId should be RcID
	RcId                string `json:"rc_id"`
// struct field RcClientId should be RcClientID
	RcClientId          string `json:"rc_client_id"`
	RcRevision          int64  `json:"rc_revision"`
	RcVersion           uint64 `json:"rc_version"`
// struct field KubernetesClusterId should be KubernetesClusterID
	KubernetesClusterId string `json:"k8s_cluster_id"`
	KubernetesCluster   string `json:"k8s_cluster"`
	KubernetesNamespace string `json:"k8s_namespace"`
	KubernetesKind      string `json:"k8s_kind"`
	KubernetesName      string `json:"k8s_name"`
}

// ApmRemoteConfigEventError stores the debugging information about remote config deployment failures
type ApmRemoteConfigEventError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Collector is the interface used to send reports about startup to the instrumentation telemetry intake
type Collector interface {
	SendRemoteConfigPatchEvent(event ApmRemoteConfigEvent)
	SendRemoteConfigMutateEvent(event ApmRemoteConfigEvent)
	SetTestHost(testHost string)
}

type telemetryCollector struct {
	client              *httputils.ResetClient
	host                string
	userAgent           string
// struct field rcClientId should be rcClientID
	rcClientId          string
// struct field kubernetesClusterId should be kubernetesClusterID
	kubernetesClusterId string
}

func httpClientFactory(timeout time.Duration) func() *http.Client {
	return func() *http.Client {
		return &http.Client{
			Timeout: timeout,
			// reusing core agent HTTP transport to benefit from proxy settings.
			Transport: httputils.CreateHTTPTransport(),
		}
	}
}

// NewCollector returns either collector, or a noop implementation if instrumentation telemetry is disabled
// func parameter rcClientId should be rcClientID
func NewCollector(rcClientId string, kubernetesClusterId string) Collector {
	return &telemetryCollector{
		client:              httputils.NewResetClient(httpClientResetInterval, httpClientFactory(httpClientTimeout)),
		host:                utils.GetMainEndpoint(config.Datadog, mainEndpointPrefix, mainEndpointUrlKey),
		userAgent:           "Datadog Cluster Agent",
		rcClientId:          rcClientId,
		kubernetesClusterId: kubernetesClusterId,
	}
}

func (tc *telemetryCollector) SetTestHost(testHost string) {
	tc.host = testHost
}

// NewNoopCollector returns a noop collector
func NewNoopCollector() Collector {
	return &noopTelemetryCollector{}
}

func (tc *telemetryCollector) SendRemoteConfigPatchEvent(event ApmRemoteConfigEvent) {
	tc.sendRemoteConfigEvent("agent.k8s.patch", event)
}

func (tc *telemetryCollector) SendRemoteConfigMutateEvent(event ApmRemoteConfigEvent) {
	tc.sendRemoteConfigEvent("agent.k8s.mutate", event)
}

// getRemoteConfigPatchEvent fills out and sends a telemetry event to the Datadog backend
// to indicate that a remote config has been successfully patched
func (tc *telemetryCollector) sendRemoteConfigEvent(eventName string, event ApmRemoteConfigEvent) {
	event.Payload.Tags.RcClientId = tc.rcClientId
	event.Payload.Tags.KubernetesClusterId = tc.kubernetesClusterId
	event.Payload.EventName = eventName
	body, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Error while trying to marshal a remote config event to JSON: %v", err)
		return
	}
	bodyLen := strconv.Itoa(len(body))

	req, err := http.NewRequest("POST", tc.host+"/api/v2/apmtelemetry", bytes.NewReader(body))
	if err != nil {
		log.Errorf("Error while trying to create a web request for a remote config event: %v", err)
		return
	}
	if !config.Datadog.IsSet("api_key") {
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", tc.userAgent)
	req.Header.Add("DD-API-KEY", config.Datadog.GetString("api_key"))
	req.Header.Add("Content-Length", bodyLen)

	resp, err := tc.client.Do(req)
	if err != nil {
		log.Errorf("Failed to transmit remote config event to Datadog: %v", err)
		return
	}
	// Unconditionally read the body and ignore any errors
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

type noopTelemetryCollector struct{}

func (*noopTelemetryCollector) SendRemoteConfigPatchEvent(event ApmRemoteConfigEvent) {
}

func (*noopTelemetryCollector) SendRemoteConfigMutateEvent(event ApmRemoteConfigEvent) {
}

func (*noopTelemetryCollector) SetTestHost(testHost string) {}
