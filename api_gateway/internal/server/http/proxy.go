package httpserver

import (
	"errors"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/labstack/echo/v5"
)

func NewProxy(targetURL, prefix string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, errors.New("error parsing url")
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.URL.Path = strings.TrimPrefix(pr.Out.URL.Path, prefix)
			pr.Out.URL.RawPath = strings.TrimPrefix(pr.Out.URL.RawPath, prefix)
			if pr.Out.URL.Path == "" {
				pr.Out.URL.Path = "/"
			}
		},
	}

	return proxy, nil
}

func RedirectTo(proxy *httputil.ReverseProxy) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if userID, ok := c.Get("userID").(string); ok && userID != "" {
			c.Request().Header.Set("X-User-ID", userID)
		}
		proxy.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
