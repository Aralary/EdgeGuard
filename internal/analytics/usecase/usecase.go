package usecase

type Dependencies struct {
	GatewayAccessRepository GatewayAccessRepository
}

type Usecase struct {
	gatewayAccessRepository GatewayAccessRepository
}

func New(deps Dependencies) *Usecase {
	return &Usecase{
		gatewayAccessRepository: deps.GatewayAccessRepository,
	}
}
