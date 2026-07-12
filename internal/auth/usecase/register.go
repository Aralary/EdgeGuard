package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/auth/domain"
)

type RegisterInput struct {
	Email    string
	Password string
}

func (u *Usecase) Register(ctx context.Context, input RegisterInput) (UserInfo, error) {
	if !isValidPassword(input.Password) {
		return UserInfo{}, ErrInvalidPassword
	}

	passwordHash, err := u.passwordHasher.Hash(input.Password)
	if err != nil {
		return UserInfo{}, err
	}

	user, err := domain.NewUser(input.Email, passwordHash, domain.RoleMember)
	if err != nil {
		return UserInfo{}, err
	}

	created, err := u.userRepository.CreateUser(ctx, user)
	if err != nil {
		return UserInfo{}, err
	}

	return userInfoFromDomain(created), nil
}

func isValidPassword(password string) bool {
	length := len(password)
	return length >= MinPasswordLength && length <= MaxPasswordLength
}
