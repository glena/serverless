package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PostFunctionBody struct {
	Name   string `json:"name" binding:"required"`
	Script string `json:"script" binding:"required"`
}

func PostFunction(c *gin.Context) {
	requestBody := PostFunctionBody{}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		// c.AbortWithError(http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, err)
		return
	}



	c.JSON(http.StatusAccepted, &requestBody)
}
