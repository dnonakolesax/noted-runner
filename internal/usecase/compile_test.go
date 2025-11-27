package usecase

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/dnonakolesax/noted-runner/internal/configs"
	"github.com/dnonakolesax/noted-runner/internal/httpclient"
	"github.com/dnonakolesax/noted-runner/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestCompile(t *testing.T) {
	lg := slog.Default()
	scfg := &configs.ServiceConfig{CompileTimeout: time.Minute, CMDTimeout: time.Minute}

	cfg := &configs.HTTPClientConfig{}

	reg := prometheus.NewRegistry()
	mtr := metrics.NewHTTPRequestMetrics(reg, "zxc")
	client, err := httpclient.NewWithRetry(cfg, mtr, lg)

	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	uc := NewCompilerUsecase(nil, "/noted/codes/kernels", "noted-kernel_", lg, scfg, client)

	uc.kernelMuxes["11"] = &sync.Mutex{}
	err = uc.RunBlock("1", "4bcb102d_d663_4bec_86b4_86e978b5b54c", "1")

	if err != nil {
		t.Fatalf("%s", err.Error())
	}
}
