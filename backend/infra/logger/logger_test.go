package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_ProductionWritesToDailyFile(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "info")
	logDir := t.TempDir()
	t.Setenv("LOG_DIR", logDir)

	l := New()
	l.Info("hello_prod_log")
	_ = l.Sync()

	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected log file generated")
	}

	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "app-") && strings.HasSuffix(e.Name(), ".log") {
			b, err := os.ReadFile(filepath.Join(logDir, e.Name()))
			if err != nil {
				t.Fatalf("read log file: %v", err)
			}
			if strings.Contains(string(b), "hello_prod_log") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected log content not found")
	}
}
