package configs

import (
	"net/http"
	"time"

	"github.com/dnonakolesax/viper"
)

const (
	clientHTTPMaxRetryAttemptsKey = "http-client.retries.max-attempts"
	cHTTPMaxRetryAttemptsDefault  = 3
	clientHTTPRetryBaseDelayKey   = "http-client.retries.base-delay"
	cHTTPRetryBaseDelayDefault    = 200 * time.Millisecond
	clientHTTPRetryMaxDelayKey    = "http-client.retries.max-delay"
	cHTTPRetryMaxDelayDefault     = 3 * time.Second
	clientHTTPRetryOnStatusKey    = "http-client.retries.on-status"
	clientHTTPDialTimeoutKey      = "http-client.dial-timeout"
	cHTTPDialTimeoutDefault       = 5 * time.Second
	clientHTTPRequestTimeoutKey   = "http-client.request-timeout"
	cHTTPRequestTimeoutDefault    = 30 * time.Second
	clientHTTPKeepAliveKey        = "http-client.keep-alive"
	cHTTPRequestKeepAliveDefault  = 30 * time.Second
	clientHTTPMaxIdleConnsKey     = "http-client.max-idle-conns"
	cHTTPMaxIdleConnsDefault      = 100
	clientHTTPIdleConnTimeoutKey  = "http-client.idle-conn-timeout"
	cHTTPIdleConnTimeoutDefault   = 90 * time.Second
)

type HTTPRetryPolicyConfig struct {
	MaxAttempts   int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	RetryOnStatus map[int]bool
}

func (hc *HTTPRetryPolicyConfig) Load(v *viper.Viper) {
	retriesConfig := v.GetIntSlice(clientHTTPRetryOnStatusKey)
	retryOnStatus := make(map[int]bool, len(retriesConfig))

	for _, status := range retriesConfig {
		retryOnStatus[status] = true
	}

	hc.MaxAttempts = v.GetInt(clientHTTPMaxRetryAttemptsKey)
	hc.BaseDelay = v.GetDuration(clientHTTPRetryBaseDelayKey)
	hc.MaxDelay = v.GetDuration(clientHTTPRetryMaxDelayKey)
	hc.RetryOnStatus = retryOnStatus
}

func (hc *HTTPRetryPolicyConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault(clientHTTPMaxRetryAttemptsKey, cHTTPMaxRetryAttemptsDefault)
	v.SetDefault(clientHTTPRetryBaseDelayKey, cHTTPRetryBaseDelayDefault)
	v.SetDefault(clientHTTPRetryMaxDelayKey, cHTTPRetryMaxDelayDefault)
	v.SetDefault(clientHTTPRetryOnStatusKey, []int{http.StatusTooManyRequests, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout})
}

type HTTPClientConfig struct {
	DialTimeout     time.Duration
	RequestTimeout  time.Duration
	KeepAlive       time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration
	RetryPolicy     HTTPRetryPolicyConfig
}

func (hc *HTTPClientConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault(clientHTTPDialTimeoutKey, cHTTPDialTimeoutDefault)
	v.SetDefault(clientHTTPRequestTimeoutKey, cHTTPRequestTimeoutDefault)
	v.SetDefault(clientHTTPKeepAliveKey, cHTTPRequestKeepAliveDefault)
	v.SetDefault(clientHTTPMaxIdleConnsKey, cHTTPMaxIdleConnsDefault)
	v.SetDefault(clientHTTPIdleConnTimeoutKey, cHTTPIdleConnTimeoutDefault)
	hc.RetryPolicy.SetDefaults(v)
}

func (hc *HTTPClientConfig) Load(v *viper.Viper) {
	hc.DialTimeout = v.GetDuration(clientHTTPDialTimeoutKey)
	hc.RequestTimeout = v.GetDuration(clientHTTPRequestTimeoutKey)
	hc.KeepAlive = v.GetDuration(clientHTTPKeepAliveKey)
	hc.MaxIdleConns = v.GetInt(clientHTTPMaxIdleConnsKey)
	hc.IdleConnTimeout = v.GetDuration(clientHTTPIdleConnTimeoutKey)
	hc.RetryPolicy.Load(v)
}
