package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/platform/events"
)

func (u *Usecase) PublishAccessEvent(ctx context.Context, event events.GatewayAccessEvent) error {
	if u.accessEventPublisher == nil {
		return nil
	}

	return u.accessEventPublisher.PublishGatewayAccess(ctx, event)
}
