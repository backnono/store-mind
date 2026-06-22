package http

import (
	"net/http"
	"strconv"

	domain "store-mind/domain/customerqa"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminHandler 管理后台 CRUD HTTP 处理器
type AdminHandler struct {
	adminRepo domain.AdminRepository
	log       *zap.Logger
}

func NewAdminHandler(adminRepo domain.AdminRepository, log *zap.Logger) *AdminHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &AdminHandler{adminRepo: adminRepo, log: log}
}

// ---------- Store ----------

func (h *AdminHandler) CreateStore(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req domain.Store
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_store_create_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	store, err := h.adminRepo.CreateStore(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_store_create_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create store")
		return
	}
	h.log.Info("admin_store_create_success", zap.String("request_id", rid), zap.Int64("id", store.ID))
	c.JSON(http.StatusCreated, gin.H{"item": store, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) UpdateStore(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var req domain.Store
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_store_update_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.ID = id
	store, err := h.adminRepo.UpdateStore(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_store_update_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to update store")
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": store, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) DeleteStore(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	if err := h.adminRepo.DeleteStore(c.Request.Context(), id); err != nil {
		h.log.Error("admin_store_delete_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to delete store")
		return
	}
	h.log.Info("admin_store_delete_success", zap.String("request_id", rid), zap.Int64("id", id))
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": rid}})
}

// ---------- Zone ----------

func (h *AdminHandler) CreateZone(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req domain.Zone
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_zone_create_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	zone, err := h.adminRepo.CreateZone(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_zone_create_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create zone")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": zone, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) UpdateZone(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var req domain.Zone
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_zone_update_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.ID = id
	zone, err := h.adminRepo.UpdateZone(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_zone_update_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to update zone")
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": zone, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) DeleteZone(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	if err := h.adminRepo.DeleteZone(c.Request.Context(), id); err != nil {
		h.log.Error("admin_zone_delete_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to delete zone")
		return
	}
	h.log.Info("admin_zone_delete_success", zap.String("request_id", rid), zap.Int64("id", id))
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": rid}})
}

// ---------- Shelf ----------

func (h *AdminHandler) CreateShelf(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req domain.Shelf
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_shelf_create_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	shelf, err := h.adminRepo.CreateShelf(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_shelf_create_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create shelf")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": shelf, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) UpdateShelf(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var req domain.Shelf
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_shelf_update_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.ID = id
	shelf, err := h.adminRepo.UpdateShelf(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_shelf_update_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to update shelf")
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": shelf, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) DeleteShelf(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	if err := h.adminRepo.DeleteShelf(c.Request.Context(), id); err != nil {
		h.log.Error("admin_shelf_delete_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to delete shelf")
		return
	}
	h.log.Info("admin_shelf_delete_success", zap.String("request_id", rid), zap.Int64("id", id))
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": rid}})
}

// ---------- Product ----------

func (h *AdminHandler) CreateProduct(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req domain.Product
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_product_create_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	product, err := h.adminRepo.CreateProduct(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_product_create_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create product")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": product, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) UpdateProduct(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var req domain.Product
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_product_update_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.ID = id
	product, err := h.adminRepo.UpdateProduct(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_product_update_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to update product")
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": product, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) DeleteProduct(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	if err := h.adminRepo.DeleteProduct(c.Request.Context(), id); err != nil {
		h.log.Error("admin_product_delete_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to delete product")
		return
	}
	h.log.Info("admin_product_delete_success", zap.String("request_id", rid), zap.Int64("id", id))
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": rid}})
}

// ---------- SKU ----------

func (h *AdminHandler) CreateSKU(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req domain.SKU
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_sku_create_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	sku, err := h.adminRepo.CreateSKU(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_sku_create_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create sku")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": sku, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) UpdateSKU(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var req domain.SKU
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_sku_update_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.ID = id
	sku, err := h.adminRepo.UpdateSKU(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_sku_update_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to update sku")
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": sku, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) DeleteSKU(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	if err := h.adminRepo.DeleteSKU(c.Request.Context(), id); err != nil {
		h.log.Error("admin_sku_delete_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to delete sku")
		return
	}
	h.log.Info("admin_sku_delete_success", zap.String("request_id", rid), zap.Int64("id", id))
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": rid}})
}

// ---------- ProductLocation ----------

func (h *AdminHandler) CreateProductLocation(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req domain.ProductLocation
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_product_location_create_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	pl, err := h.adminRepo.CreateProductLocation(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_product_location_create_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create product location")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": pl, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) UpdateProductLocation(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var req domain.ProductLocation
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_product_location_update_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.ID = id
	pl, err := h.adminRepo.UpdateProductLocation(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_product_location_update_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to update product location")
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": pl, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) DeleteProductLocation(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	if err := h.adminRepo.DeleteProductLocation(c.Request.Context(), id); err != nil {
		h.log.Error("admin_product_location_delete_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to delete product location")
		return
	}
	h.log.Info("admin_product_location_delete_success", zap.String("request_id", rid), zap.Int64("id", id))
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": rid}})
}

// ---------- Inventory ----------

func (h *AdminHandler) CreateInventory(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req domain.Inventory
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_inventory_create_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	inv, err := h.adminRepo.CreateInventory(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_inventory_create_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create inventory")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": inv, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) UpdateInventory(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var req domain.Inventory
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_inventory_update_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.ID = id
	inv, err := h.adminRepo.UpdateInventory(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_inventory_update_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to update inventory")
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": inv, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) DeleteInventory(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	if err := h.adminRepo.DeleteInventory(c.Request.Context(), id); err != nil {
		h.log.Error("admin_inventory_delete_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to delete inventory")
		return
	}
	h.log.Info("admin_inventory_delete_success", zap.String("request_id", rid), zap.Int64("id", id))
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": rid}})
}

// ---------- Promotion ----------

func (h *AdminHandler) CreatePromotion(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req domain.Promotion
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_promotion_create_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	promo, err := h.adminRepo.CreatePromotion(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_promotion_create_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create promotion")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": promo, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) UpdatePromotion(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var req domain.Promotion
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_promotion_update_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.ID = id
	promo, err := h.adminRepo.UpdatePromotion(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_promotion_update_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to update promotion")
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": promo, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) DeletePromotion(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	if err := h.adminRepo.DeletePromotion(c.Request.Context(), id); err != nil {
		h.log.Error("admin_promotion_delete_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to delete promotion")
		return
	}
	h.log.Info("admin_promotion_delete_success", zap.String("request_id", rid), zap.Int64("id", id))
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": rid}})
}

// ---------- FAQ ----------

func (h *AdminHandler) CreateFAQ(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req domain.FAQ
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_faq_create_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	faq, err := h.adminRepo.CreateFAQ(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_faq_create_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to create faq")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": faq, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) UpdateFAQ(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var req domain.FAQ
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("admin_faq_update_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.ID = id
	faq, err := h.adminRepo.UpdateFAQ(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("admin_faq_update_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to update faq")
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": faq, "meta": gin.H{"request_id": rid}})
}

func (h *AdminHandler) DeleteFAQ(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	if err := h.adminRepo.DeleteFAQ(c.Request.Context(), id); err != nil {
		h.log.Error("admin_faq_delete_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to delete faq")
		return
	}
	h.log.Info("admin_faq_delete_success", zap.String("request_id", rid), zap.Int64("id", id))
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": rid}})
}
