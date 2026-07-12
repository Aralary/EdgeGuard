package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type fakeAPIKeyValidator struct {
	principal domain.APIKeyPrincipal
	err       error
	calls     int
}

func (v *fakeAPIKeyValidator) ValidateAPIKey(
	context.Context,
	string,
) (domain.APIKeyPrincipal, error) {
	v.calls++
	return v.principal, v.err
}

func TestAuthorizeRouteAllowsPublicRoute(t *testing.T) {
	validator := &fakeAPIKeyValidator{err: errors.New("must not be called")}
	uc := New(fakeRouteRepository{}, nil, validator, nil)

	principal, err := uc.AuthorizeRoute(context.Background(), domain.Route{AuthRequired: false}, "")
	if err != nil {
		t.Fatalf("AuthorizeRoute() error = %v", err)
	}
	if principal != (domain.APIKeyPrincipal{}) {
		t.Fatalf("principal = %+v, want zero value", principal)
	}
	if validator.calls != 0 {
		t.Fatalf("validator calls = %d, want 0", validator.calls)
	}
}

func TestAuthorizeRouteRequiresAPIKey(t *testing.T) {
	uc := New(fakeRouteRepository{}, nil, &fakeAPIKeyValidator{}, nil)

	_, err := uc.AuthorizeRoute(context.Background(), domain.Route{AuthRequired: true}, " ")
	if !errors.Is(err, domain.ErrAPIKeyRequired) {
		t.Fatalf("error = %v, want ErrAPIKeyRequired", err)
	}
}

func TestAuthorizeRouteAllowsMatchingProject(t *testing.T) {
	validator := &fakeAPIKeyValidator{
		principal: domain.APIKeyPrincipal{APIKeyID: "key-1", ProjectID: "project-1"},
	}
	uc := New(fakeRouteRepository{}, nil, validator, nil)

	principal, err := uc.AuthorizeRoute(context.Background(), domain.Route{
		ProjectID:    "project-1",
		AuthRequired: true,
	}, "eg_live_key")
	if err != nil {
		t.Fatalf("AuthorizeRoute() error = %v", err)
	}
	if principal != validator.principal {
		t.Fatalf("principal = %+v, want %+v", principal, validator.principal)
	}
}

func TestAuthorizeRouteRejectsDifferentProject(t *testing.T) {
	validator := &fakeAPIKeyValidator{
		principal: domain.APIKeyPrincipal{APIKeyID: "key-1", ProjectID: "project-2"},
	}
	uc := New(fakeRouteRepository{}, nil, validator, nil)

	_, err := uc.AuthorizeRoute(context.Background(), domain.Route{
		ProjectID:    "project-1",
		AuthRequired: true,
	}, "eg_live_key")
	if !errors.Is(err, domain.ErrAPIKeyProjectMismatch) {
		t.Fatalf("error = %v, want ErrAPIKeyProjectMismatch", err)
	}
}

func TestAuthorizeRoutePropagatesInvalidKey(t *testing.T) {
	validator := &fakeAPIKeyValidator{err: domain.ErrInvalidAPIKey}
	uc := New(fakeRouteRepository{}, nil, validator, nil)

	_, err := uc.AuthorizeRoute(context.Background(), domain.Route{
		ProjectID:    "project-1",
		AuthRequired: true,
	}, "invalid")
	if !errors.Is(err, domain.ErrInvalidAPIKey) {
		t.Fatalf("error = %v, want ErrInvalidAPIKey", err)
	}
}

func TestAuthorizeRouteFailsClosedWithoutValidator(t *testing.T) {
	uc := New(fakeRouteRepository{}, nil, nil, nil)

	_, err := uc.AuthorizeRoute(context.Background(), domain.Route{
		ProjectID:    "project-1",
		AuthRequired: true,
	}, "eg_live_key")
	if !errors.Is(err, domain.ErrAPIKeyValidatorNotConfigured) {
		t.Fatalf("error = %v, want ErrAPIKeyValidatorNotConfigured", err)
	}
}
