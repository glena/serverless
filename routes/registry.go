package routes

import (
	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine) {
	r.POST("/function", PostFunction)
	r.GET("/function/:id/status", GetFunctionStatus)
}
