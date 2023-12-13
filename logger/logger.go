package logger

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var DefaultLogger *zap.SugaredLogger

// nolint:gochecknoinits // this is the simplest way to initialize the logger
func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(errors.Errorf("failed to initialize the logger: %v", err))
	}
	DefaultLogger = logger.Sugar()
}
