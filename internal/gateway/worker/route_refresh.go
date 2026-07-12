package worker

import (
	"context"
	"time"

	"github.com/aralary/edgeguard/internal/platform/logger"
)

type RoutesRefresher interface {
	RefreshRoutes(ctx context.Context) (int, error)
}

func RunRoutesRefresh(
	ctx context.Context,
	interval time.Duration,
	refresher RoutesRefresher,
	log logger.Logger,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := refresher.RefreshRoutes(ctx)
			if err != nil {
				if ctx.Err() == nil {
					log.Warnf("gateway routes refresh failed: %v", err)
				}
				continue
			}

			log.Infof("gateway routes refreshed: routes=%d", count)
		}
	}
}
