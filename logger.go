package paystack

// Logger is the simplest logger contract the SDK accepts. Callers that want
// structured output should implement LeveledLogger instead.
type Logger interface {
	Printf(format string, args ...interface{})
}

// LeveledLogger lets integration packages route SDK logs into their framework's
// level-aware logger (zap, zerolog, slog, logrus, etc).
type LeveledLogger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}
