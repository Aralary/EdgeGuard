package httpdelivery

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
)

const requestIDHeader = "X-Request-ID"

func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			requestID := c.Request().Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = newRequestID()
			}

			c.Request().Header.Set(requestIDHeader, requestID)
			c.Response().Header().Set(requestIDHeader, requestID)

			return next(c)
		}
	}
}

func newRequestID() string {
	buffer := make([]byte, 16)

	if _, err := rand.Read(buffer); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	return hex.EncodeToString(buffer)
}
