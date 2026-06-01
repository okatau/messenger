package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"chat_service/internal/domain"
	"chat_service/internal/pubsub"
	"chat_service/internal/repository"
	sl "chat_service/pkg/service_logger"

	ws "github.com/gorilla/websocket"
)

const (
	MaxBufSize  = 100
	userSvcName = "chat.service.user"
)

type User interface {
	AddRoomSub(ctx context.Context, room string)
	RemoveRoomSub(room string) error

	Listen(ctx context.Context, wg *sync.WaitGroup)

	Stop()

	ID() string
	Name() string
}

type user struct {
	id   string
	name string
	conn *ws.Conn

	subs   map[string]func() // roomid => sub
	subsMu sync.Mutex
	ps     pubsub.PubSub

	outgoing chan *domain.Message

	doneCh    chan struct{}
	closeOnce sync.Once

	hub     Hub
	logger  *slog.Logger
	msgRepo repository.MessageRepository
}

func NewUser(
	id, name string,
	conn *ws.Conn,
	ps pubsub.PubSub,
	hub Hub,
	logger *slog.Logger,
	msgRepo repository.MessageRepository,
) User {
	return &user{
		id:   id,
		name: name,
		conn: conn,

		subs: make(map[string]func()),
		ps:   ps,

		outgoing: make(chan *domain.Message, MaxBufSize),

		doneCh: make(chan struct{}),

		hub:     hub,
		logger:  logger,
		msgRepo: msgRepo,
	}
}

func (u *user) AddRoomSub(ctx context.Context, roomID string) {
	l := u.loggerWith(".addroomsub")
	u.subsMu.Lock()
	if _, exists := u.subs[roomID]; exists {
		l.Info("room already exists", slog.String("userID", u.id), slog.String("roomID", roomID))
		u.subsMu.Unlock()
		return
	}

	msgCh, unsub := u.ps.Subscribe(ctx, channelKey(roomID))
	u.subs[roomID] = unsub
	u.subsMu.Unlock()

	go func() {
		for {
			select {
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				select {
				case u.outgoing <- msg:
				default:
					l.Warn("outgoing buffer full", slog.String("userID", u.id))
				}

			case <-u.doneCh:
				return
			}
		}
	}()
}

func (u *user) RemoveRoomSub(roomID string) error {
	u.subsMu.Lock()
	defer u.subsMu.Unlock()

	unsub, ok := u.subs[roomID]
	if !ok {
		return domain.ErrRoomNotFound
	}

	delete(u.subs, roomID)
	unsub()

	return nil
}

func (u *user) Listen(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(2)
	go func() { defer wg.Done(); u.listenWrite() }()
	go func() { defer wg.Done(); u.listenRead(ctx) }()
}

func (u *user) listenWrite() {
	l := u.loggerWith(".listenwrite")

	for {
		select {
		case <-u.doneCh:
			l.Info("user done", slog.String("userID", u.id))
			return

		case msg := <-u.outgoing:
			if err := u.conn.WriteJSON(msg); err != nil {
				u.closeOnce.Do(func() { close(u.doneCh) })
				l.Info("failed to write message to conn", slog.String("userID", u.id), sl.Err(err))
			}
		}
	}
}

func (u *user) listenRead(ctx context.Context) {
	l := u.loggerWith(".listenread")

	defer func() {
		if err := u.conn.Close(); err != nil {
			l.Error("error closing ws connection", sl.Err(err))
		}
		u.hub.Disconnect(u.id) //nolint:errcheck // no need to check error
	}()

	for {
		select {
		case <-u.doneCh:
			l.Info("user done", slog.String("userID", u.id))
			return

		case <-ctx.Done():
			l.Info("ctx done", slog.String("userID", u.id))
			return

		default:
			msg := domain.Message{}
			if err := u.conn.ReadJSON(&msg); err != nil {
				u.closeOnce.Do(func() { close(u.doneCh) })
				l.Info("failed to read message from conn", slog.String("userID", u.id), sl.Err(err))
				return
			}

			u.subsMu.Lock()
			_, subscribed := u.subs[msg.RoomID]
			u.subsMu.Unlock()

			if !subscribed {
				l.Info("not subscribed to room", slog.String("userID", u.id), slog.String("roomID", msg.RoomID))
				continue
			}

			msg.UserID = u.id
			msg.Username = u.name
			msg.Timestamp = time.Now().UTC()
			// TODO group adding msg to db and redis
			go func(m domain.Message) {
				if err := u.msgRepo.WriteMessage(ctx, &m); err != nil {
					l.Error("failed to write message to db", sl.Err(err))
				}
			}(msg)
			if err := u.ps.Publish(ctx, channelKey(msg.RoomID), &msg); err != nil {
				l.Error("failed to publish message to pub sub", sl.Err(err))
			}
		}
	}
}

func (u *user) ID() string { return u.id }

func (u *user) Name() string { return u.name }

func (u *user) Stop() {
	u.closeOnce.Do(func() { close(u.doneCh) })
}

func (u *user) loggerWith(fnName string) *slog.Logger {
	return u.logger.With("op", userSvcName+fnName)
}

func channelKey(roomID string) string {
	return "room:" + roomID
}
