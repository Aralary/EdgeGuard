package httpdelivery

import (
	"io"
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

			err := next(c)

			buf, err := io.ReadAll(req.Body)

			if err != nil {
				log.Errorf(
					"gateway request failed: request_id=%s method=%s path=%s bytes=%d duration=%s error=%v",
					req.Header.Get(requestIDHeader),
					req.Method,
					req.URL.Path,
					len(buf),
					time.Since(start).String(),
					err,
				)

				return err
			}

			log.Infof(
				"gateway request completed: request_id=%s method=%s path=%s bytes=%d duration=%s",
				req.Header.Get(requestIDHeader),
				req.Method,
				req.URL.Path,
				len(buf),
				time.Since(start).String(),
			)

			return err
		}
	}
}
