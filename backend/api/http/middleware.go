package http

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	requestIDHeader = "X-Request-Id"
	maxBodyLogSize  = 4096
)

func RequestLogger(l *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		rid := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		c.Writer.Header().Set(requestIDHeader, rid)
		c.Request = c.Request.WithContext(withRequestID(c.Request.Context(), rid))

		query := c.Request.URL.RawQuery
		bodyText := ""
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.URL.Path != "/healthz" {
			bodyText = extractBodyForLog(c)
		}

		c.Next()

		fields := []zap.Field{
			zap.String("request_id", rid),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("query", query),
		}
		if bodyText != "" {
			fields = append(fields, zap.String("body", bodyText))
		}
		l.Info("http_request", fields...)
	}
}

func extractBodyForLog(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyLogSize+1))
	if err != nil {
		return ""
	}
	_ = c.Request.Body.Close()

	truncated := len(raw) > maxBodyLogSize
	if truncated {
		raw = raw[:maxBodyLogSize]
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))

	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		var obj any
		if err := json.Unmarshal(raw, &obj); err == nil {
			maskSensitive(obj)
			if b, err := json.Marshal(obj); err == nil {
				if truncated {
					return string(b) + "...(truncated)"
				}
				return string(b)
			}
		}
	}

	if truncated {
		return string(raw) + "...(truncated)"
	}
	return string(raw)
}

func maskSensitive(v any) {
	switch val := v.(type) {
	case map[string]any:
		for k, sub := range val {
			if isSensitiveKey(k) {
				val[k] = "***"
				continue
			}
			maskSensitive(sub)
		}
	case []any:
		for i := range val {
			maskSensitive(val[i])
		}
	}
}

func isSensitiveKey(k string) bool {
	lk := strings.ToLower(strings.TrimSpace(k))
	return lk == "password" || lk == "token" || lk == "authorization"
}
