package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type HTTPRequestMetrics struct {
	RequestDurations prometheus.Histogram
	RequestOks       prometheus.Counter
	RequestBads      prometheus.Counter
	RequestServErrs  prometheus.Counter
}

func NewHTTPRequestMetrics(reg *prometheus.Registry, name string) *HTTPRequestMetrics {
	requestDurations := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    name + "_duration_ms",
		Help:    "A histogram of the " + name + " request durations in ms.",
		Buckets: prometheus.ExponentialBuckets(0.1, 1.5, 5),
	})

	requestOks := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name + "_request_200s",
		Help: "The total number of 200 " + name + " POST requests.",
	})

	request400s := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name + "_request_400s",
		Help: "The total number of 400 " + name + " POST requests.",
	})

	request500s := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name + "_request_500s",
		Help: "The total number of 500 " + name + " POST requests.",
	})

	reg.MustRegister(
		requestDurations,
		requestOks,
		request400s,
		request500s,
	)

	return &HTTPRequestMetrics{
		RequestDurations: requestDurations,
		RequestOks:       requestOks,
		RequestBads:      request400s,
		RequestServErrs:  request500s,
	}
}
