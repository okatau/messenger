package service

import (
	"auth_service/internal/domain"
	"auth_service/internal/repository"
	loggerPkg "auth_service/pkg/logger"
	"auth_service/pkg/token_manager"
	"context"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	TokenLifetime = 30 * 24 * time.Hour
)

type Auth interface {
	Register(ctx context.Context, name, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*domain.UserInfo, error)
	Refresh(ctx context.Context, token string) (*domain.UserInfo, error)
	Logout(ctx context.Context, token string) error
}

type auth struct {
	userRepo     repository.UserRepository
	sessionRepo  repository.SessionRepository
	tokenManager *token_manager.TokenManager
	logger       *slog.Logger
}

func NewAuth(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	tokenManager *token_manager.TokenManager,
	logger *slog.Logger,
) Auth {
	return &auth{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		tokenManager: tokenManager,
		logger:       logger,
	}
}

func (a *auth) Register(ctx context.Context, name, email, password string) (*domain.User, error) {
	logger := a.logger.With(slog.String("op", "auth.service.register"))

	existing, err := a.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		logger.Error("error reading db", loggerPkg.Err(err))
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrUserExist
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("error encrypting password", loggerPkg.Err(err))
		return nil, err
	}

	user, err := a.userRepo.AddUser(ctx, name, email, string(hash))
	if err != nil {
		logger.Error("error add user to db", loggerPkg.Err(err))
		return nil, err
	}

	return user, nil
}

func (a *auth) Login(ctx context.Context, email, password string) (*domain.UserInfo, error) {
	logger := a.logger.With(slog.String("op", "auth.service.login"))

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

	_, err = a.sessionRepo.RemoveSessionsByUserID(ctx, user.ID)
	if err != nil {
		logger.Error("error deleting session", loggerPkg.Err(err))
		return nil, err
	}

	pair, err := a.generateAndSaveTokens(ctx, user.ID, user.Name)
	if err != nil {
		logger.Error("error generating tokens", loggerPkg.Err(err))
		return nil, err
	}

	return &domain.UserInfo{
		UserID:       user.ID,
		Username:     user.Name,
		RefreshToken: pair.RefreshToken,
		AccessToken:  pair.AccessToken,
	}, nil
}

func (a *auth) Refresh(ctx context.Context, token string) (*domain.UserInfo, error) {
	logger := a.logger.With(slog.String("op", "auth.service.refresh"))

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

	_, err = a.sessionRepo.RemoveSession(ctx, token)
	if err != nil {
		logger.Error("error removing from db", loggerPkg.Err(err))
		return nil, err
	}

	pair, err := a.generateAndSaveTokens(ctx, session.UserID, session.Username)
	if err != nil {
		logger.Error("error generating new tokens", loggerPkg.Err(err))
		return nil, err
	}

	return &domain.UserInfo{
		UserID:       session.UserID,
		Username:     session.Username,
		RefreshToken: pair.RefreshToken,
		AccessToken:  pair.AccessToken,
	}, nil
}

func (a *auth) Logout(ctx context.Context, token string) error {
	logger := a.logger.With(slog.String("op", "auth.service.logout"))

	session, err := a.sessionRepo.GetSessionByToken(ctx, token)
	if err != nil {
		logger.Error("error reading db", loggerPkg.Err(err))
		return err
	}
	if session == nil {
		return domain.ErrTokenNotFound
	}

	_, err = a.sessionRepo.RemoveSession(ctx, token)
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

	expiresAt := time.Now().Add(TokenLifetime)
	if err := a.sessionRepo.AddSession(ctx, userID, username, refreshToken, expiresAt); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
