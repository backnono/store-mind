package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewRouter(l *zap.Logger, customerQAHandler *CustomerQAHandler, feedbackHandler *FeedbackHandler) *gin.Engine {
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
	v1.POST("/customer-qa/feedback", feedbackHandler.Submit)
	v1.GET("/customer-qa/faqs/search", customerQAHandler.SearchFAQ)
	v1.GET("/customer-qa/products/search", customerQAHandler.SearchProducts)
	v1.GET("/customer-qa/products/:product_id/location", customerQAHandler.GetProductLocation)
	v1.GET("/customer-qa/skus/:sku_id/inventory", customerQAHandler.GetInventory)
	v1.GET("/customer-qa/promotions/active", customerQAHandler.ListActivePromotions)
	admin := r.Group("/api/admin")
	admin.GET("/customer-qa/sessions", customerQAHandler.ListSessions)
	admin.GET("/customer-qa/tool-calls", customerQAHandler.ListToolCalls)
	return r
}
