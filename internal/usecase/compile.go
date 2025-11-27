package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/dnonakolesax/noted-runner/internal/configs"
	"github.com/dnonakolesax/noted-runner/internal/docker"
	"github.com/dnonakolesax/noted-runner/internal/httpclient"
	"github.com/dnonakolesax/noted-runner/internal/logger"
	"github.com/dnonakolesax/noted-runner/internal/preproc"
)

type Compile struct {
	client       *docker.DockerClient
	mountPath    string
	kernelPrefix string
	kernelMuxes  map[string]*sync.Mutex
	kernelTypes  map[string]*preproc.KernelTypes
	logger       *slog.Logger
	sConfig      *configs.ServiceConfig
	hClient      *httpclient.HTTPClient
}

func NewCompilerUsecase(client *docker.DockerClient, mountPath string, kernelPrefix string, logger *slog.Logger,
	sConfig *configs.ServiceConfig, hClient *httpclient.HTTPClient) *Compile {
	return &Compile{client: client, mountPath: mountPath, kernelPrefix: kernelPrefix,
		kernelMuxes: make(map[string]*sync.Mutex),
		kernelTypes: map[string]*preproc.KernelTypes{},
		logger:      logger,
		sConfig:     sConfig,
		hClient:     hClient,
	}
}

func (uc *Compile) StartKernel(kernelID string, userID string) (string, error) {
	uc.kernelMuxes[kernelID+userID] = &sync.Mutex{}
	uc.kernelTypes[kernelID+userID] = preproc.NewKernelTypes()
	id, err := uc.client.Create(fmt.Sprintf("%s%s_u%s", uc.kernelPrefix, kernelID, userID), kernelID)
	if err != nil {
		uc.logger.Error("error starting kernel", logger.LogError(err))
		return "", err
	}

	err = uc.client.Run(id)

	if err != nil {
		uc.logger.Error("error running kernel", logger.LogError(err))
		return "", err
	}
	return id, nil
}

func (uc *Compile) RunBlock(kernelID string, blockID string, userID string) error {
	uc.kernelMuxes[kernelID+userID].Lock()
	defer uc.kernelMuxes[kernelID+userID].Unlock()
	filePath := fmt.Sprintf("%s/%s/%s/%s", uc.mountPath, kernelID, userID, "block_"+blockID)

	file, err := os.ReadFile(filePath)

	if err != nil {
		uc.logger.Error("error reading file with block", logger.LogError(err), slog.String("file", filePath))
		return err
	}

	types := uc.kernelTypes[kernelID+userID]
	block := preproc.NewBlock(blockID, string(file), types)

	err = block.Parse()

	if err != nil {
		uc.logger.Error("error parsing block", logger.LogError(err))
		return fmt.Errorf("error parsing block: %s", err)
	}

	code := block.FormExportFunc()

	//fmt.Printf("code: %s", code)

	err = os.WriteFile(filePath+".go", []byte(code), os.ModeExclusive)

	if err != nil {
		uc.logger.Error("error saving block file", logger.LogError(err), slog.String("file", filePath+".go"))
		return err
	}

	ctxI, cancelI := context.WithTimeout(context.Background(), uc.sConfig.CMDTimeout)
	defer cancelI()
	cmd := exec.CommandContext(ctxI, "goimports", "-w", filePath+".go")
	out, err := cmd.CombinedOutput()
	if err != nil {
		uc.logger.Error("error running goimports", logger.LogError(err), slog.String("file", filePath+".go"))
		return fmt.Errorf("error running goimports: %v\nOutput: %s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), uc.sConfig.CompileTimeout)
	defer cancel()

	filePath2 := fmt.Sprintf("%s/%s/%s/%s.so", uc.mountPath, kernelID, userID, "block_"+strings.ReplaceAll(blockID, "-", "_"))
	cmd = exec.CommandContext(ctx, "go", "build", "-buildmode=plugin", "-o", filePath2, filePath+".go")
	out, err = cmd.CombinedOutput()
	if err != nil {
		uc.logger.Error("error building", logger.LogError(err), slog.String("file", filePath2))
		return fmt.Errorf("error running go build: %v\nOutput: %s", err, out)
	}

	//slog.Info("before resp")
	resp, err := http.Get("http://" + uc.kernelPrefix + kernelID + "_u" + userID + ":8080/run?block_id=" + blockID + "&user_id=" + userID)
	//slog.Info("after resp")
	if err != nil {
		uc.logger.Error("error sending http", logger.LogError(err))
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	return nil
}

func (uc *Compile) StopKernel(id string) error {
	err := uc.client.Remove(id)
	if err != nil {
		uc.logger.Error("error removing kernel container", logger.LogError(err))
		return err
	}
	return nil
}
