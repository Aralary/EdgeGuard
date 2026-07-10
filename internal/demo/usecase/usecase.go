package usecase

type Usecase struct {
	orderRepository OrderRepository
}

func New(orderRepository OrderRepository) *Usecase {
	return &Usecase{
		orderRepository: orderRepository,
	}
}
