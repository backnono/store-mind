package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func buildTestLogger(buf *bytes.Buffer) *zap.Logger {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = ""
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		zapcore.AddSync(buf),
		zapcore.InfoLevel,
	)
	return zap.New(core)
}

func TestRequestLogger_AddsRequestIDAndLogsParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	l := buildTestLogger(&buf)

	r := gin.New()
	r.Use(RequestLogger(l))
	r.POST("/t", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body, _ := json.Marshal(map[string]any{"message": "hello", "password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/t?q=abc", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "rid-1")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("X-Request-Id") != "rid-1" {
		t.Fatalf("expected response request id")
	}
	out := buf.String()
	if !strings.Contains(out, "\"request_id\":\"rid-1\"") {
		t.Fatalf("log missing request_id: %s", out)
	}
	if !strings.Contains(out, "\"query\":\"q=abc\"") {
		t.Fatalf("log missing query: %s", out)
	}
	if !strings.Contains(out, "\"body\":") {
		t.Fatalf("log missing body: %s", out)
	}
	if strings.Contains(out, "secret") {
		t.Fatalf("sensitive value leaked: %s", out)
	}
}
