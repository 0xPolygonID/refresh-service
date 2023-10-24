package logger

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var DefaultLogger *zap.SugaredLogger

func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(errors.Errorf("failed to initialize the logger: %v", err))
	}
	DefaultLogger = logger.Sugar()
}
