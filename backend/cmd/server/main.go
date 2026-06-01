package main

import (
	"log"

	"store-mind/internal/bootstrap"

	"github.com/gin-gonic/gin"
)

func main() {
	app, err := bootstrap.Build()
	if err != nil {
		log.Fatal(err)
	}
	defer app.Logger.Sync() //nolint:errcheck

	r, ok := app.Router.(*gin.Engine)
	if !ok {
		log.Fatal("router type assertion failed")
	}

	app.Logger.Info("server_start")
	if err := r.Run(app.Config.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
