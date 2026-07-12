package app

import (
	"os"
	"strings"

	authclient "github.com/aralary/edgeguard/internal/gateway/infrastructure/auth"
	"github.com/aralary/edgeguard/internal/gateway/usecase"
)

func newAPIKeyValidatorFromEnv() (usecase.APIKeyValidator, error) {
	baseURL := strings.TrimSpace(os.Getenv("AUTH_SERVICE_URL"))
	if baseURL == "" {
		return nil, nil
	}

	return authclient.New(baseURL, nil)
}
