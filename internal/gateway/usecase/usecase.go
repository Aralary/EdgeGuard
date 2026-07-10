package usecase

type Usecase struct {
	routeRepository RouteRepository
}

func New(routeRepository RouteRepository) *Usecase {
	return &Usecase{
		routeRepository: routeRepository,
	}
}
