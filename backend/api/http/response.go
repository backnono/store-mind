package http

import "github.com/gin-gonic/gin"

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, errorResponse{Code: code, Message: message})
}
