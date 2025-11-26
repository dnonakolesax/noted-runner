package application

import "sync/atomic"

type HealthChecks struct {
	Docker *atomic.Bool
	Rabbit *atomic.Bool
}

func (a *App) InitHealthchecks() {
	a.health = &HealthChecks{
		Docker: &atomic.Bool{},
		Rabbit: &atomic.Bool{},
	}
}