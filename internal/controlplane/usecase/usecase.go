package usecase

type Usecase struct {
	projectRepository ProjectRepository
	serviceRepository ServiceRepository
	routeRepository   RouteRepository
}

func New(
	projectRepository ProjectRepository,
	serviceRepository ServiceRepository,
	routeRepository RouteRepository,
) *Usecase {
	return &Usecase{
		projectRepository: projectRepository,
		serviceRepository: serviceRepository,
		routeRepository:   routeRepository,
	}
}
