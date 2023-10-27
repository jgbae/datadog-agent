// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !windows

// Package apmetwtracerimpl has no implementation on non-Windows platforms
package apmetwtracerimpl

var Module = fxutil.Component()
