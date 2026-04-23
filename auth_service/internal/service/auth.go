package service

import (
	"context"
	"log/slog"
	"time"

	"auth_service/internal/domain"
	"auth_service/internal/repository"
	loggerPkg "auth_service/pkg/logger"
	"auth_service/pkg/token_manager"

	"golang.org/x/crypto/bcrypt"
)

type Auth interface {
	Register(ctx context.Context, name, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*domain.AuthSession, error)
	Refresh(ctx context.Context, token string) (*domain.AuthSession, error)
	Logout(ctx context.Context, token string) error
}

type auth struct {
	userRepo        repository.UserRepository
	sessionRepo     repository.SessionRepository
	tokenManager    *token_manager.TokenManager
	logger          *slog.Logger
	refreshTokenTTL time.Duration
}

func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	tokenManager *token_manager.TokenManager,
	logger *slog.Logger,
	refreshTokenTTL time.Duration,
) Auth {
	return &auth{
		userRepo:        userRepo,
		sessionRepo:     sessionRepo,
		tokenManager:    tokenManager,
		logger:          logger,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (a *auth) Register(ctx context.Context, name, email, password string) (*domain.User, error) {
	const op = "auth.service.register"
	logger := a.logger.With(slog.String("op", op))

	existing, err := a.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		logger.Error("error reading db", loggerPkg.Err(err))
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("error encrypting password", loggerPkg.Err(err))
		return nil, err
	}

	user, err := a.userRepo.CreateUser(ctx, name, email, string(hash))
	if err != nil {
		logger.Error("error add user to db", loggerPkg.Err(err))
		return nil, err
	}

	return user, nil
}

func (a *auth) Login(ctx context.Context, email, password string) (*domain.AuthSession, error) {
	const op = "auth.service.login"
	logger := a.logger.With(slog.String("op", op))

	user, err := a.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		logger.Error("error reading db", loggerPkg.Err(err))
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrUserForbidden
	}

	_, err = a.sessionRepo.DeleteSessionsByUserID(ctx, user.ID)
	if err != nil {
		logger.Error("error deleting session", loggerPkg.Err(err))
		return nil, err
	}

	pair, err := a.generateAndSaveTokens(ctx, user.ID, user.Username)
	if err != nil {
		logger.Error("error generating tokens", loggerPkg.Err(err))
		return nil, err
	}

	return &domain.AuthSession{
		UserID:       user.ID,
		Username:     user.Username,
		RefreshToken: pair.RefreshToken,
		AccessToken:  pair.AccessToken,
	}, nil
}

func (a *auth) Refresh(ctx context.Context, token string) (*domain.AuthSession, error) {
	const op = "auth.service.refresh"
	logger := a.logger.With(slog.String("op", op))

	session, err := a.sessionRepo.GetSessionByToken(ctx, token)
	if err != nil {
		logger.Error("error reading db", loggerPkg.Err(err))
		return nil, err
	}
	if session == nil {
		return nil, domain.ErrTokenNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, domain.ErrTokenExpired
	}

	_, err = a.sessionRepo.DeleteSession(ctx, token)
	if err != nil {
		logger.Error("error removing from db", loggerPkg.Err(err))
		return nil, err
	}

	pair, err := a.generateAndSaveTokens(ctx, session.UserID, session.Username)
	if err != nil {
		logger.Error("error generating new tokens", loggerPkg.Err(err))
		return nil, err
	}

	return &domain.AuthSession{
		UserID:       session.UserID,
		Username:     session.Username,
		RefreshToken: pair.RefreshToken,
		AccessToken:  pair.AccessToken,
	}, nil
}

func (a *auth) Logout(ctx context.Context, token string) error {
	const op = "auth.service.logout"
	logger := a.logger.With(slog.String("op", op))

	session, err := a.sessionRepo.GetSessionByToken(ctx, token)
	if err != nil {
		logger.Error("error reading db", loggerPkg.Err(err))
		return err
	}
	if session == nil {
		return domain.ErrTokenNotFound
	}

	_, err = a.sessionRepo.DeleteSession(ctx, token)
	return err
}

func (a *auth) generateAndSaveTokens(ctx context.Context, userID, username string) (*domain.TokenPair, error) {
	accessToken, err := a.tokenManager.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := a.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(a.refreshTokenTTL)
	if err := a.sessionRepo.CreateSession(ctx, userID, username, refreshToken, expiresAt); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
