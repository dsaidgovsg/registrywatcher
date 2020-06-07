package log

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

func SetUpLogger() {
	logger, err := zap.NewProduction()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing logger.")
		return
	}

	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintln(os.Stderr, "Error flushing buffered log entries.")
		}
	}()

	zap.ReplaceGlobals(logger)
}

func LogAppInfo(msg string) {
	zap.S().Infow(msg)
}

func LogAppWarn(msg string, err error) {
	zap.S().Warnw(msg,
		"cause", err,
	)
}

func LogAppErr(msg string, err error) {
	zap.S().Errorw(msg,
		"cause", err,
	)
}
