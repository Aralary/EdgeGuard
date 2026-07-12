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

type Dependencies struct {
	UserRepository         UserRepository
	RefreshTokenRepository RefreshTokenRepository
	APIKeyRepository       APIKeyRepository
	PasswordHasher         PasswordHasher
	TokenService           TokenService
	APIKeyService          APIKeyService
	Clock                  Clock
}

type Usecase struct {
	userRepository         UserRepository
	refreshTokenRepository RefreshTokenRepository
	apiKeyRepository       APIKeyRepository
	passwordHasher         PasswordHasher
	tokenService           TokenService
	apiKeyService          APIKeyService
	clock                  Clock
	refreshTokenTTL        time.Duration
}

func New(dependencies Dependencies, cfg Config) *Usecase {
	refreshTokenTTL := cfg.RefreshTokenTTL
	if refreshTokenTTL <= 0 {
		refreshTokenTTL = DefaultRefreshTokenTTL
	}

	clock := dependencies.Clock
	if clock == nil {
		clock = systemClock{}
	}

	return &Usecase{
		userRepository:         dependencies.UserRepository,
		refreshTokenRepository: dependencies.RefreshTokenRepository,
		apiKeyRepository:       dependencies.APIKeyRepository,
		passwordHasher:         dependencies.PasswordHasher,
		tokenService:           dependencies.TokenService,
		apiKeyService:          dependencies.APIKeyService,
		clock:                  clock,
		refreshTokenTTL:        refreshTokenTTL,
	}
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}
