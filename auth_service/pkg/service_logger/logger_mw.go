package service_logger

import (
	"errors"
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"
)

// LoggerMW returns an Echo middleware that logs every request with method, path,
// remote addr, user agent, request ID, status code, response size, and duration.
// 4xx errors are logged at Warn level, 5xx and handler errors at Error level.
func LoggerMW(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		const op = "middleware.logger"
		l := logger.With(slog.String("op", op))

		return func(c *echo.Context) error {
			req := c.Request()

			entry := l.With(
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("remote_addr", req.RemoteAddr),
				slog.String("user_agent", req.UserAgent()),
				slog.String("request_id", c.Request().Header.Get(echo.HeaderXRequestID)),
			)

			t1 := time.Now()

			err := next(c)

			res, unwrapErr := echo.UnwrapResponse(c.Response())
			if unwrapErr != nil {
				entry.Error("failed to unwrap response", slog.Any("error", unwrapErr))
				return err
			}

			if err != nil {
				var httpErr *echo.HTTPError
				if errors.As(err, &httpErr) && httpErr.Code < 500 {
					entry.Warn("bad request",
						slog.Int("status", httpErr.Code),
						slog.Any("message", httpErr.Message),
						slog.String("duration", time.Since(t1).String()),
					)
				} else {
					entry.Error("request failed",
						slog.Any("error", err),
						slog.Int("status", res.Status),
						slog.String("duration", time.Since(t1).String()),
					)
				}
				return err
			}

			entry.Info("request completed",
				slog.Int("status", res.Status),
				slog.Int64("bytes", res.Size),
				slog.String("duration", time.Since(t1).String()),
			)

			return err
		}
	}
}
