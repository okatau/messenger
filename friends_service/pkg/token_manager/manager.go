package token_manager

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	loggerPkg "friends_service/pkg/logger"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
	verifyOnly bool
	logger     *slog.Logger
}

var ErrVerifyOnly = errors.New("manager only verifies tokens")

func NewTokenManager(publicPem, privatePem []byte, logger *slog.Logger) (*TokenManager, error) {
	var privateKey *rsa.PrivateKey
	verifyOnly := false

	if len(privatePem) == 0 {
		verifyOnly = true
	} else {
		var err error
		privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privatePem)
		if err != nil {
			return nil, err
		}
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicPem)
	if err != nil {
		return nil, err
	}
	return &TokenManager{publicKey: publicKey, privateKey: privateKey, verifyOnly: verifyOnly, logger: logger}, nil
}

func (m *TokenManager) GenerateAccessToken(userID string) (string, error) {
	if m.verifyOnly {
		return "", ErrVerifyOnly
	}
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

func (m *TokenManager) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *TokenManager) VerifyAccessToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	logger := m.logger.With(slog.String("op", "token_manager.TokenManager.VerifyAccessToken"))

	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.publicKey, nil
	})
	if err != nil {
		logger.Error("error parse with claims", loggerPkg.Err(err))
		return nil, err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		logger.Error("error parse token", loggerPkg.Err(err))
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}
