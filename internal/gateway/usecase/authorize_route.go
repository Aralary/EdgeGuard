package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

func (u *Usecase) AuthorizeRoute(
	ctx context.Context,
	route domain.Route,
	rawAPIKey string,
) (domain.APIKeyPrincipal, error) {
	if !route.AuthRequired {
		return domain.APIKeyPrincipal{}, nil
	}

	if strings.TrimSpace(rawAPIKey) == "" {
		return domain.APIKeyPrincipal{}, domain.ErrAPIKeyRequired
	}

	if u.apiKeyValidator == nil {
		return domain.APIKeyPrincipal{}, domain.ErrAPIKeyValidatorNotConfigured
	}

	principal, err := u.apiKeyValidator.ValidateAPIKey(ctx, rawAPIKey)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAPIKey) {
			return domain.APIKeyPrincipal{}, domain.ErrInvalidAPIKey
		}

		return domain.APIKeyPrincipal{}, err
	}

	if principal.ProjectID != route.ProjectID {
		return domain.APIKeyPrincipal{}, domain.ErrAPIKeyProjectMismatch
	}

	return principal, nil
}
