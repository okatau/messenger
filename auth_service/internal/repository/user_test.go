package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var (
	bobID    = uuid.NewString()
	bobName  = "bob"
	bobMail  = "bob@mail.com"
	bobPW    = "bob"
	bobPG, _ = bcrypt.GenerateFromPassword([]byte(alicePW), bcrypt.DefaultCost)
)

func Test_CreateUser(t *testing.T) {
	pool := startPostgres(t)

	t.Run("success", func(t *testing.T) {
		uRepo := NewUserRepository(pool)

		user, err := uRepo.CreateUser(t.Context(), bobName, bobMail, bobPW)
		require.NoError(t, err)
		require.NotNil(t, user)
	})

	t.Run("user exists", func(t *testing.T) {
		uRepo := NewUserRepository(pool)
		_, err := uRepo.CreateUser(t.Context(), aliceName, aliceMail, alicePW)

		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		assert.Equal(t, pgErr.Code, "23505")
	})
}

func Test_DeleteUser(t *testing.T) {
	pool := startPostgres(t)
	t.Run("success", func(t *testing.T) {
		uRepo := NewUserRepository(pool)
		user, err := uRepo.DeleteUser(t.Context(), aliceID)
		require.NoError(t, err)
		require.Equal(t, user.Username, aliceName)
	})

	t.Run("user does not exist", func(t *testing.T) {
		uRepo := NewUserRepository(pool)
		user, err := uRepo.DeleteUser(t.Context(), bobID)
		require.Nil(t, user)
		require.Nil(t, err)
	})
}
