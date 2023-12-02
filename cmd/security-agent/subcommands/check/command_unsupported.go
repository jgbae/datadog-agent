// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build windows || !kubeapiserver

// Package check holds check related files
package check

import (
	"github.com/spf13/cobra"

	"github.com/DataDog/datadog-agent/cmd/security-agent/command"
	"github.com/DataDog/datadog-agent/comp/core"
)

// SecurityAgentCommands returns security agent commands
//
//nolint:revive // TODO(SEC) Fix revive linter
func SecurityAgentCommands(globalParams *command.GlobalParams) []*cobra.Command {
	return nil
}

// ClusterAgentCommands returns cluster agent commands
//
//nolint:revive // TODO(SEC) Fix revive linter
func ClusterAgentCommands(bundleParams core.BundleParams) []*cobra.Command {
	return nil
}
