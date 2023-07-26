package routes

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/glena/pulumi-faas/provisioning"
)

type PostFunctionBody struct {
	Name   string `json:"name" binding:"required"`
	Script string `json:"script" binding:"required"`
}

type PostFunctionResponse struct {
	Url string `json:"url"`
}

func PostFunction(c *gin.Context, program provisioning.Provisioning) {
	requestBody := PostFunctionBody{}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		// c.AbortWithError(http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	fmt.Printf("Deploying stack %q %q\n", requestBody.Name, requestBody.Script)

	url, err := program.Provision(requestBody.Name, requestBody.Script)

	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusAccepted, &PostFunctionResponse{
		Url: url,
	})
}
