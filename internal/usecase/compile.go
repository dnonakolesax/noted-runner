package usecase

import (
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"

	"github.com/dnonakolesax/noted-runner/internal/docker"
)

type Compile struct {
	client       *docker.DockerClient
	mountPath    string
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

	err = uc.client.Run(id)

	if err != nil {
		slog.Error("error running kernel", slog.String("error", err.Error()))
		return "", err
	}
	return id, nil
}

func (uc *Compile) RunBlock(kernelID string, blockID string, userID string) error {
	dir := fmt.Sprintf("%s/%s/%s", uc.mountPath, kernelID, userID)
	cmd := exec.Command("mkdir", "-p", dir)

	err := cmd.Run()

	if err != nil {
		slog.Info("error running mkdir")
		return err
	}

	cmd = exec.Command("cp", "scripts/blockparser.py", dir)

	err = cmd.Run()

	if err != nil {
		slog.Info("error running cp")
		return err
	}

	cmd = exec.Command("cp", "scripts/base", dir)

	err = cmd.Run()

	if err != nil {
		slog.Info("error running cp base")
		return err
	}

	parserPath := fmt.Sprintf("%s/blockparser.py", dir)
	fileName := strings.ReplaceAll(blockID, "-", "_")
	slog.Info("before exec")
	cmd = exec.Command("python3", parserPath, "block_"+fileName, dir)

	data, err := cmd.Output()

	slog.Info("after exec")
	if err != nil {
		slog.Info("error running blockparser", slog.String("path", parserPath), slog.String("data", string(data)), slog.String("file", dir+"/block_"+fileName), slog.String("dir", dir))
		return err
	}

	slog.Info("before resp")
	resp, err := http.Get("http://" + uc.kernelPrefix + kernelID + ":8080/run?block_id=" + blockID + "&user_id=1")
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
