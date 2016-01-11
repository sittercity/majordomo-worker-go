package majordomo_worker

import (
	"errors"
)

// Copied from go-kit/log to save us from needing the entire package as a dependency
// See https://github.com/go-kit/kit/tree/master/log for more details
type Logger interface {
	Log(keyvals ...interface{}) error
}

var ErrMissingValue = errors.New("(MISSING)")

func logDebug(logger Logger, msg string) {
	logger.Log("level", "debug", "message", msg)
}

func logWarn(logger Logger, msg string) {
	logger.Log("level", "warn", "message", msg)
}

func logError(logger Logger, msg string) {
	logger.Log("level", "error", "message", msg)
}
