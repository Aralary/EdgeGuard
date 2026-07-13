package usecase

import "time"

type Dependencies struct {
	Publisher   Publisher
	IDGenerator IDGenerator
	Clock       Clock
}

type Usecase struct {
	publisher   Publisher
	idGenerator IDGenerator
	clock       Clock
}

func New(dependencies Dependencies) *Usecase {
	clock := dependencies.Clock
	if clock == nil {
		clock = realClock{}
	}

	return &Usecase{
		publisher:   dependencies.Publisher,
		idGenerator: dependencies.IDGenerator,
		clock:       clock,
	}
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}
