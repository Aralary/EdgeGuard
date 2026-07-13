package usecase

type Dependencies struct {
	RouteRepository      RouteRepository
	RouteSource          RouteSource
	APIKeyValidator      APIKeyValidator
	RateLimiter          RateLimiter
	AccessEventPublisher AccessEventPublisher
}

type Usecase struct {
	routeRepository      RouteRepository
	routeSource          RouteSource
	apiKeyValidator      APIKeyValidator
	rateLimiter          RateLimiter
	accessEventPublisher AccessEventPublisher
}

func New(dependencies Dependencies) *Usecase {
	return &Usecase{
		routeRepository:      dependencies.RouteRepository,
		routeSource:          dependencies.RouteSource,
		apiKeyValidator:      dependencies.APIKeyValidator,
		rateLimiter:          dependencies.RateLimiter,
		accessEventPublisher: dependencies.AccessEventPublisher,
	}
}
