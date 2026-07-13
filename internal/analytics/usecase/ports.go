package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/analytics/domain"
	"github.com/aralary/edgeguard/internal/platform/events"
)

type GatewayAccessRepository interface {
	StoreGatewayAccess(
		ctx context.Context,
		event events.GatewayAccessEvent,
	) (domain.IngestionResult, error)
}
