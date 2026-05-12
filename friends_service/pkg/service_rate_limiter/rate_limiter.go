package service_rate_limiter

import (
	"log/slog"
	"net/http"

	"github.com/go-redis/redis_rate/v10"
	"github.com/labstack/echo/v5"
)

func RateLimitByIP(limiter *redis_rate.Limiter, logger *slog.Logger, limitRate int) echo.MiddlewareFunc {
	limit := redis_rate.PerMinute(limitRate)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			logger := logger.With(slog.String("mw", "rate_limiter"))

			ip := c.Request().RemoteAddr
			key := ip + ":" + c.Request().URL.Path

			res, err := limiter.Allow(c.Request().Context(), key, limit)
			if err != nil {
				logger.Error("rate limiter redis error", slog.String("err", err.Error()))
				return echo.NewHTTPError(http.StatusInternalServerError, "error getting limits")
			}
			if res.Allowed == 0 {
				logger.Warn("rate limit exceeded", slog.String("ip", ip))
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
			logger := logger.With(slog.String("mw", "rate_limiter"))

			ip := c.Request().RemoteAddr
			key := c.Get("userID").(string) + ":" + c.Request().URL.Path

			res, err := limiter.Allow(c.Request().Context(), key, limit)
			if err != nil {
				logger.Error("rate limiter redis error", slog.String("err", err.Error()))
				return echo.NewHTTPError(http.StatusInternalServerError, "error getting limits")
			}
			if res.Allowed == 0 {
				logger.Warn("rate limit exceeded", slog.String("ip", ip))
				return echo.NewHTTPError(http.StatusTooManyRequests, "request limit exceeded")
			}

			return next(c)
		}
	}
}
