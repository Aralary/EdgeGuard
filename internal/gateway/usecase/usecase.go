package usecase

type Usecase struct {
	routeRepository RouteRepository
	routeSource     RouteSource
	apiKeyValidator APIKeyValidator
	rateLimiter     RateLimiter
}

func New(
	routeRepository RouteRepository,
	routeSource RouteSource,
	apiKeyValidator APIKeyValidator,
	rateLimiter RateLimiter,
) *Usecase {
	return &Usecase{
		routeRepository: routeRepository,
		routeSource:     routeSource,
		apiKeyValidator: apiKeyValidator,
		rateLimiter:     rateLimiter,
	}
}
