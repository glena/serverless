package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/glena/pulumi-faas/provisioning"
	"github.com/glena/pulumi-faas/routes"
)

func main() {
	r := gin.Default()

	program := provisioning.Provisioning{Configuration: provisioning.AWSConfiguration{
		Region:    os.Getenv("REGION"),
		AccessKey: os.Getenv("ACCESS_KEY"),
		SecretKey: os.Getenv("SECRET_KEY"),
	}}

	routes.Register(r, program)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
