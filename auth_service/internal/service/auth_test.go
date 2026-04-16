package service

import (
	"auth_service/internal/domain"
	"auth_service/internal/repository"
	"auth_service/pkg/token_manager"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type userRepoMock struct{ mock.Mock }
type sessionRepoMock struct{ mock.Mock }

func (sr *sessionRepoMock) GetSessionByToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	args := sr.Called(ctx, refreshToken)
	return args.Get(0).(*domain.Session), args.Error(1)
}
func (sr *sessionRepoMock) AddSession(ctx context.Context, userID, name, refreshToken string, expiresAt time.Time) error {
	args := sr.Called(ctx, userID, name, refreshToken)
	return args.Error(0)
}
func (sr *sessionRepoMock) RemoveSession(ctx context.Context, refreshToken string) (*domain.Session, error) {
	args := sr.Called(ctx, refreshToken)
	return args.Get(0).(*domain.Session), args.Error(1)
}
func (sr *sessionRepoMock) RemoveSessionsByUserID(ctx context.Context, userID string) ([]*domain.Session, error) {
	args := sr.Called(ctx, userID)
	return args.Get(0).([]*domain.Session), args.Error(1)
}

func (ur *userRepoMock) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	args := ur.Called(ctx, id)
	return args.Get(0).(*domain.User), args.Error(1)
}
func (ur *userRepoMock) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := ur.Called(ctx, email)
	return args.Get(0).(*domain.User), args.Error(1)
}
func (ur *userRepoMock) AddUser(ctx context.Context, name, email, passwordHash string) (*domain.User, error) {
	args := ur.Called(ctx, name, email)
	return args.Get(0).(*domain.User), args.Error(1)
}
func (ur *userRepoMock) RemoveUser(ctx context.Context, id string) (*domain.User, error) {
	args := ur.Called(ctx, id)
	return args.Get(0).(*domain.User), args.Error(1)
}

var _ repository.UserRepository = (*userRepoMock)(nil)
var _ repository.SessionRepository = (*sessionRepoMock)(nil)

func setupTokenManager(t *testing.T) *token_manager.TokenManager {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	manager, err := token_manager.NewTokenManager(pubPEM, privPEM, slog.Default())
	require.NoError(t, err)
	return manager
}

func setupAuthSvc(t *testing.T, uMock *userRepoMock, sMock *sessionRepoMock) Auth {
	t.Helper()
	manager := setupTokenManager(t)

	svc := NewAuth(
		uMock,
		sMock,
		manager,
		slog.Default(),
	)

	return svc
}

func Test_Register(t *testing.T) {
	user := domain.User{
		Name:         "alice",
		Email:        "alice@mail.com",
		PasswordHash: "alice",
	}

	tests := []struct {
		name     string
		setup    func(*userRepoMock, *sessionRepoMock)
		wantName string
		wantErr  error
	}{
		{
			name: "success",
			setup: func(ur *userRepoMock, sr *sessionRepoMock) {
				ur.On("GetUserByEmail", mock.Anything, "alice@mail.com").Return((*domain.User)(nil), nil)
				ur.On("AddUser", mock.Anything, "alice", "alice@mail.com").Return(&user, nil)
			},
			wantName: "alice",
		},
		{
			name: "user exists",
			setup: func(ur *userRepoMock, sr *sessionRepoMock) {
				ur.On("GetUserByEmail", mock.Anything, "alice@mail.com").Return(&domain.User{}, nil)
			},
			wantName: "alice",
			wantErr:  domain.ErrUserExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uMock := &userRepoMock{}
			sMock := &sessionRepoMock{}
			tt.setup(uMock, sMock)

			svc := setupAuthSvc(t, uMock, sMock)

			user, err := svc.Register(t.Context(), "alice", "alice@mail.com", "alice")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, user.Name)
			}

		})
	}
}
