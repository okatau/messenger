package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"testing"
	"time"

	"auth_service/internal/domain"
	"auth_service/internal/service/mocks"
	"auth_service/pkg/token_manager"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var refreshTokenTTL = 30 * 24 * time.Hour
var accesttTokenTTL = 15 * time.Minute

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

	manager, err := token_manager.NewTokenManager(pubPEM, privPEM, accesttTokenTTL, slog.Default())
	require.NoError(t, err)
	return manager
}

func setupAuthSvc(t *testing.T, uMock *mocks.MockUserRepository, sMock *mocks.MockSessionRepository) Auth {
	t.Helper()
	manager := setupTokenManager(t)

	svc := NewAuthService(
		uMock,
		sMock,
		manager,
		slog.Default(),
		refreshTokenTTL,
	)

	return svc
}

func Test_Register(t *testing.T) {
	user := domain.User{
		Username:     "alice",
		Email:        "alice@mail.com",
		PasswordHash: "alice",
	}

	tests := []struct {
		name     string
		setup    func(*mocks.MockUserRepository, *mocks.MockSessionRepository)
		wantName string
		wantErr  error
	}{
		{
			name: "success",
			setup: func(ur *mocks.MockUserRepository, sr *mocks.MockSessionRepository) {
				ur.EXPECT().GetUserByEmail(mock.Anything, "alice@mail.com").Return((*domain.User)(nil), nil)
				ur.EXPECT().CreateUser(mock.Anything, "alice", "alice@mail.com", mock.Anything).Return(&user, nil)
			},
			wantName: "alice",
		},
		{
			name: "user exists",
			setup: func(ur *mocks.MockUserRepository, sr *mocks.MockSessionRepository) {
				ur.EXPECT().GetUserByEmail(mock.Anything, "alice@mail.com").Return(&domain.User{}, nil)
			},
			wantName: "alice",
			wantErr:  domain.ErrUserExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uMock := &mocks.MockUserRepository{}
			sMock := &mocks.MockSessionRepository{}
			tt.setup(uMock, sMock)

			svc := setupAuthSvc(t, uMock, sMock)

			user, err := svc.Register(t.Context(), "alice", "alice@mail.com", "alice")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, user.Username)
			}

		})
	}
}

func Test_Login(t *testing.T) {
	alice := "alice"
	aliceEmail := "alice@mail.com"
	alicePW := "alice"
	alicePWHash, err := bcrypt.GenerateFromPassword([]byte(alicePW), bcrypt.DefaultCost)
	require.NoError(t, err)
	invalidPWHash, err := bcrypt.GenerateFromPassword([]byte("invalid pw hash"), bcrypt.DefaultCost)
	require.NoError(t, err)

	tests := []struct {
		name     string
		email    string
		password string
		setup    func(*mocks.MockUserRepository, *mocks.MockSessionRepository)
		wantErr  error
	}{
		{
			name:     "success",
			email:    aliceEmail,
			password: alicePW,
			setup: func(ur *mocks.MockUserRepository, sr *mocks.MockSessionRepository) {
				ur.EXPECT().GetUserByEmail(mock.Anything, aliceEmail).Return(&domain.User{ID: alice, Username: alice, PasswordHash: string(alicePWHash)}, nil)
				sr.EXPECT().CreateSession(mock.Anything, alice, alice, mock.Anything, mock.Anything).Return(nil)
				sr.EXPECT().DeleteSessionsByUserID(mock.Anything, alice).Return(([]*domain.Session)(nil), nil)
			},
		},
		{
			name:     "user not found",
			email:    aliceEmail,
			password: alicePW,
			setup: func(ur *mocks.MockUserRepository, sr *mocks.MockSessionRepository) {
				ur.EXPECT().GetUserByEmail(mock.Anything, aliceEmail).Return((*domain.User)(nil), nil)
			},
			wantErr: domain.ErrUserNotFound,
		},
		{
			name:     "user forbidden",
			email:    aliceEmail,
			password: alicePW,
			setup: func(ur *mocks.MockUserRepository, sr *mocks.MockSessionRepository) {
				ur.EXPECT().GetUserByEmail(mock.Anything, aliceEmail).Return(&domain.User{ID: alice, Username: alice, PasswordHash: string(invalidPWHash)}, nil)
			},
			wantErr: domain.ErrUserForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uMock := &mocks.MockUserRepository{}
			sMock := &mocks.MockSessionRepository{}

			svc := setupAuthSvc(t, uMock, sMock)
			tt.setup(uMock, sMock)

			session, err := svc.Login(t.Context(), tt.email, tt.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, session)
			}
		})
	}
}

func Test_Refresh(t *testing.T) {
	session := &domain.Session{
		ID:           "session-1",
		UserID:       "alice",
		Username:     "alice",
		RefreshToken: "refresh_token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	tests := []struct {
		name    string
		token   string
		setup   func(*mocks.MockSessionRepository)
		wantErr error
	}{
		{
			name:  "success",
			token: session.RefreshToken,
			setup: func(sr *mocks.MockSessionRepository) {
				sr.EXPECT().GetSessionByToken(mock.Anything, session.RefreshToken).Return(session, nil)
				sr.EXPECT().DeleteSession(mock.Anything, session.RefreshToken).Return((*domain.Session)(nil), nil)
				sr.EXPECT().CreateSession(mock.Anything, session.UserID, session.Username, mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:  "token not found",
			token: session.RefreshToken,
			setup: func(sr *mocks.MockSessionRepository) {
				sr.EXPECT().GetSessionByToken(mock.Anything, session.RefreshToken).Return((*domain.Session)(nil), nil)
			},
			wantErr: domain.ErrTokenNotFound,
		},
		{
			name:  "expired token",
			token: session.RefreshToken,
			setup: func(sr *mocks.MockSessionRepository) {
				sr.EXPECT().GetSessionByToken(mock.Anything, session.RefreshToken).Return(&domain.Session{ExpiresAt: time.Now().Add(-2 * time.Hour)}, nil)
			},
			wantErr: domain.ErrTokenExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uMock := &mocks.MockUserRepository{}
			sMock := &mocks.MockSessionRepository{}

			svc := setupAuthSvc(t, uMock, sMock)
			tt.setup(sMock)

			session, err := svc.Refresh(t.Context(), tt.token)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, session)
			}
		})
	}
}

func Test_Logout(t *testing.T) {
	session := &domain.Session{
		ID:           "session-1",
		UserID:       "alice",
		Username:     "alice",
		RefreshToken: "refresh_token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	tests := []struct {
		name    string
		token   string
		setup   func(*mocks.MockSessionRepository)
		wantErr error
	}{
		{
			name:  "success",
			token: session.RefreshToken,
			setup: func(sr *mocks.MockSessionRepository) {
				sr.EXPECT().GetSessionByToken(mock.Anything, session.RefreshToken).Return(session, nil)
				sr.EXPECT().DeleteSession(mock.Anything, session.RefreshToken).Return((*domain.Session)(nil), nil)
			},
		},
		{
			name:  "token not found",
			token: session.RefreshToken,
			setup: func(sr *mocks.MockSessionRepository) {
				sr.EXPECT().GetSessionByToken(mock.Anything, session.RefreshToken).Return((*domain.Session)(nil), nil)
			},
			wantErr: domain.ErrTokenNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uMock := &mocks.MockUserRepository{}
			sMock := &mocks.MockSessionRepository{}

			svc := setupAuthSvc(t, uMock, sMock)
			tt.setup(sMock)

			err := svc.Logout(t.Context(), tt.token)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, session)
			}
		})
	}
}
