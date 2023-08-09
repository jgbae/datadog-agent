// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package config TODO comment
package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/cmd/process-agent/command"
	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/process"
	apiutil "github.com/DataDog/datadog-agent/pkg/api/util"
	ddconfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/config/settings"
	settingshttp "github.com/DataDog/datadog-agent/pkg/config/settings/http"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

type dependencies struct {
	fx.In

	GlobalParams *command.GlobalParams

	Config config.Component
}

// Commands returns a slice of subcommands for the `config` command in the Process Agent
func Commands(globalParams *command.GlobalParams) []*cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print the runtime configuration of a running agent",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fxutil.OneShot(showRuntimeConfiguration,
				fx.Supply(globalParams, command.GetCoreBundleParamsForOneShot(globalParams)),

				process.Bundle,
			)
		},
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list-runtime",
			Short: "List settings that can be changed at runtime",
			Long:  ``,
			RunE: func(cmd *cobra.Command, args []string) error {
				return fxutil.OneShot(listRuntimeConfigurableValue,
					fx.Supply(globalParams, command.GetCoreBundleParamsForOneShot(globalParams)),

					process.Bundle,
				)
			},
		},
	)

	cmd.AddCommand(
		&cobra.Command{
			Use:   "set [setting] [value]",
			Short: "Set, for the current runtime, the value of a given configuration setting",
			Long:  ``,
			RunE: func(cmd *cobra.Command, args []string) error {
				return fxutil.OneShot(setConfigValue,
					fx.Supply(globalParams, args, command.GetCoreBundleParamsForOneShot(globalParams)),

					process.Bundle,
				)
			},
		},
	)
	cmd.AddCommand(
		&cobra.Command{
			Use:   "get [setting]",
			Short: "Get, for the current runtime, the value of a given configuration setting",
			Long:  ``,
			RunE: func(cmd *cobra.Command, args []string) error {
				return fxutil.OneShot(getConfigValue,
					fx.Supply(globalParams, args, command.GetCoreBundleParamsForOneShot(globalParams)),

					process.Bundle,
				)
			},
		},
	)

	return []*cobra.Command{cmd}
}

func showRuntimeConfiguration(deps dependencies) error {
	c, err := getClient(deps.Config)
	if err != nil {
		return err
	}

	runtimeConfig, err := c.FullConfig()
	if err != nil {
		return err
	}

	fmt.Println(runtimeConfig)

	return nil
}

func listRuntimeConfigurableValue(deps dependencies) error {
	c, err := getClient(deps.Config)
	if err != nil {
		return err
	}

	settingsList, err := c.List()
	if err != nil {
		return err
	}

	fmt.Println("=== Settings that can be changed at runtime ===")
	for setting, details := range settingsList {
		if !details.Hidden {
			fmt.Printf("%-30s %s\n", setting, details.Description)
		}
	}

	return nil
}

func setConfigValue(deps dependencies, args []string) error {
	c, err := getClient(deps.Config)
	if err != nil {
		return err
	}

	if len(args) != 2 {
		return fmt.Errorf("exactly two parameters are required: the setting name and its value")
	}

	hidden, err := c.Set(args[0], args[1])
	if err != nil {
		return err
	}

	if hidden {
		fmt.Printf("IMPORTANT: you have modified a hidden option, this may incur billing charges or have other unexpected side-effects.\n")
	}

	fmt.Printf("Configuration setting %s is now set to: %s\n", args[0], args[1])

	return nil
}

func getConfigValue(deps dependencies, args []string) error {
	c, err := getClient(deps.Config)
	if err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("a single setting name must be specified")
	}

	value, err := c.Get(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("%s is set to: %v\n", args[0], value)

	return nil
}

func getClient(cfg ddconfig.ConfigReader) (settings.Client, error) {
	httpClient := apiutil.GetClient(false)
	ipcAddress, err := ddconfig.GetIPCAddress()

	port := cfg.GetInt("process_config.cmd_port")
	if port <= 0 {
		return nil, fmt.Errorf("invalid process_config.cmd_port -- %d", port)
	}

	ipcAddressWithPort := fmt.Sprintf("http://%s:%d/config", ipcAddress, port)
	if err != nil {
		return nil, err
	}
	settingsClient := settingshttp.NewClient(httpClient, ipcAddressWithPort, "process-agent")
	return settingsClient, nil
}
