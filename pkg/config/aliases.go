// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"context"

	slog "github.com/cihub/seelog"

	"github.com/DataDog/datadog-agent/comp/core/secrets"
	"github.com/DataDog/datadog-agent/pkg/config/env"
	"github.com/DataDog/datadog-agent/pkg/config/logs"
	"github.com/DataDog/datadog-agent/pkg/config/model"
	pkgconfigsetup "github.com/DataDog/datadog-agent/pkg/config/setup"
	"github.com/DataDog/datadog-agent/pkg/util/optional"
)

// Aliases to conf package
type (
	// Proxy alias to model.Proxy
	Proxy = model.Proxy
	// Reader is alias to model.Reader
	Reader = model.Reader
	// Writer is alias to model.Reader
	Writer = model.Writer
	// ReaderWriter is alias to model.ReaderWriter
	ReaderWriter = model.ReaderWriter
	// Loader is alias to model.Loader
	Loader = model.Loader
	// Config is alias to model.Config
	Config = model.Config
)

// NewConfig is alias for Config object.
var NewConfig = model.NewConfig

// Warnings represent the warnings in the config
type Warnings = model.Warnings

// environment Aliases
var (
	IsFeaturePresent             = env.IsFeaturePresent
	IsECS                        = env.IsECS
	IsKubernetes                 = env.IsKubernetes
	IsECSFargate                 = env.IsECSFargate
	IsServerless                 = env.IsServerless
	IsContainerized              = env.IsContainerized
	IsDockerRuntime              = env.IsDockerRuntime
	GetEnvDefault                = env.GetEnvDefault
	IsHostProcAvailable          = env.IsHostProcAvailable
	IsHostSysAvailable           = env.IsHostSysAvailable
	IsAnyContainerFeaturePresent = env.IsAnyContainerFeaturePresent
	GetDetectedFeatures          = env.GetDetectedFeatures
)

type (
	// Feature Alias
	Feature = env.Feature
	// FeatureMap Alias
	FeatureMap = env.FeatureMap
)

// Aliases for constants
const (
	ECSFargate               = env.ECSFargate
	Podman                   = env.Podman
	Docker                   = env.Docker
	EKSFargate               = env.EKSFargate
	ECSEC2                   = env.ECSEC2
	Kubernetes               = env.Kubernetes
	CloudFoundry             = env.CloudFoundry
	Cri                      = env.Cri
	Containerd               = env.Containerd
	KubeOrchestratorExplorer = env.KubeOrchestratorExplorer
)

// IsAutoconfigEnabled is alias for model.IsAutoconfigEnabled
func IsAutoconfigEnabled() bool {
	return env.IsAutoconfigEnabled(Datadog)
}

// Aliases for config overrides
var (
	AddOverride        = model.AddOverride
	AddOverrides       = model.AddOverrides
	AddOverrideFunc    = model.AddOverrideFunc
	applyOverrideFuncs = model.ApplyOverrideFuncs
)

// LoggerName Alias
type LoggerName = logs.LoggerName

// Aliases for  logs
var (
	NewLogWriter   = logs.NewLogWriter
	ChangeLogLevel = logs.ChangeLogLevel
)

// SetupLogger Alias using Datadog config
func SetupLogger(loggerName LoggerName, logLevel, logFile, syslogURI string, syslogRFC, logToConsole, jsonFormat bool) error {
	return logs.SetupLogger(loggerName, logLevel, logFile, syslogURI, syslogRFC, logToConsole, jsonFormat, Datadog)
}

// SetupJMXLogger Alias using Datadog config
func SetupJMXLogger(logFile, syslogURI string, syslogRFC, logToConsole, jsonFormat bool) error {
	return logs.SetupJMXLogger(logFile, syslogURI, syslogRFC, logToConsole, jsonFormat, Datadog)
}

// GetSyslogURI Alias using Datadog config
func GetSyslogURI() string {
	return logs.GetSyslogURI(Datadog)
}

// SetupDogstatsdLogger Alias using Datadog config
func SetupDogstatsdLogger(logFile string) (slog.LoggerInterface, error) {
	return logs.SetupDogstatsdLogger(logFile, Datadog)
}

// IsCloudProviderEnabled Alias using Datadog config
func IsCloudProviderEnabled(cloudProvider string) bool {
	return pkgconfigsetup.IsCloudProviderEnabled(cloudProvider, Datadog)
}

// GetIPCAddress Alias using Datadog config
func GetIPCAddress() (string, error) {
	return pkgconfigsetup.GetIPCAddress(Datadog)
}

type Endpoint = pkgconfigsetup.Endpoint

const (
	Metrics = pkgconfigsetup.Metrics
	Traces  = pkgconfigsetup.Traces
	Logs    = pkgconfigsetup.Logs
)

const (
	DefaultForwarderRecoveryInterval         = pkgconfigsetup.DefaultForwarderRecoveryInterval
	DefaultAPIKeyValidationInterval          = pkgconfigsetup.DefaultAPIKeyValidationInterval
	DefaultBatchWait                         = pkgconfigsetup.DefaultBatchWait
	DefaultInputChanSize                     = pkgconfigsetup.DefaultInputChanSize
	DefaultBatchMaxConcurrentSend            = pkgconfigsetup.DefaultBatchMaxConcurrentSend
	DefaultBatchMaxContentSize               = pkgconfigsetup.DefaultBatchMaxContentSize
	DefaultLogsSenderBackoffRecoveryInterval = pkgconfigsetup.DefaultLogsSenderBackoffRecoveryInterval
	DefaultLogsSenderBackoffMax              = pkgconfigsetup.DefaultLogsSenderBackoffMax
	DefaultLogsSenderBackoffFactor           = pkgconfigsetup.DefaultLogsSenderBackoffFactor
	DefaultLogsSenderBackoffBase             = pkgconfigsetup.DefaultLogsSenderBackoffBase
	DefaultBatchMaxSize                      = pkgconfigsetup.DefaultBatchMaxSize
	DefaultNumWorkers                        = pkgconfigsetup.DefaultNumWorkers
	MaxNumWorkers                            = pkgconfigsetup.MaxNumWorkers
	DefaultSite                              = pkgconfigsetup.DefaultSite
	OTLPTracePort                            = pkgconfigsetup.OTLPTracePort
	DefaultAuditorTTL                        = pkgconfigsetup.DefaultAuditorTTL
	DefaultMaxMessageSizeBytes               = pkgconfigsetup.DefaultMaxMessageSizeBytes
	DefaultProcessEntityStreamPort           = pkgconfigsetup.DefaultProcessEntityStreamPort
	DefaultProcessAgentLogFile               = pkgconfigsetup.DefaultProcessAgentLogFile
	DefaultProcessEventsCheckInterval        = pkgconfigsetup.DefaultProcessEventsCheckInterval
	DefaultProcessEventsMinCheckInterval     = pkgconfigsetup.DefaultProcessEventsMinCheckInterval
	ProcessMaxPerMessageLimit                = pkgconfigsetup.ProcessMaxPerMessageLimit
	DefaultProcessMaxPerMessage              = pkgconfigsetup.DefaultProcessMaxPerMessage
	ProcessMaxMessageBytesLimit              = pkgconfigsetup.ProcessMaxMessageBytesLimit
	DefaultProcessDiscoveryHintFrequency     = pkgconfigsetup.DefaultProcessDiscoveryHintFrequency
	DefaultProcessMaxMessageBytes            = pkgconfigsetup.DefaultProcessMaxMessageBytes
	DefaultProcessExpVarPort                 = pkgconfigsetup.DefaultProcessExpVarPort
	DefaultProcessQueueBytes                 = pkgconfigsetup.DefaultProcessQueueBytes
	DefaultProcessQueueSize                  = pkgconfigsetup.DefaultProcessQueueSize
	DefaultProcessRTQueueSize                = pkgconfigsetup.DefaultProcessRTQueueSize
)

func GetObsPipelineURL(datatype pkgconfigsetup.DataType) (string, error) {
	return pkgconfigsetup.GetObsPipelineURL(datatype, Datadog)
}

type (
	ConfigurationProviders = pkgconfigsetup.ConfigurationProviders
	Listeners              = pkgconfigsetup.Listeners
	MappingProfile         = pkgconfigsetup.MappingProfile
)

func LoadCustom(config model.Config, origin string, secretResolver optional.Option[secrets.Component], additionalKnownEnvVars []string) (*model.Warnings, error) {
	return pkgconfigsetup.LoadCustom(config, origin, secretResolver, additionalKnownEnvVars)
}

func LoadDatadogCustom(config model.Config, origin string, secretResolver optional.Option[secrets.Component], additionalKnownEnvVars []string) (*model.Warnings, error) {
	return pkgconfigsetup.LoadDatadogCustom(config, origin, secretResolver, additionalKnownEnvVars)
}

func GetValidHostAliases(ctx context.Context) ([]string, error) {
	return pkgconfigsetup.GetValidHostAliases(ctx, Datadog)
}

func IsCLCRunner() bool {
	return pkgconfigsetup.IsCLCRunner(Datadog)
}

func GetBindHostFromConfig(config model.Reader) string {
	return pkgconfigsetup.GetBindHostFromConfig(config)
}

func GetBindHost() string {
	return pkgconfigsetup.GetBindHost(Datadog)
}

func GetDogstatsdMappingProfiles() ([]MappingProfile, error) {
	return pkgconfigsetup.GetDogstatsdMappingProfiles(Datadog)
}

var (
	IsRemoteConfigEnabled   = pkgconfigsetup.IsRemoteConfigEnabled
	StartTime               = pkgconfigsetup.StartTime
	StandardJMXIntegrations = pkgconfigsetup.StandardJMXIntegrations
	SetupOTLP               = pkgconfigsetup.SetupOTLP
	InitSystemProbeConfig   = pkgconfigsetup.InitSystemProbeConfig
)

// LoadWithoutSecret reads configs files, initializes the config module without decrypting any secrets
func LoadWithoutSecret() (*model.Warnings, error) {
	return pkgconfigsetup.LoadDatadogCustom(Datadog, "datadog.yaml", optional.NewNoneOption[secrets.Component](), SystemProbe.GetEnvVars())
}

func GetProcessAPIAddressPort() (string, error) {
	return pkgconfigsetup.GetProcessAPIAddressPort(Datadog)
}
