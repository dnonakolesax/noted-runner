package main

import (
	"flag"

	_ "go.uber.org/automaxprocs"

	"github.com/dnonakolesax/noted-runner/internal/application"
)

// @title COMPILER&RUN API
// @version 1.0
// @description API for authorising users and storing their info

// @contact.name G
// @contact.email bg@dnk33.com

// @host oauth.dnk33.com
// @BasePath /api/v1/compile.
func main() {
	configsPath := flag.String("configs", "./configs", "Path to configs")
	flag.Parse()

	a, err := application.NewApp(*configsPath)
	if err != nil {
		panic(err)
	}

	a.Run()
}
