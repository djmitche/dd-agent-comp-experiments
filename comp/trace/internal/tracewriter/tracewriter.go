// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package tracewriter

import (
	"context"
	"time"

	"github.com/djmitche/dd-agent-comp-experiments/comp/health"
	"github.com/djmitche/dd-agent-comp-experiments/comp/util/log"
	"github.com/djmitche/dd-agent-comp-experiments/pkg/trace/api"
	"github.com/djmitche/dd-agent-comp-experiments/pkg/util/actor"
	"go.uber.org/fx"
)

type traceWriter struct {
	in chan *api.Payload

	actor  actor.Goroutine
	log    log.Component
	health *health.ActorRegistration
}

type dependencies struct {
	fx.In

	Lc     fx.Lifecycle
	Health health.Component
	Log    log.Component
}

func newTraceWriter(deps dependencies) Component {
	t := &traceWriter{
		in:     make(chan *api.Payload, 1000),
		health: deps.Health.RegisterActor("comp/trace/internal/tracewriter", 1*time.Second),
		log:    deps.Log,
	}
	t.actor.HookLifecycle(deps.Lc, t.run)
	return t
}

func (t *traceWriter) PayloadChan() chan<- *api.Payload {
	return t.in
}

func (t *traceWriter) run(ctx context.Context) {
	defer t.health.Stop()
	for {
		select {
		case payload := <-t.in:
			t.log.Debug("sending payload", payload)
		case <-t.health.Chan():
		case <-ctx.Done():
			return
		}
	}
}