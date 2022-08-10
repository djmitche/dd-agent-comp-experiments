// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package file

import (
	"context"
	"time"

	"github.com/DataDog/dd-agent-comp-experiments/comp/core/config"
	"github.com/DataDog/dd-agent-comp-experiments/comp/core/health"
	"github.com/DataDog/dd-agent-comp-experiments/comp/core/log"
	"github.com/DataDog/dd-agent-comp-experiments/comp/logs/internal"
	"github.com/DataDog/dd-agent-comp-experiments/comp/logs/internal/sourcemgr"
	"github.com/DataDog/dd-agent-comp-experiments/comp/logs/launchers/launchermgr"
	"github.com/DataDog/dd-agent-comp-experiments/pkg/util/actor"
	"github.com/DataDog/dd-agent-comp-experiments/pkg/util/subscriptions"
	"go.uber.org/fx"
)

type launcher struct {
	log          log.Component
	subscription subscriptions.Subscription[sourcemgr.SourceChange]
	actor        actor.Goroutine
	health       *health.Registration
}

type dependencies struct {
	fx.In

	Lc     fx.Lifecycle
	Config config.Component
	Params internal.BundleParams
	Log    log.Component
}

type provides struct {
	fx.Out

	Component
	HealthReg      *health.Registration      `group:"true"`
	Subscription   sourcemgr.Subscription    `group:"true"`
	LauncherMgrReg *launchermgr.Registration `group:"true"`
}

func newLauncher(deps dependencies) provides {
	l := &launcher{
		log:    deps.Log,
		health: health.NewRegistration(componentName),
	}
	if deps.Params.ShouldStart(deps.Config) {
		l.actor.HookLifecycle(deps.Lc, l.run)
		l.subscription = sourcemgr.Subscribe()
	}
	return provides{
		Component:      l,
		HealthReg:      l.health,
		LauncherMgrReg: launchermgr.NewRegistration("file", l),
		Subscription:   l.subscription,
	}
}

func (l *launcher) run(ctx context.Context) {
	monitor, stopMonitor := l.health.LivenessMonitor(time.Second)
	for {
		select {
		case chg := <-l.subscription.Chan():
			l.log.Debug("file launcher got LogSource change", chg)
			// XXX start a tailer, etc. etc.
		case <-monitor:
		case <-ctx.Done():
			stopMonitor()
			return
		}
	}
}
