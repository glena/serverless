package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/glena/pulumi-faas/provisioning"
)

func Register(r *gin.Engine, program provisioning.Provisioning) {
	r.POST("/function", func(c *gin.Context) {
		PostFunction(c, program)
	})
}
