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
	StoreID int64  `json:"store_id"`
	UserID  *int64 `json:"user_id"`
	Channel string `json:"channel"`
	Message string `json:"message"`
}

func (h *CustomerQAHandler) Chat(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("chat_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	h.log.Info("chat_request", zap.String("request_id", rid), zap.Int64("store_id", req.StoreID), zap.String("channel", req.Channel))
	resp, err := h.svc.Chat(c.Request.Context(), app.ChatRequest{RequestID: rid, StoreID: req.StoreID, UserID: req.UserID, Channel: req.Channel, Message: req.Message})
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

	h.log.Info("chat_success", zap.String("request_id", rid), zap.Int64("session_id", resp.SessionID), zap.Int64("message_id", resp.MessageID))
	c.JSON(http.StatusOK, gin.H{
		"session_id": resp.SessionID,
		"message_id": resp.MessageID,
		"intent":     resp.Intent,
		"answer":     resp.Answer,
		"meta": gin.H{
			"request_id": rid,
		},
	})
}

func (h *CustomerQAHandler) SearchFAQ(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	storeID, err := strconv.ParseInt(c.Query("store_id"), 10, 64)
	if err != nil {
		h.log.Warn("faq_bad_request", zap.String("request_id", rid), zap.Error(err))
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

	faqs, err := h.svc.SearchFAQ(c.Request.Context(), app.FAQSearchRequest{RequestID: rid, StoreID: storeID, Query: query, Limit: limit})
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

	h.log.Info("faq_search_success", zap.String("request_id", rid), zap.Int64("store_id", storeID), zap.Int("count", len(faqs)))
	c.JSON(http.StatusOK, gin.H{
		"items": faqs,
		"meta": gin.H{
			"request_id": rid,
		},
	})
}
