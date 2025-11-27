package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"time"

	"github.com/dnonakolesax/noted-runner/internal/configs"
	"github.com/dnonakolesax/noted-runner/internal/consts"
	"github.com/dnonakolesax/noted-runner/internal/metrics"
)

const methodLoggerKey = "method"

type HTTPClient struct {
	c       *http.Client
	retries configs.HTTPRetryPolicyConfig
	metrics *metrics.HTTPRequestMetrics
	logger  *slog.Logger
}

type HTTPRequestParams struct {
	endpoint string
}

func NewWithRetry(config *configs.HTTPClientConfig,
	reqMetrics *metrics.HTTPRequestMetrics, logger *slog.Logger) (*HTTPClient, error) {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   config.DialTimeout,
			KeepAlive: config.KeepAlive,
		}).DialContext,
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConns, // хост один, так что и значение одно
		IdleConnTimeout:     config.IdleConnTimeout,
	}

	c := &HTTPClient{
		c:       &http.Client{Transport: tr, Timeout: config.RequestTimeout},
		retries: config.RetryPolicy,
		metrics: reqMetrics,
		logger:  logger,
	}

	return c, nil
}

func (hc *HTTPClient) observeStatusCode(code int) {
	switch {
	case code <= http.StatusBadRequest:
		hc.metrics.RequestOks.Inc()
	case code <= http.StatusInternalServerError:
		hc.metrics.RequestBads.Inc()
	default:
		hc.metrics.RequestServErrs.Inc()
	}
}

func (hc *HTTPClient) Get(ctx context.Context, endpoint string) (*http.Response, error) {
	return hc.makeRequest(ctx, http.MethodGet, HTTPRequestParams{endpoint: endpoint})
}

func (hc *HTTPClient) makeRequest(ctx context.Context, method string,
	params HTTPRequestParams) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 1; attempt <= hc.retries.MaxAttempts; attempt++ {
		resp, lastErr = hc.executeRequestAttempt(ctx, method, params)
		if resp != nil || !hc.shouldRetry(lastErr, resp, attempt) {
			break
		}

		if err := sleepOrCtx(ctx, hc.backoffDelay(attempt)); err != nil {
			return nil, err
		}
	}

	return resp, lastErr
}

func (hc *HTTPClient) executeRequestAttempt(ctx context.Context, method string,
	params HTTPRequestParams) (*http.Response, error) {
	ctxCreate, cancel := context.WithTimeout(context.TODO(), hc.c.Timeout)
	defer cancel()
	req, err := hc.createRequest(ctxCreate, method, params)
	if err != nil {
		return nil, err
	}

	reqStart := time.Now().UnixMilli()
	hc.logger.InfoContext(ctx, "Executing request", slog.String(methodLoggerKey, method))
	resp, err := hc.c.Do(req)
	reqEnd := time.Now().UnixMilli()
	hc.logger.InfoContext(ctx, "Executed request", slog.Int64("time", reqEnd-reqStart),
		slog.String(methodLoggerKey, method))

	hc.metrics.RequestDurations.Observe(float64(reqEnd - reqStart))
	hc.observeRequestStatus(resp, err)

	if err != nil {
		hc.logger.ErrorContext(ctx, "Error executing http-request", slog.String(consts.ErrorLoggerKey, err.Error()),
			slog.String(methodLoggerKey, method))
		return nil, err
	}

	if hc.shouldRetryStatus(resp.StatusCode) {
		drainAndClose(resp.Body)
		hc.logger.InfoContext(ctx, "Should retry request", slog.String(methodLoggerKey, method),
			slog.Int("Code", resp.StatusCode))
		return resp, fmt.Errorf("retryable HTTP status %d", resp.StatusCode)
	}

	return resp, nil
}

func (hc *HTTPClient) createRequest(ctx context.Context, method string,
	params HTTPRequestParams) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method,
		params.endpoint, nil)
	if err != nil {
		hc.logger.ErrorContext(ctx, "Error creating http-request", slog.String(consts.ErrorLoggerKey, err.Error()))
		return nil, err
	}
	return req, nil
}

func (hc *HTTPClient) observeRequestStatus(resp *http.Response, err error) {
	if err != nil {
		hc.observeStatusCode(http.StatusBadRequest)
	} else if resp != nil {
		hc.observeStatusCode(resp.StatusCode)
	}
}

func (hc *HTTPClient) shouldRetry(err error, resp *http.Response, attempt int) bool {
	if attempt >= hc.retries.MaxAttempts {
		return false
	}

	if err != nil {
		return hc.shouldRetryError(err)
	}

	return resp != nil && hc.shouldRetryStatus(resp.StatusCode)
}

func (hc *HTTPClient) shouldRetryStatus(code int) bool {
	return hc.retries.RetryOnStatus[code]
}

func (hc *HTTPClient) shouldRetryError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var nerr net.Error
	if errors.As(err, &nerr) {
		return nerr.Timeout()
	}
	return true
}

func (hc *HTTPClient) backoffDelay(attempt int) time.Duration {
	// экспоненциальная задержка: Base * 2^(attempt-1), но не больше MaxDelay
	d := hc.retries.BaseDelay << (attempt - 1)
	if d > hc.retries.MaxDelay {
		d = hc.retries.MaxDelay
	}
	// джиттер ~±20%
	jitterFrac := 0.2

	// Здесь нам не нужен крипторандом, так как мы не нуждаемся в криптостойкости результата
	// В свою очередь, криптографический рандом использует системные вызовы, так как зависит от некоторых
	// параметров системы и компьютера (тепловой шум процессора, i/o активность, номера дисков и прочее)
	j := time.Duration(float64(d) * (rand.Float64()*2*jitterFrac - jitterFrac)) //nolint:gosec // см выше ^
	return d + j
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}

func sleepOrCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
