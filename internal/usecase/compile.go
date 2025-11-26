package usecase

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/dnonakolesax/noted-runner/internal/docker"
	"github.com/dnonakolesax/noted-runner/internal/preproc"
)

type Compile struct {
	client       *docker.DockerClient
	mountPath    string
	kernelPrefix string
	kernelMuxes  map[string]*sync.Mutex
	kernelTypes  map[string]*preproc.KernelTypes
}

func NewCompilerUsecase(client *docker.DockerClient, mountPath string, kernelPrefix string) *Compile {
	return &Compile{client: client, mountPath: mountPath, kernelPrefix: kernelPrefix, kernelMuxes: make(map[string]*sync.Mutex), kernelTypes: map[string]*preproc.KernelTypes{}}
}

func (uc *Compile) StartKernel(kernelID string, userID string) (string, error) {
	uc.kernelMuxes[kernelID+userID] = &sync.Mutex{}
	uc.kernelTypes[kernelID+userID] = preproc.NewKernelTypes()
	id, err := uc.client.Create(fmt.Sprintf("%s%s_u%s", uc.kernelPrefix, kernelID, userID), kernelID)
	if err != nil {
		slog.Error("error starting kernel", slog.String("error", err.Error()))
		return "", err
	}

	err = uc.client.Run(id)

	if err != nil {
		slog.Error("error running kernel", slog.String("error", err.Error()))
		return "", err
	}
	return id, nil
}

func (uc *Compile) RunBlock(kernelID string, blockID string, userID string) error {
	uc.kernelMuxes[kernelID+userID].Lock()
	defer uc.kernelMuxes[kernelID+userID].Unlock()
	filePath := fmt.Sprintf("%s/%s/%s/%s", uc.mountPath, kernelID, userID, "block_" + blockID)

	file, err := os.ReadFile(filePath)

	if err != nil {
		return err
	}

	types := uc.kernelTypes[kernelID+userID]
	block := preproc.NewBlock(blockID, string(file), types)

	err = block.Parse()

	if err != nil {
		return fmt.Errorf("error parsing block: %s", err)
	}

	code := block.FormExportFunc()

	fmt.Printf("code: %s", code)

	err = os.WriteFile(filePath + ".go", []byte(code), os.ModeExclusive)

	if err != nil {
		return err
	}

	cmd := exec.Command("goimports", "-w", filePath + ".go")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running goimports: %v\nOutput: %s", err, out)
	}

	filePath2 := fmt.Sprintf("%s/%s/%s/%s.so", uc.mountPath, kernelID, userID, "block_" + strings.ReplaceAll(blockID, "-", "_"))
	cmd = exec.Command("go", "build", "-buildmode=plugin", "-o", filePath2, filePath + ".go")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running go build: %v\nOutput: %s", err, out)
	}

	slog.Info("before resp")
	resp, err := http.Get("http://" + uc.kernelPrefix + kernelID + "_u" + userID + ":8080/run?block_id=" + blockID + "&user_id=" + userID)
	slog.Info("after resp")
	if err != nil {
		fmt.Println("Ошибка запроса:", err)
		return err
	} else {
		slog.Debug("200 OK")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return nil
}

func (uc *Compile) StopKernel(id string) error {
	err := uc.client.Remove(id)
	if err != nil {
		slog.Error("error removing kernel container", slog.String("error", err.Error()))
		return err
	}
	return nil
}
