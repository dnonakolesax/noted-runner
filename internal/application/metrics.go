package application

import (
	"github.com/dnonakolesax/noted-runner/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Metrics struct {
	RunnerMetrics      *metrics.HTTPRequestMetrics

	Reg *prometheus.Registry
}

func (a *App) SetupMetrics() {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	runnerRequestMetrics := metrics.NewHTTPRequestMetrics(reg, "runner_get")

	a.metrics = &Metrics{
		RunnerMetrics: runnerRequestMetrics,
		Reg: reg,
	}
}
