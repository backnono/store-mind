package customerqa

// Logger 是应用层的结构化日志接口，遵循 key=value 交替的 zap-style。
// 调用方需保证 keysAndValues 为偶数个（key1, value1, key2, value2, ...），
// 奇数个参数时末尾 key 将被忽略。
type Logger interface {
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

// nopLogger 空日志实现，不对日志做任何输出。
// 作为 Logger 的零依赖默认值：当构造方法未传入 Logger 时使用。
type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}
