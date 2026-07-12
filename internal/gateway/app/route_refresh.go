package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/infrastructure/controlplane"
	"github.com/aralary/edgeguard/internal/gateway/usecase"
	"github.com/aralary/edgeguard/internal/platform/logger"
)

const defaultRoutesRefreshInterval = 10 * time.Second

func newRouteSourceFromEnv() (usecase.RouteSource, time.Duration, error) {
	baseURL := strings.TrimSpace(os.Getenv("CONTROL_PLANE_URL"))
	if baseURL == "" {
		return nil, 0, nil
	}

	interval := defaultRoutesRefreshInterval
	if rawInterval := strings.TrimSpace(os.Getenv("ROUTES_REFRESH_INTERVAL")); rawInterval != "" {
		parsedInterval, err := time.ParseDuration(rawInterval)
		if err != nil {
			return nil, 0, fmt.Errorf("parse ROUTES_REFRESH_INTERVAL: %w", err)
		}

		if parsedInterval <= 0 {
			return nil, 0, errors.New("ROUTES_REFRESH_INTERVAL must be positive")
		}

		interval = parsedInterval
	}

	client, err := controlplane.New(baseURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create control plane client: %w", err)
	}

	return client, interval, nil
}

func runRoutesRefresh(
	ctx context.Context,
	interval time.Duration,
	gatewayUsecase *usecase.Usecase,
	log logger.Logger,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := gatewayUsecase.RefreshRoutes(ctx)
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
