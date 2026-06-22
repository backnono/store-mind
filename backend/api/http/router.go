package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewRouter(l *zap.Logger, customerQAHandler *CustomerQAHandler, feedbackHandler *FeedbackHandler, adminHandler *AdminHandler) *gin.Engine {
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

	// Admin CRUD for all resources
	resources := admin.Group("/resources")
	{
		resources.POST("/stores", adminHandler.CreateStore)
		resources.PUT("/stores/:id", adminHandler.UpdateStore)
		resources.DELETE("/stores/:id", adminHandler.DeleteStore)

		resources.POST("/zones", adminHandler.CreateZone)
		resources.PUT("/zones/:id", adminHandler.UpdateZone)
		resources.DELETE("/zones/:id", adminHandler.DeleteZone)

		resources.POST("/shelves", adminHandler.CreateShelf)
		resources.PUT("/shelves/:id", adminHandler.UpdateShelf)
		resources.DELETE("/shelves/:id", adminHandler.DeleteShelf)

		resources.POST("/products", adminHandler.CreateProduct)
		resources.PUT("/products/:id", adminHandler.UpdateProduct)
		resources.DELETE("/products/:id", adminHandler.DeleteProduct)

		resources.POST("/skus", adminHandler.CreateSKU)
		resources.PUT("/skus/:id", adminHandler.UpdateSKU)
		resources.DELETE("/skus/:id", adminHandler.DeleteSKU)

		resources.POST("/product-locations", adminHandler.CreateProductLocation)
		resources.PUT("/product-locations/:id", adminHandler.UpdateProductLocation)
		resources.DELETE("/product-locations/:id", adminHandler.DeleteProductLocation)

		resources.POST("/inventories", adminHandler.CreateInventory)
		resources.PUT("/inventories/:id", adminHandler.UpdateInventory)
		resources.DELETE("/inventories/:id", adminHandler.DeleteInventory)

		resources.POST("/promotions", adminHandler.CreatePromotion)
		resources.PUT("/promotions/:id", adminHandler.UpdatePromotion)
		resources.DELETE("/promotions/:id", adminHandler.DeletePromotion)

		resources.POST("/faqs", adminHandler.CreateFAQ)
		resources.PUT("/faqs/:id", adminHandler.UpdateFAQ)
		resources.DELETE("/faqs/:id", adminHandler.DeleteFAQ)
	}
	return r
}
