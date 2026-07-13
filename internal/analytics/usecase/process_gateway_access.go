package usecase

import (
	"context"
	"fmt"

	"github.com/aralary/edgeguard/internal/analytics/domain"
	"github.com/aralary/edgeguard/internal/platform/events"
)

func (u *Usecase) ProcessGatewayAccess(
	ctx context.Context,
	event events.GatewayAccessEvent,
) (domain.IngestionResult, error) {
	if err := event.Validate(); err != nil {
		return domain.IngestionResult{}, fmt.Errorf("%w: %v", ErrInvalidGatewayAccessEvent, err)
	}
	if u.gatewayAccessRepository == nil {
		return domain.IngestionResult{}, fmt.Errorf("gateway access repository is not configured")
	}

	result, err := u.gatewayAccessRepository.StoreGatewayAccess(ctx, event)
	if err != nil {
		return domain.IngestionResult{}, fmt.Errorf("store gateway access event: %w", err)
	}

	return result, nil
}
