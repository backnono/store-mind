package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewRouter(l *zap.Logger, customerQAHandler *CustomerQAHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	if l != nil {
		r.Use(RequestLogger(l))
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	v1.POST("/customer-qa/chat", customerQAHandler.Chat)
	v1.GET("/customer-qa/faqs/search", customerQAHandler.SearchFAQ)
	return r
}
