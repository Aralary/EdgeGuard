package usecase

type Usecase struct {
	routeRepository RouteRepository
	routeSource     RouteSource
	apiKeyValidator APIKeyValidator
}

func New(
	routeRepository RouteRepository,
	routeSource RouteSource,
	apiKeyValidator APIKeyValidator,
) *Usecase {
	return &Usecase{
		routeRepository: routeRepository,
		routeSource:     routeSource,
		apiKeyValidator: apiKeyValidator,
	}
}
