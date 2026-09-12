package tmdb

import "go.uber.org/zap"

// restyLogger routes resty's internal per-attempt logging (retries, final
// failures) to debug level. The outcome of each request is reported once by
// requesterLogger, so resty's warnings would otherwise duplicate that line for
// every retry attempt.
type restyLogger struct {
	logger *zap.SugaredLogger
}

func (l restyLogger) Errorf(format string, v ...interface{}) {
	l.logger.Debugf(format, v...)
}

func (l restyLogger) Warnf(format string, v ...interface{}) {
	l.logger.Debugf(format, v...)
}

func (l restyLogger) Debugf(format string, v ...interface{}) {
	l.logger.Debugf(format, v...)
}
