package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New() *zap.Logger {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	level := parseLevel(strings.TrimSpace(os.Getenv("LOG_LEVEL")))

	if env == "prod" || env == "production" {
		logDir := strings.TrimSpace(os.Getenv("LOG_DIR"))
		if logDir == "" {
			logDir = "logs"
		}
		w, err := newDailyFileWriter(logDir, "app")
		if err == nil {
			encCfg := zap.NewProductionEncoderConfig()
			encCfg.TimeKey = "time"
			core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(w), level)
			return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		}
	}

	encCfg := zap.NewDevelopmentEncoderConfig()
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), zapcore.AddSync(os.Stdout), level)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

func parseLevel(v string) zapcore.Level {
	switch strings.ToLower(v) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

type dailyFileWriter struct {
	dir     string
	prefix  string
	mu      sync.Mutex
	current string
	file    *os.File
	nowFunc func() time.Time
}

func newDailyFileWriter(dir, prefix string) (*dailyFileWriter, error) {
	w := &dailyFileWriter{dir: dir, prefix: prefix, nowFunc: time.Now}
	if err := w.rotateIfNeeded(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dailyFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

func (w *dailyFileWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

func (w *dailyFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	return w.file.Close()
}

func (w *dailyFileWriter) rotateIfNeeded() error {
	date := w.nowFunc().Format("2006-01-02")
	if w.file != nil && w.current == date {
		return nil
	}
	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return err
	}
	if w.file != nil {
		_ = w.file.Close()
	}
	path := filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", w.prefix, date))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	w.current = date
	return nil
}
