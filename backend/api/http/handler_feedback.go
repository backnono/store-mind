package http

import (
	"net/http"

	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// FeedbackHandler 处理回答反馈请求
type FeedbackHandler struct {
	svc app.Service
	log *zap.Logger
}

func NewFeedbackHandler(svc app.Service, log *zap.Logger) *FeedbackHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &FeedbackHandler{svc: svc, log: log}
}

type feedbackRequest struct {
	MessageID     int64 `json:"message_id" binding:"required"`
	SessionID     int64 `json:"session_id" binding:"required"`
	FeedbackValue int8  `json:"feedback_value" binding:"required"` // 1=👍 / 0=👎
}

func (h *FeedbackHandler) Submit(c *gin.Context) {
	rid := requestIDFromContext(c.Request.Context())
	var req feedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("feedback_bad_request", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusBadRequest, "bad_request", "message_id, session_id, and feedback_value are required")
		return
	}

	if req.FeedbackValue != 0 && req.FeedbackValue != 1 {
		h.log.Warn("feedback_invalid_value", zap.String("request_id", rid), zap.Int8("value", req.FeedbackValue))
		writeError(c, http.StatusBadRequest, "bad_request", "feedback_value must be 0 (👎) or 1 (👍)")
		return
	}

	err := h.svc.SaveFeedback(c.Request.Context(), req.MessageID, req.SessionID, req.FeedbackValue)
	if err != nil {
		if err == domain.ErrInvalidArgument {
			h.log.Warn("feedback_invalid_argument", zap.String("request_id", rid), zap.Error(err))
			writeError(c, http.StatusBadRequest, "invalid_argument", "invalid message_id or session_id")
			return
		}
		h.log.Error("feedback_internal_error", zap.String("request_id", rid), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	h.log.Info("feedback_success",
		zap.String("request_id", rid),
		zap.Int64("message_id", req.MessageID),
		zap.Int8("value", req.FeedbackValue),
	)
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
