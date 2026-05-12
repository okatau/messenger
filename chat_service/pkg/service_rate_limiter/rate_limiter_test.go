package service_rate_limiter

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-redis/redis_rate/v10"
	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func startRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()

	ctx := context.Background()
	ctr, err := tcredis.Run(ctx,
		"redis:7-alpine",
		tcredis.WithSnapshotting(10, 1),
		tcredis.WithLogLevel(tcredis.LogLevelVerbose),
	)
	require.NoError(t, err)

	dsn, err := ctr.ConnectionString(ctx)
	require.NoError(t, err)

	opt, err := redis.ParseURL(dsn)
	require.NoError(t, err)

	redisClient := redis.NewClient(opt)

	return redisClient, func() {
		redisClient.Close()
		ctr.Terminate(ctx)
	}
}

func newContext(method, target, body string) (*echo.Echo, *echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, reqBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	return e, e.NewContext(req, rec), rec
}

func okHandler(c *echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func Test_RateLimitByIP_Allowed(t *testing.T) {
	rdb, cleanup := startRedis(t)
	defer cleanup()

	rl := redis_rate.NewLimiter(rdb)
	mw := RateLimitByIP(rl, slog.Default(), 10)
	handler := mw(okHandler)

	_, c, rec := newContext(http.MethodGet, "/test", "")

	err := handler(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
}

func Test_RateLimitByIP_Exceeded(t *testing.T) {
	rdb, cleanup := startRedis(t)
	defer cleanup()

	rl := redis_rate.NewLimiter(rdb)
	limit := 3
	mw := RateLimitByIP(rl, slog.Default(), limit)
	handler := mw(okHandler)

	for i := 0; i < limit; i++ {
		_, c, _ := newContext(http.MethodPost, "/login", "")
		err := handler(c)
		require.NoError(t, err)
	}

	_, c, rec := newContext(http.MethodPost, "/login", "")
	err := handler(c)

	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusTooManyRequests, httpErr.Code)
	require.Equal(t, http.StatusOK, rec.Code)
}

func Test_RateLimitByIP_RedisError(t *testing.T) {
	rdb, cleanup := startRedis(t)
	cleanup()

	rl := redis_rate.NewLimiter(rdb)
	mw := RateLimitByIP(rl, slog.Default(), 10)
	handler := mw(okHandler)

	_, c, _ := newContext(http.MethodGet, "/test", "")

	err := handler(c)

	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusInternalServerError, httpErr.Code)
}
