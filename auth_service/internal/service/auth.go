package service

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"auth_service/internal/domain"
	"auth_service/internal/repository"
	sl "auth_service/pkg/service_logger"
	"auth_service/pkg/token_manager"
)

const svcName = "auth.service"

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

func New(
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
	l := a.loggerWith(".register")

	exist, err := a.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		l.Error("failed to get user", sl.Err(err))
		return nil, err
	}
	if exist != nil {
		return nil, domain.ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		l.Error("failed to hash password", sl.Err(err))
		return nil, err
	}

	user, err := a.userRepo.CreateUser(ctx, name, email, string(hash))
	if err != nil {
		l.Error("failed to create user", sl.Err(err))
		return nil, err
	}

	return user, nil
}

func (a *auth) Login(ctx context.Context, email, password string) (*domain.AuthSession, error) {
	l := a.loggerWith(".login")

	user, err := a.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		l.Error("failed to get user", sl.Err(err))
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrUserForbidden
	}

	_, err = a.sessionRepo.DeleteSessionsByUserID(ctx, user.ID)
	if err != nil {
		l.Error("failed to delete session", sl.Err(err))
		return nil, err
	}

	pair, err := a.generateAndSaveTokens(ctx, user.ID, user.Username)
	if err != nil {
		l.Error("failed to generate access tokens", sl.Err(err))
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
	l := a.loggerWith(".refresh")

	session, err := a.sessionRepo.GetSessionByToken(ctx, token)
	if err != nil {
		l.Error("failed to get session", sl.Err(err))
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
		l.Error("failed to delete session", sl.Err(err))
		return nil, err
	}

	pair, err := a.generateAndSaveTokens(ctx, session.UserID, session.Username)
	if err != nil {
		l.Error("failed to generate access tokens", sl.Err(err))
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
	l := a.loggerWith(".logout")

	session, err := a.sessionRepo.GetSessionByToken(ctx, token)
	if err != nil {
		l.Error("failed to get session", sl.Err(err))
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

func (a *auth) loggerWith(fnName string) *slog.Logger {
	return a.logger.With("op", svcName+fnName)
}
