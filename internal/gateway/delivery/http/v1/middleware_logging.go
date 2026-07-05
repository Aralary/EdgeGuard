package httpdelivery

import (
	"io"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

func Logging(log *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()

			err := next(c)

			req := c.Request()

			body, err := io.ReadAll(req.Body)
			if err != nil {
				log.WithError(err).Error("failed to read request body")
				return err
			}

			log.WithFields(logrus.Fields{
				"request_id": req.Header.Get(requestIDHeader),
				"method":     req.Method,
				"path":       req.URL.Path,
				"bytes":      len(body),
				"duration":   time.Since(start).String(),
			}).Info("gateway request completed")

			return err
		}
	}
}
