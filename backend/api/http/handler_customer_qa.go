package http

import (
	"errors"
	"net/http"
	"strconv"

	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CustomerQAHandler struct {
	svc app.Service
	log *zap.Logger
}

func NewCustomerQAHandler(svc app.Service, log *zap.Logger) *CustomerQAHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &CustomerQAHandler{svc: svc, log: log}
}

type chatRequest struct {
	StoreID   int64  `json:"store_id"`
	SessionID int64  `json:"session_id"`
	UserID    *int64 `json:"user_id"`
	Channel   string `json:"channel"`
	Message   string `json:"message"`
}

func (h *CustomerQAHandler) Chat(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("chat_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	h.log.Info(
		"chat_request",
		zap.String("request_id", rid),
		zap.Int64("store_id", req.StoreID),
		zap.String("channel", req.Channel),
	)
	resp, err := h.svc.Chat(
		c.Request.Context(),
		app.ChatRequest{
			RequestID: rid,
			StoreID:   req.StoreID,
			SessionID: req.SessionID,
			UserID:    req.UserID,
			Channel:   req.Channel,
			Message:   req.Message,
		},
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) {
			h.log.Warn("chat_invalid_argument", zap.String("request_id", rid), zap.Error(err))
			writeError(c, http.StatusBadRequest, "invalid_argument", "store_id and message are required")
			return
		}
		h.log.Error("chat_internal_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	h.log.Info(
		"chat_success",
		zap.String("request_id", rid),
		zap.Int64("session_id", resp.SessionID),
		zap.Int64("message_id", resp.MessageID),
	)
	c.JSON(
		http.StatusOK, gin.H{
			"session_id":       resp.SessionID,
			"message_id":       resp.MessageID,
			"intent":           resp.Intent,
			"answer":           resp.Answer,
			"cards":            resp.Cards,
			"handoff_required": resp.HandoffRequired,
			"meta": gin.H{
				"request_id":     rid,
				"route":          resp.Meta.Route,
				"confidence":     resp.Meta.Confidence,
				"rewrite_query":  resp.Meta.RewriteQuery,
				"fallback_used":  resp.Meta.FallbackUsed,
				"evidence_count": resp.Meta.EvidenceCount,
			},
		},
	)
}

func (h *CustomerQAHandler) SearchFAQ(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	storeID, ok := parseStoreID(c)
	if !ok {
		h.log.Warn("faq_bad_request", zap.String("request_id", rid))
		writeError(c, http.StatusBadRequest, "bad_request", "store_id is required")
		return
	}
	query := c.Query("q")
	limit := 10
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	faqs, err := h.svc.SearchFAQ(
		c.Request.Context(),
		app.FAQSearchRequest{RequestID: rid, StoreID: storeID, Query: query, Limit: limit},
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) {
			h.log.Warn("faq_invalid_argument", zap.String("request_id", rid), zap.Error(err))
			writeError(c, http.StatusBadRequest, "invalid_argument", "store_id and q are required")
			return
		}
		h.log.Error("faq_internal_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	h.log.Info(
		"faq_search_success",
		zap.String("request_id", rid),
		zap.Int64("store_id", storeID),
		zap.Int("count", len(faqs)),
	)
	c.JSON(
		http.StatusOK, gin.H{
			"items": faqs,
			"meta": gin.H{
				"request_id": rid,
			},
		},
	)
}

func (h *CustomerQAHandler) SearchProducts(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	storeID, ok := parseStoreID(c)
	if !ok {
		h.log.Warn("product_search_bad_request", zap.String("request_id", rid))
		writeError(c, http.StatusBadRequest, "bad_request", "store_id is required")
		return
	}
	query := c.Query("q")
	limit := parseLimit(c, 10)

	items, err := h.svc.SearchProducts(
		c.Request.Context(),
		app.ProductSearchRequest{RequestID: rid, StoreID: storeID, Query: query, Limit: limit},
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) {
			h.log.Warn("product_search_invalid_argument", zap.String("request_id", rid), zap.Error(err))
			writeError(c, http.StatusBadRequest, "invalid_argument", "store_id and q are required")
			return
		}
		h.log.Error("product_search_internal_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	c.JSON(
		http.StatusOK, gin.H{
			"items": items,
			"meta": gin.H{
				"request_id": rid,
			},
		},
	)
}

func (h *CustomerQAHandler) GetProductLocation(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	storeID, ok := parseStoreID(c)
	if !ok {
		h.log.Warn("product_location_bad_request", zap.String("request_id", rid))
		writeError(c, http.StatusBadRequest, "bad_request", "store_id is required")
		return
	}
	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 64)
	if err != nil {
		h.log.Warn("product_location_bad_path", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "product_id is required")
		return
	}

	item, err := h.svc.GetProductLocation(
		c.Request.Context(),
		app.ProductLocationRequest{RequestID: rid, StoreID: storeID, ProductID: productID},
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) {
			h.log.Warn("product_location_invalid_argument", zap.String("request_id", rid), zap.Error(err))
			writeError(c, http.StatusBadRequest, "invalid_argument", "store_id and product_id are required")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			h.log.Warn("product_location_not_found", zap.String("request_id", rid), zap.Error(err))
			writeError(c, http.StatusNotFound, "not_found", "product location not found")
			return
		}
		h.log.Error("product_location_internal_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *CustomerQAHandler) GetInventory(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	storeID, ok := parseStoreID(c)
	if !ok {
		h.log.Warn("inventory_bad_request", zap.String("request_id", rid))
		writeError(c, http.StatusBadRequest, "bad_request", "store_id is required")
		return
	}
	skuID, err := strconv.ParseInt(c.Param("sku_id"), 10, 64)
	if err != nil {
		h.log.Warn("inventory_bad_path", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "sku_id is required")
		return
	}

	item, err := h.svc.GetInventory(
		c.Request.Context(),
		app.InventoryRequest{RequestID: rid, StoreID: storeID, SKUID: skuID},
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) {
			h.log.Warn("inventory_invalid_argument", zap.String("request_id", rid), zap.Error(err))
			writeError(c, http.StatusBadRequest, "invalid_argument", "store_id and sku_id are required")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			h.log.Warn("inventory_not_found", zap.String("request_id", rid), zap.Error(err))
			writeError(c, http.StatusNotFound, "not_found", "inventory not found")
			return
		}
		h.log.Error("inventory_internal_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *CustomerQAHandler) ListActivePromotions(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	storeID, ok := parseStoreID(c)
	if !ok {
		h.log.Warn("promotion_bad_request", zap.String("request_id", rid))
		writeError(c, http.StatusBadRequest, "bad_request", "store_id is required")
		return
	}
	limit := parseLimit(c, 10)

	items, err := h.svc.ListActivePromotions(
		c.Request.Context(),
		app.PromotionListRequest{RequestID: rid, StoreID: storeID, Limit: limit},
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) {
			h.log.Warn("promotion_invalid_argument", zap.String("request_id", rid), zap.Error(err))
			writeError(c, http.StatusBadRequest, "invalid_argument", "store_id is required")
			return
		}
		h.log.Error("promotion_internal_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	c.JSON(
		http.StatusOK, gin.H{
			"items": items,
			"meta": gin.H{
				"request_id": rid,
			},
		},
	)
}

func (h *CustomerQAHandler) ListSessions(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	storeID, ok := parseStoreID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "bad_request", "store_id is required")
		return
	}
	limit := parseLimit(c, 20)
	items, err := h.svc.ListSessions(
		c.Request.Context(),
		app.SessionListRequest{RequestID: rid, StoreID: storeID, Limit: limit},
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) {
			writeError(c, http.StatusBadRequest, "invalid_argument", "store_id is required")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	c.JSON(
		http.StatusOK, gin.H{
			"items": items,
			"meta":  gin.H{"request_id": rid},
		},
	)
}

func (h *CustomerQAHandler) ListToolCalls(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	sessionID, err := strconv.ParseInt(c.Query("session_id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "session_id is required")
		return
	}
	limit := parseLimit(c, 20)
	items, err := h.svc.ListToolCalls(
		c.Request.Context(),
		app.ToolCallListRequest{RequestID: rid, SessionID: sessionID, Limit: limit},
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) {
			writeError(c, http.StatusBadRequest, "invalid_argument", "session_id is required")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	c.JSON(
		http.StatusOK, gin.H{
			"items": items,
			"meta":  gin.H{"request_id": rid},
		},
	)
}

func parseStoreID(c *gin.Context) (int64, bool) {
	storeID, err := strconv.ParseInt(c.Query("store_id"), 10, 64)
	if err != nil {
		return 0, false
	}
	return storeID, true
}

func parseLimit(c *gin.Context, fallback int) int {
	limit := fallback
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	return limit
}
