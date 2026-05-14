package handlers

import (
	"api_gateway/internal/components"
	"errors"
	"log"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/labstack/echo/v5"
)

func createProxy(targetUrl string, prefix string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetUrl)
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

func InitChatEndpoints(
	chat *echo.Group,
	targetUrl string,
	cl components.ChatLimits,
	rl func(limit int) echo.MiddlewareFunc,
	auth echo.MiddlewareFunc,
) {
	proxy, err := createProxy(targetUrl, "/api/v1/rooms")
	if err != nil {
		log.Fatal(err)
	}

	chat.Use(auth)

	chat.GET("", redirectTo(proxy), auth)
	chat.GET("/:roomId/users", redirectTo(proxy), auth)
	chat.GET("/:roomId/messages", redirectTo(proxy), auth, rl(cl.MessagesLimit))
	chat.POST("", redirectTo(proxy), auth, rl(cl.CreateRoomLimit))
	chat.POST("/:roomId/invite", redirectTo(proxy), auth, rl(cl.InviteLimit))
	chat.POST("/:roomId/leave", redirectTo(proxy), auth)
}
