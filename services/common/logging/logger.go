package logging

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger     *zap.SugaredLogger
	loggerOnce sync.Once
)

// Init은 지정된 로그 레벨로 전역 로거를 초기화합니다.
func Init(service string, level string) (*zap.SugaredLogger, error) {
	var initErr error

	loggerOnce.Do(func() {
		cfg := zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "time"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.EncoderConfig.CallerKey = "caller"

		lvl := zap.NewAtomicLevel()
		if err := lvl.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
			initErr = fmt.Errorf("invalid log level %q: %w", level, err)
			return
		}
		cfg.Level = lvl
		cfg.InitialFields = map[string]interface{}{
			"service": service,
		}

		zapLogger, err := cfg.Build()
		if err != nil {
			initErr = fmt.Errorf("build zap logger: %w", err)
			return
		}

		logger = zapLogger.Sugar()
	})

	return logger, initErr
}

// L는 초기화된 전역 로거를 반환합니다.
func L() *zap.SugaredLogger {
	if logger == nil {
		// 초기화 이전에 호출되는 경우를 대비해 기본 로거 생성
		return zap.NewExample().Sugar()
	}
	return logger
}

// Sync는 버퍼된 로그를 플러시합니다.
func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}
