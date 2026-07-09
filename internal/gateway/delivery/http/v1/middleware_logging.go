package httpdelivery

import (
	"time"

	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/labstack/echo/v5"
)

func Logging(log logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()

			if req.URL.Path == "/health" {
				return next(c)
			}

			start := time.Now()
			originalResponse := c.Response()
			recorder := newResponseRecorder(originalResponse)

			c.SetResponse(recorder)
			defer c.SetResponse(originalResponse)

			err := next(c)

			log.Infof(
				"gateway request completed: request_id=%s method=%s path=%s status=%d bytes=%d duration=%s",
				req.Header.Get(requestIDHeader),
				req.Method,
				req.URL.Path,
				recorder.status,
				recorder.bytes,
				time.Since(start).String(),
			)

			return err
		}
	}
}
