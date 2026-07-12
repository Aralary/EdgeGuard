package usecase

import (
	"context"
	"strings"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

func (u *Usecase) CheckRateLimit(
	ctx context.Context,
	route domain.Route,
	clientID string,
) (domain.RateLimitResult, error) {
	if !route.RateLimit.Enabled {
		return domain.RateLimitResult{
			Enabled: false,
			Allowed: true,
		}, nil
	}

	if route.RateLimit.Limit <= 0 || route.RateLimit.Window <= 0 {
		return domain.RateLimitResult{}, domain.ErrInvalidRateLimitPolicy
	}

	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return domain.RateLimitResult{}, domain.ErrInvalidRateLimitClient
	}

	if u.rateLimiter == nil {
		return domain.RateLimitResult{}, domain.ErrRateLimiterNotConfigured
	}

	return u.rateLimiter.Allow(ctx, domain.RateLimitRequest{
		RouteKey: route.ProjectID + "\x00" + route.Name + "\x00" + route.PathPrefix,
		ClientID: clientID,
		Policy:   route.RateLimit,
	})
}
