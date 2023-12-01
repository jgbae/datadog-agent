// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package tracecmd holds the start command of CWS injector
package tracecmd

import (
	"github.com/spf13/cobra"

	"github.com/DataDog/datadog-agent/pkg/config/logs"
	"github.com/DataDog/datadog-agent/pkg/security/ptracer"
)

const (
	// eBPFLessGRPCAddr defines the system-probe GRPC addr
	eBPFLessGRPCAddr = "grpc-addr"
	// logLevel defines the log level
	logLevel = "log-level"
)

type traceCliParams struct {
	EBPFLessGRPCAddr string
	LogLevel         string
}

// Command returns the commands for the trace subcommand
func Command() []*cobra.Command {
	var params traceCliParams

	traceCmd := &cobra.Command{
		Use:   "trace",
		Short: "trace the syscalls and signals of the given binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ptracer.StartCWSPtracer(args, params.EBPFLessGRPCAddr)
		},
	}

	_ = logs.ChangeLogLevel(params.LogLevel)

	traceCmd.Flags().StringVar(&params.EBPFLessGRPCAddr, eBPFLessGRPCAddr, "localhost:5678", "system-probe eBPF less GRPC address")

	return []*cobra.Command{traceCmd}
}
