package domain

import "errors"

var (
	ErrRouteNotFound = errors.New("route not found")
	ErrInvalidRoute  = errors.New("invalid route")
)