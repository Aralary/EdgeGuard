package httpdelivery

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := newResponseRecorder(w)

		next.ServeHTTP(recorder, r)

		log.Info(
			"gateway request completed",
			slog.String("request_id", r.Header.Get(requestIDHeader)),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Int("bytes", recorder.bytes),
			slog.Duration("duration", time.Since(start)),
		)
	})
}