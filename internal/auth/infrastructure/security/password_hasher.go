package security

import "golang.org/x/crypto/bcrypt"

const DefaultBcryptCost = bcrypt.DefaultCost

type PasswordHasher struct {
	cost int
}

func NewPasswordHasher(cost int) (*PasswordHasher, error) {
	if cost == 0 {
		cost = DefaultBcryptCost
	}

	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, ErrInvalidBcryptCost
	}

	return &PasswordHasher{cost: cost}, nil
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func (h *PasswordHasher) Matches(passwordHash string, password string) bool {
	return bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	) == nil
}
