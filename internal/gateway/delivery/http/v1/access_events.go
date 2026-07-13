package httpdelivery

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/aralary/edgeguard/internal/platform/events"
	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/labstack/echo/v5"
)

const accessEventMetadataKey = "edgeguard.gateway.access_event_metadata"

type accessEventMetadata struct {
	projectID        string
	routeName        string
	routePathPrefix  string
	apiKeyID         string
	authRequired     bool
	rateLimitEnabled bool
	rateLimited      bool
}

func AccessEvents(publisher AccessEventUsecase, log logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			if req.URL.Path == "/health" {
				return next(c)
			}

			start := time.Now()
			originalResponse := c.Response()
			recorder := newResponseRecorder(originalResponse)

			c.SetResponse(recorder)
			defer c.SetResponse(originalResponse)

			err := next(c)
			completedAt := time.Now().UTC()
			metadata := getAccessEventMetadata(c)
			clientType, clientID := accessEventClient(directClientIP(req), metadata.apiKeyID)

			requestID := req.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = newRequestID()
				req.Header.Set(requestIDHeader, requestID)
			}

			event := events.GatewayAccessEvent{
				SchemaVersion:    events.GatewayAccessEventSchemaVersion,
				EventID:          newAccessEventID(),
				OccurredAt:       completedAt,
				RequestID:        requestID,
				ProjectID:        metadata.projectID,
				RouteName:        metadata.routeName,
				RoutePathPrefix:  metadata.routePathPrefix,
				Method:           req.Method,
				RequestPath:      req.URL.Path,
				StatusCode:       recorder.status,
				DurationMS:       time.Since(start).Milliseconds(),
				ResponseBytes:    int64(recorder.bytes),
				ClientType:       clientType,
				ClientID:         clientID,
				AuthRequired:     metadata.authRequired,
				RateLimitEnabled: metadata.rateLimitEnabled,
				RateLimited:      metadata.rateLimited,
			}

			publishContext := context.WithoutCancel(req.Context())
			if publishErr := publisher.PublishAccessEvent(publishContext, event); publishErr != nil {
				log.Warnf(
					"failed to publish gateway access event: request_id=%s event_id=%s error=%v",
					event.RequestID,
					event.EventID,
					publishErr,
				)
			}

			return err
		}
	}
}

func setAccessEventRoute(c *echo.Context, route domain.Route) {
	metadata := getAccessEventMetadata(c)
	metadata.projectID = route.ProjectID
	metadata.routeName = route.Name
	metadata.routePathPrefix = route.PathPrefix
	metadata.authRequired = route.AuthRequired
	metadata.rateLimitEnabled = route.RateLimit.Enabled
	c.Set(accessEventMetadataKey, metadata)
}

func setAccessEventAPIKey(c *echo.Context, apiKeyID string) {
	metadata := getAccessEventMetadata(c)
	metadata.apiKeyID = apiKeyID
	c.Set(accessEventMetadataKey, metadata)
}

func setAccessEventRateLimited(c *echo.Context) {
	metadata := getAccessEventMetadata(c)
	metadata.rateLimited = true
	c.Set(accessEventMetadataKey, metadata)
}

func getAccessEventMetadata(c *echo.Context) *accessEventMetadata {
	value := c.Get(accessEventMetadataKey)
	metadata, ok := value.(*accessEventMetadata)
	if ok && metadata != nil {
		return metadata
	}

	return &accessEventMetadata{}
}

func accessEventClient(clientIP string, apiKeyID string) (string, string) {
	if apiKeyID != "" {
		return events.ClientTypeAPIKey, apiKeyID
	}

	hash := sha256.Sum256([]byte(clientIP))
	return events.ClientTypeIP, hex.EncodeToString(hash[:])
}

func newAccessEventID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	return hex.EncodeToString(buffer)
}
