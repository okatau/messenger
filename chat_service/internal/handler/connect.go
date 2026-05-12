package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"chat_service/internal/domain"
	"chat_service/internal/service"
	"chat_service/pkg/token_manager"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
)

const wsCodeUnauthorized = 4001

func getUpgrader(whitelist []string) websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if slices.Contains(whitelist, origin) {
				return true
			}
			return false
		},
	}
}

func Connect(hub service.Hub, manager *token_manager.TokenManager, ctx context.Context, whitelist []string) echo.HandlerFunc {
	return func(c *echo.Context) error {
		upgrader := getUpgrader(whitelist)
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			return nil
		}

		var handshake struct {
			Token  string `json:"token"`
			RoomID string `json:"roomId"`
		}

		if err := json.Unmarshal(msg, &handshake); err != nil {
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseInvalidFramePayloadData, "invalid handshake data"))
			conn.Close()
			return nil
		}

		claims, err := manager.VerifyAccessToken(handshake.Token)
		if err != nil {
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(wsCodeUnauthorized, "unauthorized"))
			conn.Close()
			return nil
		}

		if err := hub.Connect(ctx, claims.Subject, conn); err != nil {
			code := websocket.CloseInternalServerErr
			msg := "internal server error"
			if errors.Is(err, domain.ErrUserNotFound) {
				code = websocket.CloseNormalClosure
				msg = "user not found"
			}
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(code, msg))
			conn.Close()
			return nil
		}

		return nil
	}
}
