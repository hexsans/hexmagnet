package testutil

import "go.uber.org/zap"

func NewTestLogger() *zap.SugaredLogger {
	return zap.NewNop().Sugar()
}
