package utils

import (
	"runtime/debug"

	"go.uber.org/zap"
)

// Recover logs a panic recovered in a deferred call together with a stack
// trace. Usage: defer utils.Recover(logger, "read loop panicked").
func Recover(logger *zap.SugaredLogger, message string, kv ...any) {
	r := recover() //nolint:revive // Recover is invoked via defer at every call site
	if r == nil {
		return
	}

	kv = append(kv, "panic", r, "stack", string(debug.Stack()))
	logger.Errorw(message, kv...)
}
