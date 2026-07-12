package usecase

type Usecase struct {
	routeRepository RouteRepository
	routeSource     RouteSource
}

func New(routeRepository RouteRepository, routeSource RouteSource) *Usecase {
	return &Usecase{
		routeRepository: routeRepository,
		routeSource:     routeSource,
	}
}
