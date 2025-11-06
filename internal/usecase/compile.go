package usecase

import (
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"

	"github.com/dnonakolesax/noted-runner/internal/docker"
)

type Compile struct {
	client *docker.DockerClient
	mountPath string
	kernelPrefix string
}

func NewCompilerUsecase(client *docker.DockerClient, mountPath string, kernelPrefix string) *Compile {
	return &Compile{client: client, mountPath: mountPath, kernelPrefix: kernelPrefix}
}

func (uc *Compile) StartKernel(id string) (string, error) {
	id, err := uc.client.Create(fmt.Sprintf("%s%s", uc.kernelPrefix, id))
	if err != nil {
		slog.Error("error starting kernel", slog.String("error", err.Error()))
		return "", err
	}
	return id, nil
}

func (uc *Compile) RunBlock(kernelID string, blockID string) error {
	dir := fmt.Sprintf("%s/%s", uc.mountPath, kernelID)
	cmd := exec.Command("mkdir", "-p", dir)
	
	err := cmd.Run()

	if err != nil {
		return err
	}

	cmd = exec.Command("cp", "scripts/blockparser.py", dir)

	err = cmd.Run()

	if err != nil {
		return err
	}

	parserPath := fmt.Sprintf("%s/blockparser.py", dir)
	cmd = exec.Command("python3", parserPath, "block_" + blockID + ".go", dir)

	err = cmd.Run()

	if err != nil {
		return err
	}

	resp, err := http.Get("http://" + uc.kernelPrefix + kernelID + ":8089/compile")  
    if err != nil {  
        fmt.Println("Ошибка запроса:", err)  
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
		slog.Error("error removing kernel container", slog.String("error", err.Error()))
		return err
	}
	return nil
}
