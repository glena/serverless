package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/glena/pulumi-faas/provisioning"
	"github.com/glena/pulumi-faas/routes"
)

func main() {
	configuration := provisioning.AWSConfiguration{
		Region:    os.Getenv("REGION"),
		AccessKey: os.Getenv("ACCESS_KEY"),
		SecretKey: os.Getenv("SECRET_KEY"),
	}

	if configuration.Region == "" {
		log.Fatal("REGION configuration is not set")
	}

	if configuration.AccessKey == "" {
		log.Fatal("ACCESS_KEY configuration is not set")
	}

	if configuration.SecretKey == "" {
		log.Fatal("SECRET_KEY configuration is not set")
	}

	r := gin.Default()

	program := provisioning.Provisioning{Configuration: configuration}

	routes.Register(r, program)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
