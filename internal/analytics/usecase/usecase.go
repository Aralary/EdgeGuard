package usecase

type Dependencies struct {
	GatewayAccessRepository GatewayAccessRepository
	GatewayStatsRepository  GatewayStatsRepository
}

type Usecase struct {
	gatewayAccessRepository GatewayAccessRepository
	gatewayStatsRepository  GatewayStatsRepository
}

func New(deps Dependencies) *Usecase {
	return &Usecase{
		gatewayAccessRepository: deps.GatewayAccessRepository,
		gatewayStatsRepository:  deps.GatewayStatsRepository,
	}
}
