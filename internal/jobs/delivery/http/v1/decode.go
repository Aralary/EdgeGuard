package httpdelivery

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/labstack/echo/v5"
)

var errInvalidRequestBody = errors.New("invalid request body")

func decodeRequest(c *echo.Context, destination any) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return errInvalidRequestBody
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errInvalidRequestBody
	}

	return nil
}
