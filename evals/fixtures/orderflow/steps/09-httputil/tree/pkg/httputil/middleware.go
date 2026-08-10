// Package httputil holds the middleware both services put in front of their routers.
package httputil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// HeaderRequestID carries a request id between services and into the logs.
const HeaderRequestID = "X-Request-Id"

// Middleware decorates a handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware left to right: the first one sees the request first.
func Chain(h http.Handler, middleware ...Middleware) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

// RequestID keeps the id the caller sent, so one request has one id from end
// to end, and makes one up otherwise.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(HeaderRequestID)
			if id == "" {
				id = newRequestID()
			}
			w.Header().Set(HeaderRequestID, id)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
		})
	}
}

// Logger writes one line per request after the handler returned.
// TODO: sample this once we serve more than a few requests a second.
func Logger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			log.InfoContext(r.Context(), "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", recorder.status),
				slog.Duration("took", time.Since(started)),
				slog.String("request_id", RequestIDFrom(r.Context())),
			)
		})
	}
}

// Recover keeps one panicking handler from taking the whole process down.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				panicked := recover()
				if panicked == nil {
					return
				}
				log.ErrorContext(r.Context(), "handler panicked",
					slog.Any("panic", panicked),
					slog.String("path", r.URL.Path),
					slog.String("request_id", RequestIDFrom(r.Context())),
				)
				WriteError(w, http.StatusInternalServerError, "internal", "internal error")
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDFrom returns the id attached by the RequestID middleware, if any.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

type contextKey int

const requestIDKey contextKey = iota

type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.written {
		return
	}
	w.status = status
	w.written = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(b)
}

func newRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf[:])
}
