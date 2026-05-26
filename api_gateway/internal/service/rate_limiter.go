package rate_limiter

import (
	"log/slog"
	"net"
	"net/http"

	"github.com/go-redis/redis_rate/v10"
	"github.com/labstack/echo/v5"
)

func RateLimitByIP(limiter *redis_rate.Limiter, logger *slog.Logger, limitRate int) echo.MiddlewareFunc {
	limit := redis_rate.PerMinute(limitRate)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			l := logger.With(slog.String("mw", "rate_limiter"))

			ip := c.Request().Header.Get("X-Real-IP")
			if ip == "" {
				var err error
				ip, _, err = net.SplitHostPort(c.Request().RemoteAddr)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "error parsing ip")
				}
			}
			key := ip + ":" + c.Request().URL.Path

			res, err := limiter.Allow(c.Request().Context(), key, limit)
			if err != nil {
				l.Error("rate limiter redis error", slog.String("err", err.Error()))
				return echo.NewHTTPError(http.StatusInternalServerError, "error getting limits")
			}
			if res.Allowed == 0 {
				l.Warn("rate limit exceeded", slog.String("ip", ip))
				return echo.NewHTTPError(http.StatusTooManyRequests, "request limit exceeded")
			}

			return next(c)
		}
	}
}

func RateLimitByUser(limiter *redis_rate.Limiter, logger *slog.Logger, limitRate int) echo.MiddlewareFunc {
	limit := redis_rate.PerMinute(limitRate)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			l := logger.With(slog.String("mw", "rate_limiter"))

			//nolint:errcheck // userID sets in api_gateway/internal/middleware/auth.go
			userID := c.Get("userID").(string)
			key := userID + ":" + c.Path()

			res, err := limiter.Allow(c.Request().Context(), key, limit)
			if err != nil {
				l.Error("rate limiter redis error", slog.String("err", err.Error()))
				return echo.NewHTTPError(http.StatusInternalServerError, "error getting limits")
			}
			if res.Allowed == 0 {
				l.Warn("rate limit exceeded", slog.String("userID", userID))
				return echo.NewHTTPError(http.StatusTooManyRequests, "request limit exceeded")
			}

			return next(c)
		}
	}
}
