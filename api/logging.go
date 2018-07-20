package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/sirupsen/logrus"
)

// contextKey is the type used for context keys by this package.
type contextKey string

// loggerContextKey is the key at which a logger is stored in a request's context by loggerMiddleware().
const loggerContextKey contextKey = "logger"

// loggerMiddleware returns a http.Handler that adds a logger to r and calls next.ServeHTTP().
// The logger can be retrieved with reqLogger().
func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loggerContextKey, logrus.WithFields(logrus.Fields{
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
			"method":      r.Method,
			"uri":         r.URL.RequestURI(),
		}))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// logRequestResponseMiddleware returns a http.Handler that writes a log entry for each request and response.
func logRequestResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqLogger(r).Info("Received request")
		m := httpsnoop.CaptureMetrics(next, w, r)
		reqLogger(r).WithFields(logrus.Fields{
			"status_code": m.Code,
			"duration":    m.Duration,
		}).Info("Handled request")
	})
}

// reqLogger returns the logger added to r by loggerMiddleware().
func reqLogger(r *http.Request) logrus.FieldLogger {
	loggerVal := r.Context().Value(loggerContextKey)
	if loggerVal == nil {
		logrus.WithField("key", loggerContextKey).Panic("can't find logger in request context at expected key")
	}

	logger, ok := loggerVal.(logrus.FieldLogger)
	if !ok {
		logrus.WithField("type", fmt.Sprintf("%T", loggerVal)).Panic("unexpected type for logger in request context")
	}

	return logger
}
