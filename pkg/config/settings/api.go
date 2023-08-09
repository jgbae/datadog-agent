// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package settings TODO comment
package settings

import "github.com/spf13/cobra"

// Client is the interface for interacting with the runtime settings API
type Client interface {
	Get(key string) (interface{}, error)
	Set(key string, value string) (bool, error)
	List() (map[string]RuntimeSettingResponse, error)
	FullConfig() (string, error)
}

// ClientBuilder represents a function returning a runtime settings API client
type ClientBuilder func(_ *cobra.Command, _ []string) (Client, error)
