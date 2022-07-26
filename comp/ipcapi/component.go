// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package ipcapi implements a component to manage the IPC API server.  It
// allows other components to register handlers, and manages startup and shutdown
// of the HTTP server.
//
// XXX In a real agent, this would support TLS and gRPC and Auth and timeouts
// and stuff; see cmd/agent/api.
package ipcapi

import (
	"net/http"

	"go.uber.org/fx"
)

// team: agent-shared-components

// Component is the component type.
type Component interface {
	// Register registers a handler at an HTTP path.
	Register(path string, handler http.HandlerFunc)
}

type ModuleParams struct {
	// Disabled indicates that the component should ignore all registration and
	// perform no monitoring.  This is intended for one-shot processes such as
	// `agent status`.
	Disabled bool
}

var Module = fx.Module(
	"comp/ipcapi",
	fx.Provide(newServer),
)
