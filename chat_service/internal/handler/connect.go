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
			return slices.Contains(whitelist, origin)
		},
	}
}

func Connect(ctx context.Context, hub service.Hub, manager *token_manager.TokenManager, whitelist []string) echo.HandlerFunc {
	return func(c *echo.Context) error {
		upgrader := getUpgrader(whitelist)
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			conn.Close() //nolint:errcheck  // connection is already in error state, close error is irrelevant
			return nil   //nolint:nilerr // connection read error is expected on client disconnect
		}

		var handshake struct {
			Token  string `json:"token"`
			RoomID string `json:"roomId"`
		}

		if err = json.Unmarshal(msg, &handshake); err != nil {
			closeWSConn(conn, websocket.CloseInvalidFramePayloadData, "invalid handshake data")
			return nil //nolint:nilerr // invalid handshake: close message sent to client, error is not useful to caller
		}

		claims, err := manager.VerifyAccessToken(handshake.Token)
		if err != nil {
			closeWSConn(conn, wsCodeUnauthorized, "unauthorized")
			return nil //nolint:nilerr // unauthorized: close message sent to client, error is not useful to caller
		}

		if err := hub.Connect(ctx, claims.Subject, conn); err != nil {
			code, msg := websocket.CloseInternalServerErr, "internal server error"
			if errors.Is(err, domain.ErrUserNotFound) {
				code, msg = websocket.CloseNormalClosure, "user not found"
			}
			closeWSConn(conn, code, msg)
			return nil // hub connect error handled: close message sent to client with appropriate code
		}

		return nil
	}
}

func closeWSConn(conn *websocket.Conn, code int, text string) {
	//nolint:errcheck // best-effort close notification, error not actionable
	conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, text))
	//nolint:errcheck // connection is being closed, error is not actionable
	conn.Close()
}
