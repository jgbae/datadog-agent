// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"errors"
	"fmt"
	"io/fs"
	"runtime"
	"strings"

	"github.com/DataDog/viper"

	pkgconfigmodel "github.com/DataDog/datadog-agent/pkg/config/model"
	pkgconfigsetup "github.com/DataDog/datadog-agent/pkg/config/setup"
)

// setupConfig is copied from cmd/agent/common/helpers.go.
func setupConfig(deps configDependencies) (pkgconfigmodel.Config, *pkgconfigmodel.Warnings, error) {
	p := deps.getParams()

	confFilePath := p.ConfFilePath
	configName := p.configName
	withoutSecrets := !p.configLoadSecrets
	failOnMissingFile := !p.configMissingOK
	defaultConfPath := p.defaultConfPath

	config := pkgconfigmodel.NewConfig("datadog", "DD", strings.NewReplacer(".", "_"))
	pkgconfigsetup.InitConfig(config)

	systemProbe := pkgconfigmodel.NewConfig("system-probe", "DD", strings.NewReplacer(".", "_"))
	pkgconfigsetup.InitSystemProbeConfig(systemProbe)

	if configName != "" {
		config.SetConfigName(configName)
	}

	// set the paths where a config file is expected
	if len(confFilePath) != 0 {
		// if the configuration file path was supplied on the command line,
		// add that first so it's first in line
		config.AddConfigPath(confFilePath)
		// If they set a config file directly, let's try to honor that
		if strings.HasSuffix(confFilePath, ".yaml") || strings.HasSuffix(confFilePath, ".yml") {
			config.SetConfigFile(confFilePath)
		}
	}
	if defaultConfPath != "" {
		config.AddConfigPath(defaultConfPath)
	}

	// load the configuration
	var err error
	var warnings *pkgconfigmodel.Warnings

	if withoutSecrets {
		warnings, err = pkgconfigsetup.LoadWithoutSecret(config, systemProbe.GetEnvVars())
	} else {
		warnings, err = pkgconfigsetup.Load(config, systemProbe.GetEnvVars())
	}
	// If `!failOnMissingFile`, do not issue an error if we cannot find the default config file.
	var e viper.ConfigFileNotFoundError
	if err != nil && (failOnMissingFile || !errors.As(err, &e) || confFilePath != "") {
		// special-case permission-denied with a clearer error message
		if errors.Is(err, fs.ErrPermission) {
			if runtime.GOOS == "windows" {
				err = fmt.Errorf(`cannot access the Datadog config file (%w); try running the command in an Administrator shell"`, err)
			} else {
				err = fmt.Errorf("cannot access the Datadog config file (%w); try running the command under the same user as the Datadog Agent", err)
			}
		} else {
			err = fmt.Errorf("unable to load Datadog config file: %w", err)
		}
		return config, warnings, err
	}
	return config, warnings, nil
}
