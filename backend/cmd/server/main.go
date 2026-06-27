package main

import (
	"log"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/docs"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/cli"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/buildinfo"
)

// @title openachieve API
// @version 0.2.8
// @description openachieve backend API documentation
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	docs.SwaggerInfo.Version = buildinfo.ResolveVersion()
	if err := cli.Run(); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
