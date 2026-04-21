package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func Test_CreateUser(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	uRepo := NewUserRepository(pool)
	ctx := context.Background()

	bobName := "bob"
	bobEmail := "bob@mail.com"
	bobPasswordHash := "bob"

	user, err := uRepo.CreateUser(ctx, bobName, bobEmail, bobPasswordHash)
	require.NoError(t, err)
	require.NotNil(t, user, "user is nil")
}

func Test_CreateUser_ExistUser(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	uRepo := NewUserRepository(pool)
	ctx := context.Background()
	_, err := uRepo.CreateUser(ctx, aliceName, aliceEmail, alicePasswordHash)

	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
}

func Test_DeleteUser(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	uRepo := NewUserRepository(pool)
	ctx := context.Background()
	user, err := uRepo.DeleteUser(ctx, aliceID)
	require.NoError(t, err)
	require.Equal(t, user.Username, aliceName)
}

func Test_DeleteUser_NoRows(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	uRepo := NewUserRepository(pool)
	ctx := context.Background()

	user, err := uRepo.DeleteUser(ctx, aliceName)
	log.Println(user, err)
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
