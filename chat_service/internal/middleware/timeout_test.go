package middleware

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Timeout(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var deadline time.Time
		var hasDeadline bool

		next := func(c *echo.Context) error {
			deadline, hasDeadline = c.Request().Context().Deadline()
			return nil
		}

		_, c, _ := newContext(http.MethodGet, "/", "")
		before := time.Now()

		require.NoError(t, Timeout(time.Second)(next)(c))
		require.True(t, hasDeadline)
		assert.WithinDuration(t, before.Add(time.Second), deadline, 50*time.Millisecond)
	})

	t.Run("context cancelled after timeout", func(t *testing.T) {
		var ctxErr error

		next := func(c *echo.Context) error {
			ctx := c.Request().Context()
			select {
			case <-ctx.Done():
				ctxErr = ctx.Err()
			case <-time.After(2 * time.Second):
			}
			return nil
		}

		_, c, _ := newContext(http.MethodGet, "/", "")

		Timeout(50 * time.Millisecond)(next)(c)

		assert.ErrorIs(t, ctxErr, context.DeadlineExceeded)
	})
}
