package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	c := e.NewContext(req, rec)
	return e, c, rec
}

func Test_ExtractUserID(t *testing.T) {
	next := func(c *echo.Context) error {
		return c.JSON(http.StatusOK, nil)
	}

	t.Run("success", func(t *testing.T) {
		userID := uuid.NewString()
		_, c, _ := newContext(http.MethodGet, "/extract_id", "")

		c.Request().Header.Set("X-User-ID", userID)

		err := ExtractUserID()(next)(c)
		require.NoError(t, err)
	})

	t.Run("empty header", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/extract_id", "")

		err := ExtractUserID()(next)(c)
		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, echoErr.Code, http.StatusUnauthorized)
	})

	t.Run("invalid id", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/extract_id", "")

		c.Request().Header.Set("X-User-ID", "invalid-uuid")

		err := ExtractUserID()(next)(c)
		var echoErr *echo.HTTPError
		require.ErrorAs(t, err, &echoErr)
		assert.Equal(t, echoErr.Code, http.StatusUnauthorized)
	})
}
