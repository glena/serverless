package main

import (
	"github.com/gin-gonic/gin"
	"github.com/glena/pulumi-faas/routes"
)

func main() {
	r := gin.Default()

	routes.Register(r)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
