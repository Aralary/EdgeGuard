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
) error {
	if !route.AuthRequired {
		return nil
	}

	if strings.TrimSpace(rawAPIKey) == "" {
		return domain.ErrAPIKeyRequired
	}

	if u.apiKeyValidator == nil {
		return domain.ErrAPIKeyValidatorNotConfigured
	}

	principal, err := u.apiKeyValidator.ValidateAPIKey(ctx, rawAPIKey)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAPIKey) {
			return domain.ErrInvalidAPIKey
		}

		return err
	}

	if principal.ProjectID != route.ProjectID {
		return domain.ErrAPIKeyProjectMismatch
	}

	return nil
}
