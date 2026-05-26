package token_manager

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenManager issues and verifies RS256 JWT access tokens and generates opaque refresh tokens.
type TokenManager struct {
	publicKey      *rsa.PublicKey
	privateKey     *rsa.PrivateKey
	accessTokenTTL time.Duration
	verifyOnly     bool
	logger         *slog.Logger
}

const refreshTokenSize = 32

// ErrVerifyOnly is returned when an operation that requires the private key is
// called on a verify-only instance (created without a private PEM).
var ErrVerifyOnly = errors.New("manager only verifies tokens")

// NewTokenManager creates a TokenManager from RSA PEM keys.
// Pass an empty privatePem to create a verify-only instance (GenerateAccessToken will return ErrVerifyOnly).
func NewTokenManager(publicPem, privatePem []byte, accessTokenTTL time.Duration, logger *slog.Logger) (*TokenManager, error) {
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
	return &TokenManager{
		publicKey:      publicKey,
		privateKey:     privateKey,
		verifyOnly:     verifyOnly,
		logger:         logger,
		accessTokenTTL: accessTokenTTL,
	}, nil
}

// GenerateAccessToken signs a new RS256 JWT with the given userID as the subject.
// Returns ErrVerifyOnly if the manager was initialized without a private key.
func (m *TokenManager) GenerateAccessToken(userID string) (string, error) {
	if m.verifyOnly {
		return "", ErrVerifyOnly
	}
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// GenerateRefreshToken returns a cryptographically random 64-character hex string.
func (m *TokenManager) GenerateRefreshToken() (string, error) {
	b := make([]byte, refreshTokenSize)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// VerifyAccessToken parses and validates the given JWT access token,
// returning the claims on success.
func (m *TokenManager) VerifyAccessToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	logger := m.logger.With(slog.String("op", "token_manager.TokenManager.VerifyAccessToken"))

	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.publicKey, nil
	})
	if err != nil {
		logger.Error(
			"error parse with claims",
			slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			})
		return nil, err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		logger.Error("error parse token")
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}
