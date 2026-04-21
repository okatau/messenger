package token_manager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const aliceID = "00000000-0000-0000-0000-0000000a11c3"

var (
	manager *TokenManager
)

func generateRSAKeyPair() (pubPEM, privPEM []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	privPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return
}

func TestMain(m *testing.M) {
	publicPEM, privatePEM, err := generateRSAKeyPair()
	if err != nil {
		log.Fatalf("error generate rsa key pair %v", err)
	}

	manager, err = NewTokenManager(publicPEM, privatePEM, slog.Default())
	if err != nil {
		log.Fatalf("error Create token manager %v", err)
	}

	code := m.Run()
	os.Exit(code)
}

func Test_VerifyAccessToken_Success(t *testing.T) {
	token, err := manager.GenerateAccessToken(aliceID)
	require.NoError(t, err)

	claims, err := manager.VerifyAccessToken(token)
	require.NoError(t, err)

	assert.Equal(t, claims.Subject, aliceID)
}

func Test_VerifyAccessToken_Errors(t *testing.T) {
	_, err := manager.VerifyAccessToken("not.a.valid.jwt.token")
	require.Error(t, err)
}

func TestNewTokenManager_Success(t *testing.T) {
	pubKey, privKey, err := generateRSAKeyPair()
	require.NoError(t, err)

	m, err := NewTokenManager(pubKey, privKey, slog.Default())
	require.NoError(t, err)
	assert.False(t, m.verifyOnly)
}

func TestNewTokenManager_VerifyOnly(t *testing.T) {
	pubKey, _, err := generateRSAKeyPair()
	require.NoError(t, err)

	m, err := NewTokenManager(pubKey, nil, slog.Default())
	require.NoError(t, err)
	assert.True(t, m.verifyOnly)
}

func TestNewTokenManager_InvalidPublicKey(t *testing.T) {
	_, privKey, err := generateRSAKeyPair()
	require.NoError(t, err)

	_, err = NewTokenManager([]byte("not-a-pem"), privKey, slog.Default())
	require.Error(t, err)
}

func TestNewTokenManager_InvalidPrivateKey(t *testing.T) {
	pubKey, _, err := generateRSAKeyPair()
	require.NoError(t, err)

	_, err = NewTokenManager(pubKey, []byte("not-a-pem"), slog.Default())
	require.Error(t, err)
}

func TestGenerateAccessToken_Success(t *testing.T) {
	token, err := manager.GenerateAccessToken(aliceID)
	require.NoError(t, err)
	assert.NotEqual(t, token, "")
}

func TestGenerateAccessToken_ContainsSubject(t *testing.T) {
	token, err := manager.GenerateAccessToken(aliceID)
	require.NoError(t, err)

	claims, err := manager.VerifyAccessToken(token)
	require.NoError(t, err)

	assert.Equal(t, aliceID, claims.Subject)
}

func TestGenerateAccessToken_VerifyOnly(t *testing.T) {
	pubKey, _, err := generateRSAKeyPair()
	require.NoError(t, err)

	m, err := NewTokenManager(pubKey, nil, slog.Default())
	require.NoError(t, err)

	_, err = m.GenerateAccessToken(aliceID)
	require.ErrorIs(t, err, ErrVerifyOnly)
}

func TestGenerateRefreshToken_Format(t *testing.T) {
	token, err := manager.GenerateRefreshToken()
	require.NoError(t, err)

	assert.Equal(t, len(token), 64)
	for _, c := range token {
		if !('0' <= c && c <= '9') && !('a' <= c && c <= 'f') {
			t.Errorf("token contains non-hex character: %c", c)
			break
		}
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	t1, err := manager.GenerateRefreshToken()
	require.NoError(t, err)

	t2, err := manager.GenerateRefreshToken()
	require.NoError(t, err)

	assert.NotEqual(t, t1, t2)
}

func TestVerifyAccessToken_ExpiredToken(t *testing.T) {
	_, privKey, err := generateRSAKeyPair()
	require.NoError(t, err)

	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKey)
	require.NoError(t, err)

	claims := jwt.RegisteredClaims{
		Subject:   aliceID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(rsaKey)
	require.NoError(t, err)

	_, err = manager.VerifyAccessToken(tokenStr)
	require.Error(t, err)
}

func TestVerifyAccessToken_WrongKey(t *testing.T) {
	pubKey, privKey, err := generateRSAKeyPair()
	require.NoError(t, err)

	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKey)
	require.NoError(t, err)

	claims := jwt.RegisteredClaims{
		Subject:   aliceID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(rsaKey)
	require.NoError(t, err)

	m, err := NewTokenManager(pubKey, privKey, slog.Default())
	require.NoError(t, err)
	_ = m

	_, err = manager.VerifyAccessToken(tokenStr)
	require.Error(t, err)
}

func TestVerifyAccessToken_WrongSigningMethod(t *testing.T) {
	secret := []byte("hmac-secret")
	claims := jwt.RegisteredClaims{
		Subject:   aliceID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(secret)
	require.NoError(t, err)

	_, err = manager.VerifyAccessToken(tokenStr)
	require.Error(t, err)
}
