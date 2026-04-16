package token_manager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const aliceID = "00000000-0000-0000-0000-0000000a11c3"

var (
	publicPEM  string
	privatePEM string
	manager    *TokenManager
)

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

	claims, err := manager.VerifyAccessToken(token)
	if err != nil {
		t.Errorf("errorf verify token %v", err)
	}

	assertEqual(t, claims.Subject, aliceID)
}

func Test_VerifyAccessToken_Errors(t *testing.T) {
	_, err := manager.VerifyAccessToken("not.a.valid.jwt.token")
	if err == nil {
		t.Error("must fail")
	}
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v want %v", got, want)
	}
}

func assertNotEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got == want {
		t.Errorf("got %v want %v", got, want)
	}
}

// generateRSAKeyPair генерирует новую RSA пару ключей в PEM формате для тестов.
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

// --- NewTokenManager ---

func TestNewTokenManager_Success(t *testing.T) {
	pubKey, privKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("generateRSAKeyPair: %v", err)
	}

	m, err := NewTokenManager(pubKey, privKey, slog.Default())
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	if m.verifyOnly {
		t.Error("expected verifyOnly=false when private key provided")
	}
}

func TestNewTokenManager_VerifyOnly(t *testing.T) {
	pubKey, _, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("generateRSAKeyPair: %v", err)
	}

	m, err := NewTokenManager(pubKey, nil, slog.Default())
	if err != nil {
		t.Fatalf("NewTokenManager verifyOnly: %v", err)
	}
	if !m.verifyOnly {
		t.Error("expected verifyOnly=true when no private key provided")
	}
}

func TestNewTokenManager_InvalidPublicKey(t *testing.T) {
	_, privKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("generateRSAKeyPair: %v", err)
	}

	_, err = NewTokenManager([]byte("not-a-pem"), privKey, slog.Default())
	if err == nil {
		t.Error("expected error for invalid public key PEM")
	}
}

func TestNewTokenManager_InvalidPrivateKey(t *testing.T) {
	pubKey, _, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("generateRSAKeyPair: %v", err)
	}

	_, err = NewTokenManager(pubKey, []byte("not-a-pem"), slog.Default())
	if err == nil {
		t.Error("expected error for invalid private key PEM")
	}
}

// --- GenerateAccessToken ---

func TestGenerateAccessToken_Success(t *testing.T) {
	token, err := manager.GenerateAccessToken(aliceID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestGenerateAccessToken_ContainsSubject(t *testing.T) {

	token, err := manager.GenerateAccessToken(aliceID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := manager.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}
	assertEqual(t, claims.Subject, aliceID)
}

func TestGenerateAccessToken_VerifyOnly(t *testing.T) {
	pubKey, _, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("generateRSAKeyPair: %v", err)
	}

	m, err := NewTokenManager(pubKey, nil, slog.Default())
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}

	_, err = m.GenerateAccessToken(aliceID)
	if !errors.Is(err, ErrVerifyOnly) {
		t.Errorf("expected ErrVerifyOnly, got %v", err)
	}
}

// --- GenerateRefreshToken ---

func TestGenerateRefreshToken_Format(t *testing.T) {
	token, err := manager.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	// 32 random bytes → 64 hex characters
	if len(token) != 64 {
		t.Errorf("expected token length 64, got %d", len(token))
	}
	for _, c := range token {
		if !('0' <= c && c <= '9') && !('a' <= c && c <= 'f') {
			t.Errorf("token contains non-hex character: %c", c)
			break
		}
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	t1, err := manager.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken first: %v", err)
	}
	t2, err := manager.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken second: %v", err)
	}
	assertNotEqual(t, t1, t2)
}

// --- VerifyAccessToken ---

func TestVerifyAccessToken_ExpiredToken(t *testing.T) {
	_, privKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("generateRSAKeyPair: %v", err)
	}
	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKey)
	if err != nil {
		t.Fatalf("parse private key: %v", err)
	}

	claims := jwt.RegisteredClaims{
		Subject:   aliceID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(rsaKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = manager.VerifyAccessToken(tokenStr)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestVerifyAccessToken_WrongKey(t *testing.T) {
	pubKey, privKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("generateRSAKeyPair: %v", err)
	}

	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKey)
	if err != nil {
		t.Fatalf("parse generated private key: %v", err)
	}

	claims := jwt.RegisteredClaims{
		Subject:   aliceID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(rsaKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	// менеджер с другим публичным ключом — верификация должна упасть
	m, err := NewTokenManager(pubKey, privKey, slog.Default())
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	_ = m

	_, err = manager.VerifyAccessToken(tokenStr)
	if err == nil {
		t.Error("expected error when token signed with different key")
	}
}

func TestVerifyAccessToken_WrongSigningMethod(t *testing.T) {
	secret := []byte("hmac-secret")
	claims := jwt.RegisteredClaims{
		Subject:   aliceID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign HMAC token: %v", err)
	}

	_, err = manager.VerifyAccessToken(tokenStr)
	if err == nil {
		t.Error("expected error for token signed with HMAC instead of RSA")
	}
}
