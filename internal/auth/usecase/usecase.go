package usecase

import "time"

const (
	DefaultRefreshTokenTTL = 30 * 24 * time.Hour
	MinPasswordLength      = 8
	MaxPasswordLength      = 72
)

type Config struct {
	RefreshTokenTTL time.Duration
}

type Usecase struct {
	userRepository         UserRepository
	refreshTokenRepository RefreshTokenRepository
	passwordHasher         PasswordHasher
	tokenService           TokenService
	clock                  Clock
	refreshTokenTTL        time.Duration
}

func New(
	userRepository UserRepository,
	refreshTokenRepository RefreshTokenRepository,
	passwordHasher PasswordHasher,
	tokenService TokenService,
	clock Clock,
	cfg Config,
) *Usecase {
	refreshTokenTTL := cfg.RefreshTokenTTL
	if refreshTokenTTL <= 0 {
		refreshTokenTTL = DefaultRefreshTokenTTL
	}

	if clock == nil {
		clock = systemClock{}
	}

	return &Usecase{
		userRepository:         userRepository,
		refreshTokenRepository: refreshTokenRepository,
		passwordHasher:         passwordHasher,
		tokenService:           tokenService,
		clock:                  clock,
		refreshTokenTTL:        refreshTokenTTL,
	}
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}
