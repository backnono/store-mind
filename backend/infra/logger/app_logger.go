package logger

import "go.uber.org/zap"

type AppLogger struct {
	sugar *zap.SugaredLogger
}

func NewAppLogger(base *zap.Logger) *AppLogger {
	if base == nil {
		base = zap.NewNop()
	}
	return &AppLogger{sugar: base.Sugar()}
}

func (l *AppLogger) Info(msg string, keysAndValues ...any) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l *AppLogger) Warn(msg string, keysAndValues ...any) {
	l.sugar.Warnw(msg, keysAndValues...)
}

func (l *AppLogger) Error(msg string, keysAndValues ...any) {
	l.sugar.Errorw(msg, keysAndValues...)
}
