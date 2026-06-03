package middleware

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/go-redis/redis_rate/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func startRedis(t *testing.T) *redis.Client {
	t.Helper()

	ctr, err := tcredis.Run(t.Context(), "redis:7-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { ctr.Terminate(t.Context()) })

	dsn, err := ctr.ConnectionString(t.Context())
	require.NoError(t, err)

	opt, err := redis.ParseURL(dsn)
	require.NoError(t, err)

	return redis.NewClient(opt)
}

func initRL(t *testing.T) *redis_rate.Limiter {
	t.Helper()

	rdb := startRedis(t)
	rl := redis_rate.NewLimiter(rdb)

	return rl
}

func Test_RateLimitByIP(t *testing.T) {
	addr1 := "10.0.0.1:0001"
	addr2 := "10.0.0.2:0001"

	next := func(c *echo.Context) error {
		return c.JSON(http.StatusOK, nil)
	}

	t.Run("successfully lock by ip", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/case_1", "")
		rl := initRL(t)

		c.Request().RemoteAddr = addr1

		for range 3 {
			err := RateLimitByIP(rl, slog.Default(), 3)(next)(c)
			require.NoError(t, err)
		}

		err := RateLimitByIP(rl, slog.Default(), 3)(next)(c)

		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, http.StatusTooManyRequests, echoErr.Code)
	})

	t.Run("different addresses no lock", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/case_2", "")
		rl := initRL(t)

		c.Request().RemoteAddr = addr1

		for range 3 {
			err := RateLimitByIP(rl, slog.Default(), 3)(next)(c)
			require.NoError(t, err)
		}

		err := RateLimitByIP(rl, slog.Default(), 3)(next)(c)

		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, http.StatusTooManyRequests, echoErr.Code)

		c.Request().RemoteAddr = addr2
		err = RateLimitByIP(rl, slog.Default(), 3)(next)(c)
		require.NoError(t, err)
	})

	t.Run("get ip from header", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/case_2", "")
		rl := initRL(t)

		c.Request().Header.Set("X-Real-IP", "10.0.0.1")

		for range 3 {
			err := RateLimitByIP(rl, slog.Default(), 3)(next)(c)
			require.NoError(t, err)
		}

		err := RateLimitByIP(rl, slog.Default(), 3)(next)(c)

		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, http.StatusTooManyRequests, echoErr.Code)
	})

	t.Run("invalid ip header", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/case_4", "")
		rl := initRL(t)

		c.Request().Header.Set("X-Real-IP", "some_bad_IP")

		err := RateLimitByIP(rl, slog.Default(), 3)(next)(c)
		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, http.StatusBadRequest, echoErr.Code)
	})

	t.Run("invalid IP", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/case_4", "")
		rl := initRL(t)

		c.Request().RemoteAddr = "some_bad_IP"

		err := RateLimitByIP(rl, slog.Default(), 3)(next)(c)
		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, http.StatusBadRequest, echoErr.Code)
	})

	t.Run("exceed limits", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/case_5", "")
		rl := initRL(t)

		c.Request().RemoteAddr = addr1

		RateLimitByIP(rl, slog.Default(), 1)(next)(c)
		err := RateLimitByIP(rl, slog.Default(), 1)(next)(c)

		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, http.StatusTooManyRequests, echoErr.Code)
	})
}

func Test_RateLimitByID(t *testing.T) {
	next := func(c *echo.Context) error {
		return c.JSON(http.StatusOK, nil)
	}

	aliceID := uuid.NewString()
	bobID := uuid.NewString()

	t.Run("successfully lock", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/case_1", "")
		rl := initRL(t)

		c.Set("userID", aliceID)

		for range 3 {
			err := RateLimitByUser(rl, slog.Default(), 3)(next)(c)
			require.NoError(t, err)
		}

		err := RateLimitByUser(rl, slog.Default(), 3)(next)(c)

		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, http.StatusTooManyRequests, echoErr.Code)
	})

	t.Run("different users no lock", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/case_2", "")
		rl := initRL(t)

		c.Set("userID", aliceID)

		for range 3 {
			err := RateLimitByUser(rl, slog.Default(), 3)(next)(c)
			require.NoError(t, err)
		}

		err := RateLimitByUser(rl, slog.Default(), 3)(next)(c)

		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, http.StatusTooManyRequests, echoErr.Code)

		c.Set("userID", bobID)
		err = RateLimitByUser(rl, slog.Default(), 3)(next)(c)
		require.NoError(t, err)
	})

	t.Run("error exceed limits", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/case_5", "")
		rl := initRL(t)

		c.Set("userID", aliceID)

		RateLimitByUser(rl, slog.Default(), 1)(next)(c)
		err := RateLimitByUser(rl, slog.Default(), 1)(next)(c)

		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, http.StatusTooManyRequests, echoErr.Code)
	})
}
